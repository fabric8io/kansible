package main

import (
	"errors"
	"fmt"
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

// version is the version of the app.
//
// This value is overwritten by the linker during build. The default version
// here is SemVer 2, but basically indicates that this was a one-off build
// and should not be trusted.
var version = "0.1.0-unstable"

func main() {
	app := cli.NewApp()
	app.Name = "kansible"
	app.Usage = `Kansible

This command supervises a remote process inside a Pod inside Kubernetes to make
it look and feel like legacy processes running outside of Kubernetes are really
running inside Docker inside Kubernetes.

`
	app.Version = version
	app.EnableBashCompletion = true
	app.After = func(c *cli.Context) error {
		if log.ErrorState {
			return errors.New("Exiting with errors")
		}

		return nil
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "port",
			Value:  "22",
			Usage:  "The port for the remote SSH connection",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable verbose debugging output",
		},
	}

	app.CommandNotFound = func(c *cli.Context, command string) {
		log.Err("No matching command '%s'", command)
		cli.ShowAppHelp(c)
		log.Die("")
	}

	app.Commands = []cli.Command{
		{
			Name:    "pod",
			Usage:   "Runs the supervisor pod for a single host in a set of hosts from an Ansible inventory.",
			Description: `This commmand will begin running the supervisor command on one host from the Ansible inventory.`,
			ArgsUsage: "[hosts] [command]",
			Action: runAnsiblePod,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "inventory",
					Value:  "inventory",
					Usage:  "The location of your Ansible inventory file",
				},
				cli.StringFlag{
					Name:   "rc",
					Value:  "$KANSIBLE_RC",
					Usage:  "The name of the ReplicationController for the supervisors",
				},
				cli.StringFlag{
					Name:   "password",
					Value:  "$KANSIBLE_PASSWORD",
					Usage:  "The password used for WinRM connections",
				},
				cli.StringFlag{
					Name:   "connection",
					Value:  "$KANSIBLE_CONNECTION",
					Usage:  "The Ansible connection type to use. Defaults to SSH unless 'winrm' is defined to use WinRM on Windows",
				},
				cli.StringFlag{
					Name:   "bash",
					Value:  "$KANSIBLE_BASH",
					Usage:  "If specified a script is generated for running a bash like shell on the remote machine",
				},
			},
		},
		{
			Name:    "rc",
			Usage:   "Applies ReplicationController for the supervisors for some hosts in an Ansible inventory.",
			Description: `This commmand will analyse the hosts in an Ansible inventory and creates or updates the ReplicationController for its supervisors.`,
			ArgsUsage: "[hosts] [command]",
			Action: applyAnsibleRC,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "inventory",
					Value:  "inventory",
					Usage:  "The location of your Ansible inventory file",
				},
				cli.StringFlag{
					Name:   "rc",
					Value:  "rc.yml",
					Usage:  "The YAML file of the ReplicationController for the supervisors",
				},
			},
		},
		{
			Name:    "run",
			Usage:   "Runs a supervisor command on a given host as a user without using Ansible.",
			Description: `This commmand will begin running the supervisor on an avaiable host.`,
			ArgsUsage: "[string]",
			Action: run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "user",
					Value:  "$KANSIBLE_USER",
					Usage:  "The user to use on the remote connection",
				},
				cli.StringFlag{
					Name:   "privatekey",
					Value:  "$KANSIBLE_PRIVATEKEY",
					Usage:  "The private key used for SSH",
				},
				cli.StringFlag{
					Name:   "host",
					Value:  "$KANSIBLE_HOST",
					Usage:  "The host for the remote connection",
				},
				cli.StringFlag{
					Name:   "command",
					Value:  "$KANSIBLE_COMMAND",
					Usage:  "The remote command to invoke on the host",
				},
				cli.StringFlag{
					Name:   "password",
					Usage:  "The password if using WinRM to execute the command",
				},
				cli.StringFlag{
					Name:   "connection",
					Value:  "$KANSIBLE_CONNECTION",
					Usage:  "The Ansible connection type to use. Defaults to SSH unless 'winrm' is defined to use WinRM on Windows",
				},
			},
		},
	}

	app.Before = func(c *cli.Context) error {
		log.IsDebugging = c.Bool("debug")
		return nil
	}

	app.RunAndExitOnError()
}


func osExpand(c *cli.Context, name string) string {
	flag := c.String(name)
	value := os.ExpandEnv(flag)
	log.Debug("flag %s is %s", name, value)
	return value
}

func osExpandAndVerify(c *cli.Context, name string) (string, error) {
	value := osExpand(c, name)
	if len(value) == 0 {
		return "", fmt.Errorf("No parameter supplied for: %s", name)
	}
	return value, nil
}

func osExpandAndVerifyGlobal(c *cli.Context, name string) (string, error) {
	flag := c.GlobalString(name)
	value := os.ExpandEnv(flag)
	if len(value) == 0 {
		return "", fmt.Errorf("No parameter supplied for: %s", name)
	}
	log.Debug("flag %s is %s", name, value)
	return value, nil
}

func fail(err error) {
	log.Die("Failed: %s", err)
}

func applyAnsibleRC(c *cli.Context) {
	args := c.Args()
	if len(args) < 1 {
		log.Die("Expected an argument!")
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

func runAnsiblePod(c *cli.Context) {
	args := c.Args()
	if len(args) < 2 {
		log.Die("Expected at least 2 arguments!")
	}
	hosts := args[0]
	command := strings.Join(args[1:], " ")

	log.Info("running command on a host from %s and command `%s`", hosts, command)

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

	inventory, err := osExpandAndVerify(c, "inventory")
	if err != nil {
		fail(err)
	}
	rcName, err := osExpandAndVerify(c, "rc")
	if err != nil {
		fail(err)
	}
	hostEntry, err := ansible.ChooseHostAndPrivateKey(inventory, hosts, kubeclient, ns, rcName)
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
		connection = c.String("connection")
	}

	bash := osExpand(c, "bash")
	if len(bash) > 0 {
		err = generateBashScript(bash, connection)
		if err != nil {
			log.Err("Failed to generate bash script at %s due to: %v", bash, err)
		}
	}

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
		err = ssh.RemoteSshCommand(user, privatekey, host, port, command)
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

func run(c *cli.Context) {
	log.Info("Running GoSupervise!")

	port, err := osExpandAndVerifyGlobal(c, "port")
	if err != nil {
		fail(err)
	}
	command, err := osExpandAndVerify(c, "command")
	if err != nil {
		fail(err)
	}
	host, err := osExpandAndVerify(c, "host")
	if err != nil {
		fail(err)
	}
	user, err := osExpandAndVerify(c, "user")
	if err != nil {
		fail(err)
	}
	connection := c.String("connection")
	if connection == ansible.ConnectionWinRM {
		password, err := osExpandAndVerify(c, "password")
		if err != nil {
			fail(err)
		}
		err = winrm.RemoteWinRmCommand(user, password, host, port, command)
	} else {
		privatekey, err := osExpandAndVerify(c, "privatekey")
		if err != nil {
			fail(err)
		}
		err = ssh.RemoteSshCommand(user, privatekey, host, port, command)
	}
	if err != nil {
		log.Err("Failed: %v", err)
	}
}
