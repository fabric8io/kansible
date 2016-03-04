// Package k8s provides Kubernetes client access.
package k8s

import (
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

// PodClient is a Kubernetes API client that lives inside of a Pod. Pods
// uniformly declare environment variables that we can use to establish a
// client connection.
//
// For a generic client that can read Kubectl configs, see LocalClient.
func PodClient() (*unversioned.Client, error) {
	return unversioned.NewInCluster()
}

// LocalClient gets a Kubernetes client from the local environment.
//
// It does a lot of configuration file loading. Use it for cases where you
// have a kubectl client configured.
//
// For cases where you know exactly where the configuration information is,
// you should use Client.
//
// If you are constructing an interactive client, you may also want to look
// at the Kubernetes interactive client configuration.
func LocalClient() (*unversioned.Client, error) {
	// Please, if you find these poor Java developers help them find their
	// way out of the deep dark forests of Go, and back to the happy
	// halls of IntelliJ.
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg := &clientcmd.ConfigOverrides{}
	kcfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, cfg)
	ccfg, err := kcfg.ClientConfig()
	if err != nil {
		return nil, err
	}
	return unversioned.New(ccfg)
}
