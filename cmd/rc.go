package cmd

import (
	"os"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/log"
)

const (
	// MessageFailedToCreateKubernetesClient is the message to report if a kuberentes client cannot be created
	MessageFailedToCreateKubernetesClient = "Failed to create Kubernetes client. Maybe you need to run `oc login`?. Error: %s"
)

var (
	inventory string
	replicas  int
)

func init() {
	rcCmd.Flags().StringVar(&inventory, "inventory", "inventory", "the location of your Ansible inventory file")
	rcCmd.Flags().IntVar(&replicas, "replicas", -1, "specifies the number of replicas to create for the RC")

	RootCmd.AddCommand(rcCmd)
}

// RCCmd is the root command for the whole program.
var rcCmd = &cobra.Command{
	Use:   "rc <hosts>",
	Short: "Creates or updates the kansible ReplicationController for some hosts in an Ansible inventory",
	Long:  `This commmand will analyse the hosts in an Ansible inventory and creates or updates the ReplicationController for the kansible pods.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Die("Expected argument <hosts> for the name of the hosts in the ansible inventory file")
		}
		hosts := args[0]

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

		inventory = os.ExpandEnv(inventory)
		if inventory == "" {
			log.Die("Value for inventory flag is empty")
		}

		hostEntries, err := ansible.LoadHostEntries(inventory, hosts)
		if err != nil {
			log.Die("Cannot load host entries: %s", err)
		}
		log.Info("Found %d host entries in the Ansible inventory for %s", len(hostEntries), hosts)

		rcFile := "kubernetes/" + hosts + "/rc.yml"

		_, err = ansible.UpdateKansibleRC(hostEntries, hosts, f, kubeclient, ns, rcFile, replicas)
		if err != nil {
			log.Die("Failed to update Kansible RC: ", err)
		}
	},
}
