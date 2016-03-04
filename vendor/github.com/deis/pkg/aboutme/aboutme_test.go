package aboutme

import (
	"net"
	"os"
	"strings"
	"testing"

	"k8s.io/kubernetes/pkg/client/unversioned"
)

func TestFromEnv(t *testing.T) {
	if _, err := unversioned.InClusterConfig(); err != nil {
		t.Skip("This can only be run inside Kubernetes. Skipping.")
	}

	me, err := FromEnv()
	if err != nil {
		t.Errorf("Could not get an environment: %s", err)
	}
	if len(me.Name) == 0 {
		t.Error("Could not get a pod name.")
	}
}

func TestNamespaceFromEnv(t *testing.T) {
	if ns := NamespaceFromEnv(); ns != DefaultNamespace {
		t.Errorf("You did something stupid.")
	}

	os.Setenv(EnvNamespace, "slurm")
	if ns := NamespaceFromEnv(); ns != "slurm" {
		t.Errorf("Expected slurm, got %q", ns)
	}
}

func TestNameFromEnv(t *testing.T) {
	os.Setenv("HOSTNAME", "example")
	if n := NameFromEnv(); n != "example" {
		t.Errorf("Expected example, got %q", n)
	}
	os.Setenv(EnvName, "slumber")
	if n := NameFromEnv(); n != "slumber" {
		t.Errorf("Expected slumber, got %s", n)
	}
}

func TestShuntEnv(t *testing.T) {
	e := &Me{
		Annotations: map[string]string{"a": "a"},
		Labels:      map[string]string{"b": "b"},
		Name:        "c",
	}

	e.ShuntEnv()

	if "a" != os.Getenv("MY_ANNOTATION_A") {
		t.Errorf("Expected 'a', got '%s'", os.Getenv("MY_ANNOTATION_A"))
	}
	if "b" != os.Getenv("MY_LABEL_B") {
		t.Errorf("Expected 'b', got '%s'", os.Getenv("MY_LABEL_B"))
	}

	if "c" != os.Getenv("MY_NAME") {
		t.Errorf("Expected 'c', got '%s'", os.Getenv("MY_NAME"))
	}
}

func TestMyIPLocal(t *testing.T) {
	// This version does not require running inside of k8s.
	ip, _ := MyIP()
	if len(ip) == 0 {
		t.Error("Expected a string, got empty")
	}
	octets := strings.Split(ip, ".")
	if len(octets) != 4 {
		t.Errorf("Expected 4 octets, got %d", len(octets))
	}
}

func TestByInterfaceEth0(t *testing.T) {
	if _, err := net.InterfaceByName("eth0"); err != nil {
		t.Skip("Host operating system does not have an eth0 device to test.")
	}

	ip, err := IPByInterface("eth0")
	if err != nil {
		t.Errorf("Failed to get eth0: %s", err)
	}
	if len(ip) == 0 {
		t.Error("IP address is empty.")
	}

	if ip == "0.0.0.0" {
		t.Error("Got generic IP instead of real one.")
	}

}

func TestByInterfaceEn0(t *testing.T) {
	// This works on most Macs.
	if _, err := net.InterfaceByName("en0"); err != nil {
		t.Skip("Host operating system does not have an en0 device to test.")
	}

	ip, err := IPByInterface("en0")
	if err != nil {
		t.Errorf("Failed to get en0: %s", err)
	}
	if len(ip) == 0 {
		t.Error("IP address is empty.")
	}

	if ip == "0.0.0.0" {
		t.Error("Got generic IP instead of real one.")
	}

}

func TestMyIP(t *testing.T) {
	if _, err := net.InterfaceByName("eth0"); err != nil {
		t.Skip("Host operating system does not have an eth0 device to test.")
	}

	ip, err := MyIP()
	if err != nil {
		t.Errorf("Could not get IP address: %s", err)
	}

	if len(ip) == 0 {
		t.Errorf("Expected a valid IP address. Got nuthin.")
	}
}
