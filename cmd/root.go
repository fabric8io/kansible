package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"

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

	clientConfig clientcmd.ClientConfig
)

func init() {
	RootCmd.PersistentFlags().IntVar(&sshPort, "port", 22, "the port for the remote SSH connection")
	RootCmd.PersistentFlags().BoolVar(&log.IsDebugging, "debug", false, "enable verbose debugging output")

	clientConfig = defaultClientConfig(RootCmd.PersistentFlags())
}

func defaultClientConfig(flags *pflag.FlagSet) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	flags.StringVar(&loadingRules.ExplicitPath, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests.")

	overrides := &clientcmd.ConfigOverrides{}
	flagNames := clientcmd.RecommendedConfigOverrideFlags("")

	clientcmd.BindOverrideFlags(overrides, flags, flagNames)
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, overrides, os.Stdin)

	return clientConfig
}
