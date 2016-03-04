// Package aboutme provides information to a pod about itself.
//
// Typical usage is to let the Pod auto-detect information about itself:
//
//	my, err := aboutme.FromEnv()
//  if err != nil {
// 		// Error connecting to tke k8s API server
// 	}
//
// 	fmt.Printf("My Pod Name is %s", my.Name)
package aboutme

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/deis/pkg/k8s"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
)

// DefaultNamespace is the Kubernetes default namespace.
const DefaultNamespace = "default"

var (
	// EnvNamespace is the environment variable this looks for to get the namespace.
	//
	// You can set this via the Downward API.
	EnvNamespace = "POD_NAMESPACE"
	// EnvName is the environment variable this looks for to get the pod name.
	//
	// You can set this via the Downward API.
	EnvName = "POD_NAME"
)

type Me struct {
	ApiServer, Name                      string
	IP, NodeIP, Namespace, SelfLink, UID string
	Labels                               map[string]string
	Annotations                          map[string]string

	c *unversioned.Client
}

// FromEnv uses the environment to create a new Me.
//
// To use this, a client MUST be running inside of a Pod environment. It uses
// a combination of environment variables and file paths to determine
// information about the cluster.
func FromEnv() (*Me, error) {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	name := NameFromEnv()

	// FIXME: Better way? Probably scanning secrets for
	// an SSL cert would help?
	proto := "https"

	url := proto + "://" + host + ":" + port

	me := &Me{
		ApiServer: url,
		Name:      name,
		Namespace: NamespaceFromEnv(),
	}

	client, err := k8s.PodClient()
	if err != nil {
		return me, err
	}
	me.c = client

	if err := me.init(); err != nil {
		return me, err
	}

	return me, nil
}

// Client returns an initialized Kubernetes API client.
func (me *Me) Client() *unversioned.Client {
	return me.c
}

// NameFromEnv gets the pod name from either the Downward API or the hostname.
func NameFromEnv() string {
	n := os.Getenv(EnvName)
	if n == "" {
		return os.Getenv("HOSTNAME")
	}
	return n
}

//NamespaceFromEnv attempts to get the namespace from the downward API.
//
// If EnvNamespace is not set, or if the name is not recovered from the
// environment, then the DefaultNamespace is used.
func NamespaceFromEnv() string {
	ns := os.Getenv(EnvNamespace)
	if ns == "" {
		return DefaultNamespace
	}
	return ns
}

// ShuntEnv puts the Me object into the environment.
//
// The properties of Me are placed into the environment according to the
// following rules:
//
// 	- In general, all variables are prefaced with MY_ (MY_IP, MY_NAMESPACE)
// 	- Labels become MY_LABEL_[NAME]=[value]
// 	- Annotations become MY_ANNOTATION_[NAME] = [value]
func (me *Me) ShuntEnv() {
	env := map[string]string{
		"MY_APISERVER": me.ApiServer,
		"MY_NAME":      me.Name,
		"MY_IP":        me.IP,
		"MY_NODEIP":    me.NodeIP,
		"MY_NAMESPACE": me.Namespace,
		"MY_SELFLINK":  me.SelfLink,
		"MY_UID":       me.UID,
	}
	for k, v := range env {
		os.Setenv(k, v)
	}
	var name string
	for k, v := range me.Labels {
		name = "MY_LABEL_" + strings.ToUpper(k)
		os.Setenv(name, v)
	}
	for k, v := range me.Annotations {
		name = "MY_ANNOTATION_" + strings.ToUpper(k)
		os.Setenv(name, v)
	}
}

func (me *Me) init() error {
	p, n, err := me.loadPod()
	if err != nil {
		return err
	}

	me.Namespace = n
	me.IP = p.Status.PodIP
	me.NodeIP = p.Status.HostIP
	me.SelfLink = p.SelfLink
	me.UID = string(p.UID)
	me.Labels = p.Labels
	me.Annotations = me.Annotations

	// FIXME: It appears that sometimes the k8s API server does not set the
	// PodIP, even though the pod is issued an IP. We need to figure out why,
	// and if this is an expected case. In the meantime, we get the IP by
	// scanning interfaces.
	if strings.TrimSpace(me.IP) == "" {
		// We swallow the error, letting me.IP set the interface address to
		// 0.0.0.0.
		me.IP, _ = MyIP()
	}

	return nil
}

// loadPod loads a pod using the downward API.
func (me *Me) loadPod() (*api.Pod, string, error) {
	ns := NamespaceFromEnv()
	p, err := me.c.Pods(ns).Get(me.Name)
	return p, ns, err
}

// findPodInNamespaces searches relevant namespaces for this pod.
//
// It returns a PodInterface for working with the pod, a namespace name as a
// string, and an error if something goes wrong.
//
// The selector must be a label selector.
func (me *Me) findPodInNamespaces(selector string) (*api.Pod, string, error) {
	// Get the deis namespace. If it does not exist, get the default namespce.
	s, err := labels.Parse(selector)
	if err == nil {
		ns, err := me.c.Namespaces().List(s, nil)
		if err != nil {
			return nil, "default", err
		}
		for _, n := range ns.Items {
			p, err := me.c.Pods(n.Name).Get(me.Name)

			// If there is no error, we got a matching pod.
			if err == nil {
				return p, n.Name, nil
			}
		}
	}

	// If we get here, it's really the last ditch.
	p, err := me.c.Pods("default").Get(me.Name)
	return p, "default", err
}

// MyIP examines the local interfaces and guesses which is its IP.
//
// Containers tend to put the IP address in eth0, so this attempts to look up
// that interface and retrieve its IP. It is fairly naive. To get more
// thorough IP information, you may prefer to use the `net` package and
// look up the desired information.
//
// Because this queries the interfaces, not the Kube API server, this could,
// in theory, return an IP address different from Me.IP.
func MyIP() (string, error) {

	maxIface := 5
	var err error
	for i := 0; i < maxIface; i++ {
		var ip string
		ip, err = IPByInterface(fmt.Sprintf("eth%d", i))
		if err == nil {
			return ip, nil
		}
	}

	return "0.0.0.0", err
}
func IPByInterface(name string) (string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	var ip string
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}
	if len(ip) == 0 {
		return ip, errors.New("Found no IPv4 addresses.")
	}
	return ip, nil
}
