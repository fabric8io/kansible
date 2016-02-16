package ansible

import (
	"bufio"
	"fmt"
	"os"
	"math/rand"
	"strings"
	"time"

	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/api"

	"github.com/fabric8io/gosupervise/log"
)

const (
	AnsbileHostPodAnnotationPrefix = "pod.ansible.fabric8.io/"
)

type HostEntry struct {
	Name       string
	Host       string
	User       string
	PrivateKey string
}

// ChooseHostAndPrivateKey parses the given Ansbile inventory file for the hosts
// and chooses a single host inside it, returning the host name and the private key
func ChooseHostAndPrivateKey(inventoryFile string, hosts string, c *client.Client, ns string, rcName string) (*HostEntry, error) {
	file, err := os.Open(inventoryFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hostEntries := []HostEntry{}
	hostsLine := "[" + hosts + "]"
	foundHeader := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if len(text) > 0 {
			if (foundHeader) {
				if text[0] == '[' {
					break
				} else {
					hostEntry := parseHostEntry(text)
					if hostEntry != nil {
						hostEntries = append(hostEntries, *hostEntry)
					}
				}
			} else if text == hostsLine {
				foundHeader = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	log.Info("Found %v host entries", len(hostEntries))

	// lets pick a random entry
	if len(hostEntries) > 0 {

		thisPodName := os.Getenv("HOSTNAME")
		if len(thisPodName) == 0 {
			return nil, fmt.Errorf("Could not find the pod name using $HOSTNAME!")
		}

		retryAttempts := 20

		for i := 0; i < retryAttempts; i++ {
			if i > 0 {
				// lets sleep before retrying
				time.Sleep(time.Duration(random(1000, 20000)) * time.Millisecond)
			}
			rc, err := c.ReplicationControllers(ns).Get(rcName)
			if err != nil {
				return nil, err
			}

			pods, err := c.Pods(ns).List(nil, nil)
			if err != nil {
				return nil, err
			}

			metadata := rc.ObjectMeta
			resourceVersion := metadata.ResourceVersion
			annotations := metadata.Annotations
			log.Info("found RC with name %s and version %s", rcName, resourceVersion)

			filteredHostEntries := hostEntries
			for annKey, podName := range annotations {
				if strings.HasPrefix(annKey, AnsbileHostPodAnnotationPrefix) {
					hostName := annKey[len(AnsbileHostPodAnnotationPrefix):]

					if (podIsRunning(pods, podName)) {
						if podName != thisPodName {
							log.Info("Pod %s podName has already claimed host %s", podName, hostName)
							filteredHostEntries = removeHostEntry(filteredHostEntries, hostName)
						}
					} else {
						// lets remove this annotation as the pod is no longer valid
						log.Info("Pod %s is no longer running so removing the annotation %s", podName, annKey)
						delete(metadata.Annotations, annKey)
					}
				}
			}

			count := len(filteredHostEntries)

			if count == 0 {
				log.Info("There are no more hosts available to be supervised by this pod!")
				return nil, fmt.Errorf("No more hosts available to be supervised!")
			}
			log.Info("After filtering out hosts owned by other pods we have %v host entries left", count)

			pickedEntry := filteredHostEntries[random(0, count)]
			hostName := pickedEntry.Name;
			if len(pickedEntry.Host) == 0 {
				return nil, fmt.Errorf("Could not find host name for entry %s", pickedEntry.Name)
			}
			if len(pickedEntry.PrivateKey) == 0 {
				return nil, fmt.Errorf("Could not find PrivateKey for entry %s", pickedEntry.Name)
			}
			if len(pickedEntry.User) == 0 {
				return nil, fmt.Errorf("Could not find User for entry %s", pickedEntry.Name)
			}

			// lets try pick this pod
			annotations[AnsbileHostPodAnnotationPrefix + hostName] = thisPodName

			_, err = c.ReplicationControllers(ns).Update(rc)
			if err != nil {
				log.Info("Failed to update the RC, could be concurrent update failure: %s", err)
			} else {
				log.Info("Picked host " + pickedEntry.Host)
				return &pickedEntry, nil
			}
		}

	}
	return nil, fmt.Errorf("Could not find any hosts for inventory file %s and hosts %s", inventoryFile, hosts)
}

func removeHostEntry(hostEntries []HostEntry, name string) []HostEntry {
	for i, entry := range hostEntries {
		if entry.Name == name {
			if i < len(hostEntries) - 1 {
				return append(hostEntries[:i], hostEntries[i + 1:]...)
			} else {
				return hostEntries[:i]
			}
		}
	}
	log.Info("Did not find a host entry with name %s", name)
	return hostEntries
}


func podIsRunning(pods *api.PodList, podName string) bool {
	for _, pod := range pods.Items {
		if pod.ObjectMeta.Name == podName {
			return true
		}
	}
	return false
}


func random(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}

func parseHostEntry(text string) *HostEntry {
	values := strings.Split(text, " ")
	name := ""
	user := ""
	host := ""
	privateKey := ""
	count := len(values)
	if count > 0 {
		name = values[0];

		// lets parse the key value expressions for the host name
		for _, exp := range values[1:] {
			params := strings.Split(exp, "=")
			if len(params) == 2 {
				paramValue := params[1]
				switch (params[0]) {
				case "ansible_ssh_host":
					host = paramValue
				case "ansible_ssh_user":
					user = paramValue
				case "ansible_ssh_private_key_file":
					privateKey = paramValue
				}
			}
		}

		// if there's no host defined yet, lets assume that the name is the host name
		if len(host) == 0 {
			host = name
		}
	}
	return &HostEntry{
		Name: name,
		Host: host,
		User: user,
		PrivateKey: privateKey,
	}
}