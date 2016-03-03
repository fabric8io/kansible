package cmd

import (
	"github.com/spf13/cobra"

	"github.com/fabric8io/kansible/log"
)

var (
	// RootCmd is the root command for the whole program.
	RootCmd = &cobra.Command{
		Use:   "kansible",
		Short: "Orchestrate processes in the same way as you orchestrate Docker containers with Kubernetes",
		Long: `Kansible

Kansible orchestrates processes in the same way as you orchestrate Docker containers with Kubernetes.

Once you have created an Ansible playbook to install and configure your software you can use Kansible to create
a Kubernetes Replication Controller to run, scale and manage the processes providing a universal view in Kubernetes
of all your containers and processes along with common scaling, high availability, service discovery and load balancing.

More help is here: https://github.com/fabric8io/kansible/blob/master/README.md
`,
	}

	sshPort int
)

func init() {
	RootCmd.PersistentFlags().IntVar(&sshPort, "port", 22, "the port for the remote SSH connection")
	RootCmd.PersistentFlags().BoolVar(&log.IsDebugging, "debug", false, "enable verbose debugging output")
}
