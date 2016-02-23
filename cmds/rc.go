package cmds

import (
	"fmt"
	"strconv"

	"github.com/codegangsta/cli"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/log"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
// MessageFailedToCreateKubernetesClient is the message to report if a kuberentes client cannot be created
	MessageFailedToCreateKubernetesClient = "Failed to create Kubernetes client. Maybe you need to run `oc login`?. Error: %s"
)

// RC creates or updates the kansible ReplicationController for some hosts in an Ansible inventory
func RC(c *cli.Context) {
	args := c.Args()
	if len(args) < 1 {
		log.Die("Expected argument [hosts] for the name of the hosts in the ansible inventory file")
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

	inventory, err := osExpandAndVerify(c, "inventory")
	if err != nil {
		fail(err)
	}

	hostEntries, err := ansible.LoadHostEntries(inventory, hosts)
	if err != nil {
		fail(err)
	}
	log.Info("Found %d host entries in the Ansible inventory for %s", len(hostEntries), hosts)

	rcFile := "kubernetes/" + hosts + "/rc.yml"

	replicas := -1
	replicaText := c.String("replicas")
	if len(replicaText) > 0 {
		replicas, err = strconv.Atoi(replicaText)
		if err != nil {
			fail(fmt.Errorf("Failed to parse replicas text `%s`. Error: %s", replicaText, err))
		}
	}

	_, err = ansible.UpdateKansibleRC(hostEntries, hosts, kubeclient, ns, rcFile, replicas)
	if err != nil {
		fail(err)
	}
}

