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
	"os"
	"strconv"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/log"
	"github.com/fabric8io/kansible/ssh"
	"github.com/fabric8io/kansible/winrm"
	"github.com/spf13/cobra"
)

// 		cli.StringFlag{
// 			Name:  "connection",
// 			Usage: "The Ansible connection type to use. Defaults to SSH unless 'winrm' is defined to use WinRM on Windows",
// 		},
// 	},
// },

var (
	user, password, host, command, privatekey string
)

func init() {
	runCmd.Flags().StringVar(&user, "user", "${KANSIBLE_USER}", "the user to use on the remote connection")
	runCmd.Flags().StringVar(&privatekey, "privatekey", "${KANSIBLE_PRIVATEKEY}", "the private key used for SSH")
	runCmd.Flags().StringVar(&host, "host", "${KANSIBLE_HOST}", "the host for the remote connection")
	runCmd.Flags().StringVar(&command, "command", "${KANSIBLE_COMMAND}", "the remote command to invoke on the host")
	runCmd.Flags().StringVar(&password, "password", "", "the password if using WinRM to execute the command")
	runCmd.Flags().StringVar(&connection, "connection", "", "the Ansible connection type to use. Defaults to SSH unless 'winrm' is defined to use WinRM on Windows")

	RootCmd.AddCommand(runCmd)
}

// runCmd runs a remote command on a given host to test out SSH / WinRM
var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Runs a remote command on a given host to test out SSH / WinRM",
	Long:  `This commmand will begin running the supervisor on an avaiable host.`,
	Run: func(cmd *cobra.Command, args []string) {
		command = os.ExpandEnv(command)
		if command == "" {
			log.Die("Command is required")
		}
		host = os.ExpandEnv(host)
		if host == "" {
			log.Die("Host is required")
		}
		user = os.ExpandEnv(user)
		if user == "" {
			log.Die("User is required")
		}
		if connection == ansible.ConnectionWinRM {
			password = os.ExpandEnv(password)
			if password == "" {
				log.Die("Password is required")
			}
			err := winrm.RemoteWinRmCommand(user, password, host, strconv.Itoa(sshPort), command, nil, nil, "")
			if err != nil {
				log.Err("Failed: %v", err)
			}
		} else {
			privatekey = os.ExpandEnv(privatekey)
			if privatekey == "" {
				log.Die("Private key is required")
			}
			err := ssh.RemoteSSHCommand(user, privatekey, host, strconv.Itoa(sshPort), command, nil)
			if err != nil {
				log.Err("Failed: %v", err)
			}
		}
	},
}
