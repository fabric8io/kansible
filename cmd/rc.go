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

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/log"
	"github.com/spf13/cobra"
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

		f := cmdutil.NewFactory(clientConfig)
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
			log.Die("Failed to update Kansible RC: %s", err)
		}
	},
}
