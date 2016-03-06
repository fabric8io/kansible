/*
 * Copyright 2016 Red Hat
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/k8s"
	"github.com/fabric8io/kansible/log"
	"github.com/fabric8io/kansible/ssh"
	"github.com/fabric8io/kansible/winrm"
)

var (
	rcName, passwordFlag, connection, bash string
)

func init() {
	podCmd.Flags().StringVar(&rcName, "rc", "$KANSIBLE_RC", "the name of the ReplicationController for the supervisors")
	podCmd.Flags().StringVar(&passwordFlag, "password", "$KANSIBLE_PASSWORD", "the password used for WinRM connections")
	podCmd.Flags().StringVar(&connection, "connection", "", "the Ansible connection type to use. Defaults to SSH unless 'winrm' is defined to use WinRM on Windows")
	podCmd.Flags().StringVar(&bash, "bash", "$KANSIBLE_BASH", "if specified a script is generated for running a bash like shell on the remote machine")

	RootCmd.AddCommand(podCmd)
}

// podCmd is the root command for the whole program.
var podCmd = &cobra.Command{
	Use:   "pod <hosts> [command]",
	Short: "Runs the kansible pod which owns a host from the Ansible inventory then runs a remote command on the host",
	Long:  `This commmand will pick an available host from the Ansible inventory, then run a remote command on that host.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Pod runs the kansible pod for a given group of hosts in an Ansible playbook
		// this grabs a specific host (using annotations on the RC) then runs a remote command
		// on that host binding stdin, stdout, stderr to the remote process
		if len(args) < 1 {
			log.Die("Expected arguments <hosts> [command]")
		}
		hosts := os.ExpandEnv(args[0])
		command := ""
		if len(args) > 1 {
			command = os.ExpandEnv(strings.Join(args[1:], " "))
		}

		f := cmdutil.NewFactory(clientConfig)
		if f == nil {
			log.Die("Failed to create Kubernetes client factory!")
		}
		kubeclient, err := f.Client()
		if err != nil || kubeclient == nil {
			log.Die(MessageFailedToCreateKubernetesClient, err)
		}
		ns := os.Getenv(ansible.EnvNamespace)
		if len(ns) == 0 {
			ns, _, _ = f.DefaultNamespace()
			if len(ns) == 0 {
				ns = "default"
			}
		}
		rcName = os.ExpandEnv(rcName)
		if rcName == "" {
			log.Die("RC name is required")
		}
		thisPodName, err := k8s.GetThisPodName()
		if err != nil {
			log.Die("Couldn't get pod name: %s", err)
		}

		hostEntry, rc, envVars, err := ansible.ChooseHostAndPrivateKey(thisPodName, hosts, kubeclient, ns, rcName)
		if err != nil {

			log.Die("Couldn't find host: %s", err)
		}
		host := hostEntry.Host
		user := hostEntry.User
		port := hostEntry.Port
		if len(port) == 0 {
			port = strconv.Itoa(sshPort)
		}

		connection := hostEntry.Connection
		if len(connection) == 0 {
			connection = os.ExpandEnv(connection)
		}

		runCommand := hostEntry.RunCommand
		if len(runCommand) != 0 {
			command = runCommand
		}

		commandEnvVars := []string{}
		if len(command) == 0 {
			if len(connection) > 0 {
				envVarName := ansible.EnvCommand + "_" + strings.ToUpper(connection)
				commandEnvVars = append(commandEnvVars, envVarName)
				command = os.Getenv(envVarName)
			}
		}
		commandEnvVars = append(commandEnvVars, ansible.EnvCommand)
		if len(command) == 0 {
			command = os.Getenv(ansible.EnvCommand)
		}
		if len(command) == 0 {
			plural := ""
			if len(commandEnvVars) > 1 {
				plural = "s"
			}
			log.Die("Could not find a command to execute from the environment variable%s: %s", plural, strings.Join(commandEnvVars, ", "))
		}

		bash := os.ExpandEnv(bash)
		if len(bash) > 0 {
			err = generateBashScript(bash, connection)
			if err != nil {
				log.Err("Failed to generate bash script at %s due to: %v", bash, err)
				return
			}
		}

		if connection == ansible.ConnectionWinRM {
			password := hostEntry.Password
			if len(password) == 0 {
				password = os.ExpandEnv(passwordFlag)
				if password == "" {
					log.Die("Cannot connect without a password")
				}
			}
			err = winrm.RemoteWinRmCommand(user, password, host, port, command, kubeclient, rc, hostEntry.Name)
		} else {
			privatekey := hostEntry.PrivateKey

			err = ssh.RemoteSSHCommand(user, privatekey, host, port, command, envVars)
		}
		if err != nil {
			log.Err("Failed: %v", err)
		}
	},
}

func generateBashScript(file string, connection string) error {
	shellCommand := "bash"
	if connection == ansible.ConnectionWinRM {
		shellCommand = "cmd"
	}
	text := `#!/bin/sh
echo "opening shell on remote machine..."
export ` + ansible.EnvIsBashShell + `=true
export ` + ansible.EnvPortForward + `=false
kansible pod appservers ` + shellCommand + "\n"
	return ioutil.WriteFile(file, []byte(text), 0555)
}
