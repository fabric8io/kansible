package cmds

import (
	"fmt"

	"github.com/codegangsta/cli"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/k8s"
	"github.com/fabric8io/kansible/log"
	"github.com/fabric8io/kansible/winrm"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// Kill kills the pending windows shell of the current pod if its still running
func Kill(c *cli.Context) {
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
		fail(err)
	}

	pod, err := kubeclient.Pods(ns).Get(thisPodName)
	if err != nil {
		fail(err)
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
	rcName, err := osExpandAndVerify(c, "rc")
	if err != nil {
		fail(err)
	}
	rc, err := kubeclient.ReplicationControllers(ns).Get(rcName)
	if err != nil {
		fail(err)
	}
	if rc == nil {
		fail(fmt.Errorf("No ReplicationController found for name %s", rcName))
	}
	metadata := &rc.ObjectMeta
	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}
	rcAnnotations := metadata.Annotations

	hostsText := rcAnnotations[ansible.HostInventoryAnnotation]
	if len(hostsText) == 0 {
		fail(fmt.Errorf("Could not find annotation %s on ReplicationController %s", ansible.HostInventoryAnnotation, rcName))
	}
	shellID := rcAnnotations[ansible.WinRMShellAnnotationPrefix+hostName]
	if len(shellID) == 0 {
		log.Info("No annotation `%s` available on pod %s", ansible.WinRMShellAnnotationPrefix, thisPodName)
		return
	}

	hostEntries, err := ansible.LoadHostEntriesFromText(hostsText)
	if err != nil {
		fail(err)
	}
	log.Info("Found %d host entries", len(hostEntries))

	hostEntry := ansible.GetHostEntryByName(hostEntries, hostName)
	if hostEntry == nil {
		fail(fmt.Errorf("Could not find a HostEntry called `%s` from %d host entries", hostName, len(hostEntries)))
	}

	err = winrm.CloseShell(hostEntry.User, hostEntry.Password, hostEntry.Host, hostEntry.Port, shellID)
	if err != nil {
		fail(err)
	}
	log.Info("Shell %s has been closed", shellID)
}
