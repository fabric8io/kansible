package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/codegangsta/cli"

	"github.com/fabric8io/gosupervise/ansible"
	"github.com/fabric8io/gosupervise/log"
	"github.com/fabric8io/gosupervise/k8s"

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
	app.Name = "gosupervise"
	app.Usage = `Go Supervise

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
					Value:  "rc.yml",
					Usage:  "The YAML file of the ReplicationController for the supervisors",
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
					Value:  "$GOSUPERVISE_USER",
					Usage:  "The user to use on the remote SSH connection",
				},
				cli.StringFlag{
					Name:   "privatekey",
					Value:  "$GOSUPERVISE_PRIVATEKEY",
					Usage:  "The private key used for SSH",
				},
				cli.StringFlag{
					Name:   "host",
					Value:  "$GOSUPERVISE_HOST",
					Usage:  "The host for the remote SSH connection",
				},
				cli.StringFlag{
					Name:   "command",
					Value:  "$GOSUPERVISE_COMAND",
					Usage:  "The remote command to invoke over SSH",
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


func osExpandAndVerify(c *cli.Context, name string) (string, error) {
	flag := c.String(name)
	value := os.ExpandEnv(flag)
	if len(value) == 0 {
		return "", fmt.Errorf("No parameter supplied for: %s", name)
	}
	log.Debug("flag %s is %s", name, value)
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

func runAnsiblePod(c *cli.Context) {
	args := c.Args()
	if len(args) < 2 {
		log.Die("Expected at least 2 arguments!")
	}
	hosts := args[0]
	command := strings.Join(args[1:], " ")

	log.Info("running command on a host from %s and command `%s`", hosts, command)

	f := cmdutil.NewFactory(nil)
	kubeclient, _ := f.Client()
	ns, _, _ := f.DefaultNamespace()

	rcFile, err := osExpandAndVerify(c, "rc")
	if err != nil {
		fail(err)
	}

	port, err := osExpandAndVerifyGlobal(c, "port")
	if err != nil {
		fail(err)
	}
	inventory, err := osExpandAndVerify(c, "inventory")
	if err != nil {
		fail(err)
	}
	rc, err := k8s.ReadReplicationControllerFromFile(rcFile)
	if err != nil {
		fail(err)
	}
	rcName := rc.ObjectMeta.Name
	hostEntry, err := ansible.ChooseHostAndPrivateKey(inventory, hosts, kubeclient, ns, rcName)
	if err != nil {
		fail(err)
	}
	host := hostEntry.Host
	privatekey := hostEntry.PrivateKey
	user := hostEntry.User
	hostPort := host + ":" + port
	err = remoteSshCommand(user, privatekey, hostPort, command)
	if err != nil {
		log.Err("Failed: %v", err)
	}
}

func applyAnsibleRC(c *cli.Context) {
	args := c.Args()
	if len(args) < 1 {
		log.Die("Expected an argument!")
	}
	hosts := args[0]

	f := cmdutil.NewFactory(nil)
	kubeclient, _ := f.Client()
	ns, _, _ := f.DefaultNamespace()

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
	privatekey, err := osExpandAndVerify(c, "privatekey")
	if err != nil {
		fail(err)
	}
	user, err := osExpandAndVerify(c, "user")
	if err != nil {
		fail(err)
	}
	hostPort := host + ":" + port
	err = remoteSshCommand(user, privatekey, hostPort, command)
	if err != nil {
		log.Err("Failed: %v", err)
	}
}

func remoteSshCommand(user string, privateKey string, hostPort string, cmd string) error {
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(privateKey),
		},
	}
	if sshConfig == nil {
		log.Info("Whoah!")
	}
	connection, err := ssh.Dial("tcp", hostPort, sshConfig)
	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}
	session, err := connection.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create session: %s", err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		// ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return fmt.Errorf("Request for pseudo terminal failed: %s", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdin for session: %v", err)
	}
	go io.Copy(stdin, os.Stdin)

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for session: %v", err)
	}
	go io.Copy(os.Stdout, stdout)

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stderr for session: %v", err)
	}
	go io.Copy(os.Stderr, stderr)

	log.Info("Running command %s", cmd)
	err = session.Run(cmd)
	if err != nil {
		return fmt.Errorf("Failed to run command: " + cmd + ": %v", err)
	}
	return nil
}

func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

