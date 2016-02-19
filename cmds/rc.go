package cmds


import (
	"github.com/codegangsta/cli"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/log"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)


func RC(c *cli.Context) {
	args := c.Args()
	if len(args) < 1 {
		log.Die("Expected argument [hosts] for the name of the hosts in the ansible inventory file")
	}
	hosts := args[0]

	f := cmdutil.NewFactory(nil)
	if f == nil {
		log.Die("Failed to create Kuberentes client factory!")
	}
	kubeclient, _ := f.Client()
	if kubeclient == nil {
		log.Die("Failed to create Kuberentes client!")
	}
	ns, _, _ := f.DefaultNamespace()
	if len(ns) == 0 {
		ns = "default"
	}

	rcName, err := osExpandAndVerify(c, "rc")
	if err != nil {
		fail(err)
	}

	inventory, err := osExpandAndVerify(c, "inventory")
	if err != nil {
		fail(err)
	}
	_, err = ansible.UpdateAnsibleRC(inventory, hosts, kubeclient, ns, rcName)
	if err != nil {
		fail(err)
	}
}

