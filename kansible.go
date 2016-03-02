package main

import (
	"errors"

	"github.com/codegangsta/cli"

	"github.com/fabric8io/kansible/cmds"
	"github.com/fabric8io/kansible/log"
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

Kansible orchestrates processes in the same way as you orchestrate Docker containers with Kubernetes.

Once you have created an Ansible playbook to install and configure your software you can use Kansible to create
a Kubernetes Replication Controller to run, scale and manage the processes providing a universal view in Kubernetes
of all your containers and processes along with common scaling, high availability, service discovery and load balancing.

More help is here: https://github.com/fabric8io/kansible/blob/master/README.md`
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
			Name:  "port",
			Value: "22",
			Usage: "The port for the remote SSH connection",
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
			Name:        "rc",
			Usage:       "Creates or updates the kansible ReplicationController for some hosts in an Ansible inventory.",
			Description: `This commmand will analyse the hosts in an Ansible inventory and creates or updates the ReplicationController for the kansible pods.`,
			ArgsUsage:   "[hosts] [command]",
			Action:      cmds.RC,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "inventory",
					Value: "inventory",
					Usage: "The location of your Ansible inventory file",
				},
				cli.StringFlag{
					Name:  "rc",
					Value: "rc.yml",
					Usage: "The YAML file of the ReplicationController for the supervisors",
				},
				cli.StringFlag{
					Name:  "replicas",
					Usage: "Specifies the number of replicas to create for the RC",
				},
			},
		},
		{
			Name:        "pod",
			Usage:       "Runs the kansible pod which owns a host from the Ansible inventory then runs a remote command on the host.",
			Description: `This commmand will pick an available host from the Ansible inventory, then run a remote command on that host.`,
			ArgsUsage:   "[hosts] [command]",
			Action:      cmds.Pod,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "rc",
					Value: "$KANSIBLE_RC",
					Usage: "The name of the ReplicationController for the supervisors",
				},
				cli.StringFlag{
					Name:  "password",
					Value: "$KANSIBLE_PASSWORD",
					Usage: "The password used for WinRM connections",
				},
				cli.StringFlag{
					Name:  "connection",
					Usage: "The Ansible connection type to use. Defaults to SSH unless 'winrm' is defined to use WinRM on Windows",
				},
				cli.StringFlag{
					Name:  "bash",
					Value: "$KANSIBLE_BASH",
					Usage: "If specified a script is generated for running a bash like shell on the remote machine",
				},
			},
		},
		{
			Name:        "kill",
			Usage:       "Kills any pending shells for this pod.",
			Description: `This commmand will find the shell thats associated with a pod and kill it.`,
			Action:      cmds.Kill,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "rc",
					Value: "$KANSIBLE_RC",
					Usage: "The name of the ReplicationController for the supervisors",
				},
			},
		},
		{
			Name:        "run",
			Usage:       "Runs a remote command on a given host to test out SSH / WinRM",
			Description: `This commmand will begin running the supervisor on an avaiable host.`,
			ArgsUsage:   "[string]",
			Action:      cmds.Run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "user",
					Value: "$KANSIBLE_USER",
					Usage: "The user to use on the remote connection",
				},
				cli.StringFlag{
					Name:  "privatekey",
					Value: "$KANSIBLE_PRIVATEKEY",
					Usage: "The private key used for SSH",
				},
				cli.StringFlag{
					Name:  "host",
					Value: "$KANSIBLE_HOST",
					Usage: "The host for the remote connection",
				},
				cli.StringFlag{
					Name:  "command",
					Value: "$KANSIBLE_COMMAND",
					Usage: "The remote command to invoke on the host",
				},
				cli.StringFlag{
					Name:  "password",
					Usage: "The password if using WinRM to execute the command",
				},
				cli.StringFlag{
					Name:  "connection",
					Usage: "The Ansible connection type to use. Defaults to SSH unless 'winrm' is defined to use WinRM on Windows",
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
