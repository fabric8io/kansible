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
	"github.com/fabric8io/gosupervise/k8s"
)

const (
	// AnsibleHostPodAnnotationPrefix is the annotation prefix used on the RC to associate a host name with a pod name
	AnsibleHostPodAnnotationPrefix = "pod.ansible.fabric8.io/"

	// EnvHosts is the environment variable on a pod for specifying the Ansible hosts in the inventory
	EnvHosts = "GOSUPERVISE_HOSTS"

	// EnvCommand is the environment variable on a pod for specifying the command to run on each host
	EnvCommand = "GOSUPERVISE_COMMAND"

	// PlaybookVolumeMount is the volume mount point where the playbook is assumed to be in the supervisor pod
	PlaybookVolumeMount = "/playbook"

	gitURLPrefix = "url = "
	gitConfig = ".git/config"
)

// HostEntry represents a single host entry in an Ansible inventory
type HostEntry struct {
	Name       string
	Host       string
	Port       string
	User       string
	PrivateKey string
	UseWinRM   bool
	Password   string
}

// LoadHostEntries loads the Ansible inventory for a given hosts string value
func LoadHostEntries(inventoryFile string, hosts string) ([]HostEntry, error) {
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
	return hostEntries, nil
}


// ChooseHostAndPrivateKey parses the given Ansible inventory file for the hosts
// and chooses a single host inside it, returning the host name and the private key
func ChooseHostAndPrivateKey(inventoryFile string, hosts string, c *client.Client, ns string, rcName string) (*HostEntry, error) {
	hostEntries, err := LoadHostEntries(inventoryFile, hosts)
	if err != nil {
		return nil, err
	}
	log.Info("Found %d host entries", len(hostEntries))

	// lets pick a random entry
	if len(hostEntries) > 0 {
		thisPodName := os.Getenv("HOSTNAME")
		if len(thisPodName) == 0 {
			thisPodName, err = os.Hostname()
			if err != nil {
				return nil, err
			}
		}
		if len(thisPodName) == 0 {
			return nil, fmt.Errorf("Could not find the pod name using $HOSTNAME!")
		}

		retryAttempts := 20

		for i := 0; i < retryAttempts; i++ {
			if i > 0 {
				// lets sleep before retrying
				time.Sleep(time.Duration(random(1000, 20000)) * time.Millisecond)
			}
			if c == nil {
				return nil, fmt.Errorf("No Kubernetes Client specified!")
			}
			rc, err := c.ReplicationControllers(ns).Get(rcName)
			if err != nil {
				return nil, err
			}
			if rc == nil {
				return nil, fmt.Errorf("No ReplicationController found for name %s", rcName)
			}

			pods, err := c.Pods(ns).List(nil, nil)
			if err != nil {
				return nil, err
			}

			metadata := &rc.ObjectMeta
			resourceVersion := metadata.ResourceVersion
			if metadata.Annotations == nil {
				metadata.Annotations = make(map[string]string)
			}
			annotations := metadata.Annotations
			log.Info("found RC with name %s.%s and version %s", ns, rcName, resourceVersion)

			filteredHostEntries := hostEntries
			for annKey, podName := range annotations {
				if strings.HasPrefix(annKey, AnsibleHostPodAnnotationPrefix) {
					hostName := annKey[len(AnsibleHostPodAnnotationPrefix):]

					if (k8s.PodIsRunning(pods, podName)) {
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
			if len(pickedEntry.User) == 0 {
				return nil, fmt.Errorf("Could not find User for entry %s", pickedEntry.Name)
			}

			// lets try pick this pod
			annotations[AnsibleHostPodAnnotationPrefix + hostName] = thisPodName

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

// UpdateAnsibleRC reads the Ansible inventory and the RC for the supervisor and updates it in Kubernetes
// along with removing any remaining pods which are for old hosts
func UpdateAnsibleRC(inventoryFile string, hosts string, c *client.Client, ns string, rcFile string) (*api.ReplicationController, error) {
	rcConfig, err := k8s.ReadReplicationControllerFromFile(rcFile)
	if err != nil {
		return nil, err
	}

	gitURL, err := findGitURL()
	if err != nil {
		return nil, err
	}
	if len(gitURL) == 0 {
		return nil, fmt.Errorf("Could not find git URL in git configu file %s", gitConfig)
	}

	podSpec := k8s.GetOrCreatePodSpec(rcConfig)
	container := k8s.GetFirstContainerOrCreate(rcConfig)
	if len(container.Image) == 0 {
		container.Image = "fabric8/gosupervise"
	}
	if len(container.Name) == 0 {
		container.Name = "gosupervise"
	}
	if len(container.ImagePullPolicy) == 0 {
		container.ImagePullPolicy = "IfNotPresent"
	}
	k8s.EnsureContainerHasEnvVar(container, EnvHosts, hosts)
	command := k8s.GetContainerEnvVar(container, EnvCommand)
	if len(command) == 0 {
		return nil, fmt.Errorf("No environemnt variable value defined for %s in ReplicationController YAML file %s", EnvCommand, rcFile)
	}
	volumeName := "playbook"
	k8s.EnsurePodSpecHasGitVolume(podSpec, volumeName, gitURL, "")
	k8s.EnsureContainerHasGitVolumeMount(container, volumeName, PlaybookVolumeMount)

	hostEntries, err := LoadHostEntries(inventoryFile, hosts)
	if err != nil {
		return nil, err
	}
	log.Info("Found %d host entries in the Ansible inventory for %s", len(hostEntries), hosts)
	log.Info("Using git URL %s", gitURL)

	rcName := rcConfig.ObjectMeta.Name
	isUpdate := true
	rc, err := c.ReplicationControllers(ns).Get(rcName)
	if err != nil {
		isUpdate = false
		rc = &api.ReplicationController{
			ObjectMeta: api.ObjectMeta{
				Namespace: ns,
				Name: rcName,
			},
		}
	}
	pods, err := c.Pods(ns).List(nil, nil)
	if err != nil {
		return nil, err
	}

	// merge the RC configuration to allow configuration
	originalReplicas := rc.Spec.Replicas
	rc.Spec = rcConfig.Spec

	metadata := rc.ObjectMeta
	resourceVersion := metadata.ResourceVersion
	annotations := metadata.Annotations
	rcSpec := &rc.Spec
	hostCount := len(hostEntries)
	replicas := originalReplicas
	if replicas == 0 || replicas > hostCount {
		replicas = hostCount
	}
	rcSpec.Replicas = replicas

	log.Info("found RC with name %s and version %s and replicas %d", rcName, resourceVersion, rcSpec.Replicas)

	deletePodsForOldHosts(c, ns, annotations, pods, hostEntries)

	replicationController := c.ReplicationControllers(ns)
	if isUpdate {
		_, err = replicationController.Update(rc)
	} else {
		_, err = replicationController.Create(rc)
	}
	if err != nil {
		log.Info("Failed to update the RC, could be concurrent update failure: %s", err)
		return nil, err
	}
	return rc, nil
}

func findGitURL() (string, error) {
	file, err := os.Open(gitConfig)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(text, gitURLPrefix) {
			return text[len(gitURLPrefix):], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}


func removeHostEntry(hostEntries []HostEntry, name string) []HostEntry {
	for i, entry := range hostEntries {
		if entry.Name == name {
			if i < len(hostEntries) - 1 {
				return append(hostEntries[:i], hostEntries[i + 1:]...)
			}
			return hostEntries[:i]
		}
	}
	log.Warn("Did not find a host entry with name %s", name)
	return hostEntries
}

func getHostEntryByName(hostEntries []HostEntry, name string) *HostEntry {
	for _, entry := range hostEntries {
		if entry.Name == name {
			return &entry
		}
	}
	return nil
}


func deletePodsForOldHosts(c *client.Client, ns string, annotations map[string]string, pods *api.PodList, hostEntries []HostEntry) {
	for annKey, podName := range annotations {
		if strings.HasPrefix(annKey, AnsibleHostPodAnnotationPrefix) {
			hostName := annKey[len(AnsibleHostPodAnnotationPrefix):]
			if (k8s.PodIsRunning(pods, podName)) {
				hostEntry := getHostEntryByName(hostEntries, hostName)
				if hostEntry == nil {
					log.Info("Deleting pod %s as there is no longer an Ansible inventory host called %s", podName, hostName)
					c.Pods(ns).Delete(podName, nil)
				}
			}
		}
	}
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
	port := ""
	privateKey := ""
	useWinRM := false
	password := ""
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
				case "ansible_port":
					port = paramValue
				case "ansible_ssh_private_key_file":
					privateKey = paramValue
				case "winrm":
					useWinRM = paramValue == "true"
				case "ansible_ssh_pass":
					password = paramValue
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
		Port: port,
		User: user,
		PrivateKey: privateKey,
		UseWinRM: useWinRM,
		Password: password,
	}
}