package cmd

import (
	"os"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/k8s"
	"github.com/fabric8io/kansible/log"
	"github.com/fabric8io/kansible/winrm"
)

func init() {
	killCmd.Flags().StringVar(&rcName, "rc", "$KANSIBLE_RC", "the name of the ReplicationController for the supervisors")

	RootCmd.AddCommand(killCmd)
}

// killCmd kills the pending windows shell of the current pod if its still running
var killCmd = &cobra.Command{
	Use:   "kill <hosts> [command]",
	Short: "Kills any pending shells for this pod.",
	Long:  `This commmand will find the shell thats associated with a pod and kill it.`,
	Run: func(cmd *cobra.Command, args []string) {
		f := cmdutil.NewFactory(nil)
		if f == nil {
			log.Die("Failed to create Kubernetes client factory!")
		}
		kubeclient, err := f.Client()
		if err != nil || kubeclient == nil {
			log.Die(MessageFailedToCreateKubernetesClient, err)
		}
		ns, _, _ := f.DefaultNamespace()
		if len(ns) == 0 {
			ns = "default"
		}
		thisPodName, err := k8s.GetThisPodName()
		if err != nil {
			log.Die("Failed to get this pod name: %s", err)
		}

		pod, err := kubeclient.Pods(ns).Get(thisPodName)
		if err != nil {
			log.Die("Failed to get pod from API server: %s", err)
		}

		annotations := pod.ObjectMeta.Annotations
		if annotations == nil {
			log.Die("No annotations available on pod %s", thisPodName)
		}
		hostName := annotations[ansible.HostNameAnnotation]
		if len(hostName) == 0 {
			log.Info("No annotation `%s` available on pod %s", ansible.HostNameAnnotation, thisPodName)
			return
		}

		// now lets load the connection details from the RC annotations
		rcName = os.ExpandEnv(rcName)
		if rcName == "" {
			log.Die("Replication controller name is required")
		}
		rc, err := kubeclient.ReplicationControllers(ns).Get(rcName)
		if err != nil {
			log.Die("Failed to get replication controller from API server: %s", err)
		}
		if rc == nil {
			log.Die("No ReplicationController found for name %s", rcName)
		}
		metadata := &rc.ObjectMeta
		if metadata.Annotations == nil {
			metadata.Annotations = make(map[string]string)
		}
		rcAnnotations := metadata.Annotations

		hostsText := rcAnnotations[ansible.HostInventoryAnnotation]
		if len(hostsText) == 0 {
			log.Die("Could not find annotation %s on ReplicationController %s", ansible.HostInventoryAnnotation, rcName)
		}
		shellID := rcAnnotations[ansible.WinRMShellAnnotationPrefix+hostName]
		if len(shellID) == 0 {
			log.Info("No annotation `%s` available on pod %s", ansible.WinRMShellAnnotationPrefix, thisPodName)
			return
		}

		hostEntries, err := ansible.LoadHostEntriesFromText(hostsText)
		if err != nil {
			log.Die("Failed to load hosts: %s", err)
		}
		log.Info("Found %d host entries", len(hostEntries))

		hostEntry := ansible.GetHostEntryByName(hostEntries, hostName)
		if hostEntry == nil {
			log.Die("Could not find a HostEntry called `%s` from %d host entries", hostName, len(hostEntries))
		}

		err = winrm.CloseShell(hostEntry.User, hostEntry.Password, hostEntry.Host, hostEntry.Port, shellID)
		if err != nil {
			log.Die("Failed to close shell: %s", err)
		}
		log.Info("Shell %s has been closed", shellID)
	},
}
