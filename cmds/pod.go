package cmds

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/codegangsta/cli"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/log"
	"github.com/fabric8io/kansible/ssh"
	"github.com/fabric8io/kansible/winrm"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// Pod runs the kansible pod for a given group of hosts in an Ansible playbook
// this grabs a specific host (using annotations on the RC) then runs a remote command
// on that host binding stdin, stdout, stderr to the remote process
func Pod(c *cli.Context) {
	args := c.Args()
	if len(args) < 2 {
		log.Die("Expected arguments [hosts] [command]")
	}
	hosts := os.ExpandEnv(args[0])
	command := os.ExpandEnv(strings.Join(args[1:], " "))

	f := cmdutil.NewFactory(nil)
	if f == nil {
		log.Die("Failed to create Kubernetes client factory!")
	}
	kubeclient, _ := f.Client()
	if kubeclient == nil {
		log.Die("Failed to create Kubernetes client!")
	}
	ns, _, _ := f.DefaultNamespace()
	if len(ns) == 0 {
		ns = "default"
	}

	inventory, err := osExpandAndVerify(c, "inventory")
	if err != nil {
		fail(err)
	}
	rcName, err := osExpandAndVerify(c, "rc")
	if err != nil {
		fail(err)
	}
	envVars := make(map[string]string)
	hostEntry, err := ansible.ChooseHostAndPrivateKey(inventory, hosts, kubeclient, ns, rcName, envVars)
	if err != nil {
		fail(err)
	}
	host := hostEntry.Host
	user := hostEntry.User
	port := hostEntry.Port
	if len(port) == 0 {
		port, err = osExpandAndVerifyGlobal(c, "port")
	}
	if err != nil {
		fail(err)
	}

	connection := hostEntry.Connection
	if len(connection) == 0 {
		connection = osExpand(c, "connection")
	}

	runCommand := hostEntry.RunCommand
	if len(runCommand) != 0 {
		command = runCommand
	}

	log.Info("running command on a host from %s and command `%s`", hosts, command)
	bash := osExpand(c, "bash")
	if len(bash) > 0 {
		err = generateBashScript(bash, connection)
		if err != nil {
			log.Err("Failed to generate bash script at %s due to: %v", bash, err)
		}
	}

	log.Info("using connection %s", connection)
	if connection == ansible.ConnectionWinRM {
		log.Info("Using WinRM to connect to the hosts %s", hosts)
		password := hostEntry.Password
		if len(password) == 0 {
			password, err = osExpandAndVerify(c, "password")
			if err != nil {
				fail(err)
			}
		}
		err = winrm.RemoteWinRmCommand(user, password, host, port, command)
	} else {
		privatekey := hostEntry.PrivateKey

		err = ssh.RemoteSSHCommand(user, privatekey, host, port, command, envVars)
	}
	if err != nil {
		log.Err("Failed: %v", err)
	}
}

func generateBashScript(file string, connection string) error {
	shellCommand := "bash"
	if connection == ansible.ConnectionWinRM {
		shellCommand = "PowerShell"
	}
	text :=  "#!/bin/sh\n" + "echo opening shell on remote machine...\n" + "kansible pod appservers " + shellCommand + "\n";
	return ioutil.WriteFile(file, []byte(text), 0555)
}
