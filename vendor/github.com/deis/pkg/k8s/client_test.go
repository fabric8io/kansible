package k8s

import (
	"testing"

	"k8s.io/kubernetes/pkg/client/unversioned"
)

func TestLocalClient(t *testing.T) {
	c, err := LocalClient()
	if err != nil {
		t.Errorf("Failed to get a client: %s", err)
	}
	if c == nil {
		t.Errorf("Could not get a kube client, and no reason was given.")
	}
}

func TestPodClient(t *testing.T) {
	// A pod can't really be mocked efficiently without major filesystem
	// manipulation. So we're testing fully only when this is running inside of
	// a k8s pod.
	if _, err := unversioned.InClusterConfig(); err != nil {
		t.Skip("This can only be run inside Kubernetes. Skipping.")
	}

	c, err := PodClient()
	if err != nil {
		t.Errorf("Error constructing client: %s", err)
	}

	if _, err := c.ServerVersion(); err != nil {
		t.Errorf("Failed to connect to given server: %s", err)
	}
}
