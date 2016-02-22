package ansible

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"math/rand"
	"sort"
	"strings"
	"time"

	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/api"

	"github.com/fabric8io/kansible/log"
	"github.com/fabric8io/kansible/k8s"
	"strconv"
)

const (
// AnsibleHostPodAnnotationPrefix is the annotation prefix used on the RC to associate a host name with a pod name
	AnsibleHostPodAnnotationPrefix = "pod.ansible.fabric8.io/"

// HostInventoryAnnotation is the list of hosts from the inventory
	HostInventoryAnnotation = "ansible.fabric8.io/host-inventory"

// HostNameAnnotation is used to annotate a pod with the host name its processing
	HostNameAnnotation = "ansible.fabric8.io/host-name"

// HostAddressAnnotation is used to annotate a pod with the host address its processing
	HostAddressAnnotation = "ansible.fabric8.io/host-address"

// EnvHosts is the environment variable on a pod for specifying the Ansible hosts in the inventory
	EnvHosts = "KANSIBLE_HOSTS"

// EnvCommand is the environment variable on a pod for specifying the command to run on each host
	EnvCommand = "KANSIBLE_COMMAND"

// EnvRC is the environment variable on a pod for the name of the ReplicationController
	EnvRC = "KANSIBLE_RC"

// EnvExportEnvVars is the space separated list of environment variables exported to the remote process
	EnvExportEnvVars = "KANSIBLE_EXPORT_ENV_VARS"

// EnvBash is the environment variable on a pod for the name of the bash script to generate on startup for
// opening a remote shell
	EnvBash = "KANSIBLE_BASH"

// PlaybookVolumeMount is the volume mount point where the playbook is assumed to be in the supervisor pod
	PlaybookVolumeMount = "/playbook"


// AnsibleVariableHost is the Ansible inventory host variable for the remote host
	AnsibleVariableHost = "ansible_host"

// AnsibleVariableUser is the Ansible inventory host variable for the remote user
	AnsibleVariableUser = "ansible_user"

// AnsibleVariablePort is the Ansible inventory host variable for the reote port
	AnsibleVariablePort = "ansible_port"

// AnsibleVariablePrivateKey is the Ansible inventory host variable for the SSH private key file
	AnsibleVariablePrivateKey = "ansible_ssh_private_key_file"

// AnsibleVariableConnection is the Ansible inventory host variable for the kind of connection; e.g. 'winrm' for windows
	AnsibleVariableConnection = "ansible_connection"

// AnsibleVariablePassword is the Ansible inventory host variable for the password
	AnsibleVariablePassword = "ansible_ssh_pass"

// ConnectionWinRM is the value AnsibleVariableConnection of for using Windows with WinRM
	ConnectionWinRM = "winrm"

// AppRunCommand is the Ansible inventory host variable for the run command that is executed on the remote host
	AppRunCommand = "app_run_command"

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
	Connection string
	Password   string
	RunCommand string
}

// LoadHostEntries loads the Ansible inventory for a given hosts string value
func LoadHostEntries(inventoryFile string, hosts string) ([]*HostEntry, error) {
	file, err := os.Open(inventoryFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hostEntries := []*HostEntry{}
	hostsLine := "[" + hosts + "]"
	foundHeader := false
	completed := false
	hostNames := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if len(text) > 0 && !strings.HasPrefix(text, "#") {
			isHost := strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]")
			if isHost {
				hostNames = append(hostNames, text[1:len(text) - 1])
			}
			if (foundHeader) {
				if isHost {
					completed = true
				} else if !completed {
					hostEntry := parseHostEntry(text)
					if hostEntry != nil {
						hostEntries = append(hostEntries, hostEntry)
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
	if !foundHeader {
		sort.Strings(hostNames)
		return nil, fmt.Errorf("Could not find hosts `%s` in Ansible inventory file %s. Possible values are: %s",
			hosts, inventoryFile, strings.Join(hostNames, ", "))
	}
	return hostEntries, nil
}



// LoadHostEntriesFromText loads the host entries from the given text which is typically taken from
// an annotation on the ReplicationController
func LoadHostEntriesFromText(text string) ([]*HostEntry, error) {
	hostEntries := []*HostEntry{}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		text := strings.TrimSpace(line)
		if len(text) > 0 && !strings.HasPrefix(text, "#") {
			hostEntry := parseHostEntry(text)
			if hostEntry != nil {
				hostEntries = append(hostEntries, hostEntry)
			}
		}
	}
	return hostEntries, nil
}


// ChooseHostAndPrivateKey parses the given Ansible inventory file for the hosts
// and chooses a single host inside it, returning the host name and the private key
func ChooseHostAndPrivateKey(inventoryFile string, hosts string, c *client.Client, ns string, rcName string, envVars map[string]string) (*HostEntry, error) {
	var err error
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

		hostsText := annotations[HostInventoryAnnotation]
		if len(hostsText) == 0 {
			return nil, fmt.Errorf("Could not find annotation %s on ReplicationController %s", HostInventoryAnnotation, rcName)
		}
		hostEntries, err := LoadHostEntriesFromText(hostsText)
		if err != nil {
			return nil, err
		}
		log.Info("Found %d host entries", len(hostEntries))

		// lets pick a random entry
		if len(hostEntries) > 0 {
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

				// lets update the Pod with the host name label
				podClient := c.Pods(ns)
				pod, err := podClient.Get(thisPodName)
				if err != nil {
					return pickedEntry, err
				}
				metadata := &pod.ObjectMeta
				if metadata.Annotations == nil {
					metadata.Annotations = make(map[string]string)
				}
				metadata.Annotations[HostNameAnnotation] = pickedEntry.Name
				metadata.Annotations[HostAddressAnnotation] = pickedEntry.Host
				//pod.Status = api.PodStatus{}
				_, err = podClient.UpdateStatus(pod)
				if err != nil {
					return pickedEntry, err
				}

				// lets export required environment variables
				exportEnvVars := os.Getenv(EnvExportEnvVars)
				if len(exportEnvVars) > 0 {
					names := strings.Split(exportEnvVars, " ")
					for _, name := range names {
						name = strings.TrimSpace(name)
						if len(name) > 0 {
							value := os.Getenv(name)
							if len(value) > 0 {
								envVars[name] = value
								log.Debug("Exporting environment variable %s = %s", name, value)
							}
						}
					}
				}

				err = forwardPorts(pod, pickedEntry)
				return pickedEntry, err
			}
		}
	}
	return nil, fmt.Errorf("Could not find any hosts for inventory file %s and hosts %s", inventoryFile, hosts)
}

// forwardPorts forwards any ports that are defined in the PodSpec to the host
func forwardPorts(pod *api.Pod, hostEntry *HostEntry) error {
	podSpec := pod.Spec
	host := hostEntry.Host
	for _, container := range podSpec.Containers {
		for _, port := range container.Ports {
			name := port.Name
			portNum := port.ContainerPort
			if portNum > 0 {
				address := "localhost:" + strconv.Itoa(portNum)
				forwardAddress := host + ":" + strconv.Itoa(portNum)
				err := forwardPortLoop(name, address, forwardAddress)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func forwardPortLoop(name string, address string, forwardAddress string) error {
	log.Info("forwarding port %s %s => %s", name, address, forwardAddress)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	log.Info("About to start the acceptor goroutine!")
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Err("Failed to accept listener: %v", err)
			}
			log.Info("Accepted connection %v\n", conn)
			go forwardPort(conn, forwardAddress)
		}
	}()
	return nil
}

func forwardPort(conn net.Conn, address string) {
	client, err := net.Dial("tcp", address)
	if err != nil {
		log.Err("Dial failed: %v", err)
	}
	log.Info("Connected to localhost %v\n", conn)
	go func() {
		defer client.Close()
		defer conn.Close()
		io.Copy(client, conn)
	}()
	go func() {
		defer client.Close()
		defer conn.Close()
		io.Copy(conn, client)
	}()
}

// UpdateKansibleRC reads the Ansible inventory and the RC YAML for the hosts and updates it in Kubernetes
// along with removing any remaining pods which are running against old hosts that have been removed from the inventory
func UpdateKansibleRC(hostEntries []*HostEntry, hosts string, c *client.Client, ns string, rcFile string) (*api.ReplicationController, error) {
	variables, err := LoadAnsibleVariables(hosts)
	if err != nil {
		return nil, err
	}
	data, err := LoadFileAndReplaceVariables(rcFile, variables)
	if err != nil {
		return nil, err
	}
	rcConfig, err := k8s.ReadReplicationController(data)
	if err != nil {
		return nil, err
	}
	rcName := rcConfig.ObjectMeta.Name
	podSpec := k8s.GetOrCreatePodSpec(rcConfig)

	// lets default labels and selectors if they are missing
	rcLabels := rcConfig.ObjectMeta.Labels
	if len(rcLabels) > 0 {
		rcSpec := rcConfig.Spec
		if len(rcSpec.Selector) == 0 {
			rcSpec.Selector = rcLabels
		}
		template := rcSpec.Template
		if template != nil {
			if len(template.ObjectMeta.Labels) == 0 {
				template.ObjectMeta.Labels = rcLabels
			}
		}
	}

	container := k8s.GetFirstContainerOrCreate(rcConfig)
	if len(container.Image) == 0 {
		container.Image = "fabric8/kansible"
	}
	if len(container.Name) == 0 {
		container.Name = "kansible"
	}
	if len(container.ImagePullPolicy) == 0 {
		container.ImagePullPolicy = "IfNotPresent"
	}
	k8s.EnsureContainerHasEnvVar(container, EnvHosts, hosts)
	k8s.EnsureContainerHasEnvVar(container, EnvRC, rcName)
	k8s.EnsureContainerHasEnvVar(container, EnvBash, "/usr/local/bin/bash")
	command := k8s.GetContainerEnvVar(container, EnvCommand)
	if len(command) == 0 {
		return nil, fmt.Errorf("No environemnt variable value defined for %s in ReplicationController YAML file %s", EnvCommand, rcFile)
	}

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

	metadata := &rc.ObjectMeta
	resourceVersion := metadata.ResourceVersion
	rcSpec := &rc.Spec
	hostCount := len(hostEntries)
	replicas := originalReplicas
	if replicas == 0 || replicas > hostCount {
		replicas = hostCount
	}
	rcSpec.Replicas = replicas

	err = generatePrivateKeySecrets(c, ns, hostEntries, rc, podSpec, container)
	if err != nil {
		return rc, err
	}

	text := HostEntriesToString(hostEntries)
	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}
	metadata.Annotations[HostInventoryAnnotation] = text

	log.Info("found RC with name %s and version %s and replicas %d", rcName, resourceVersion, rcSpec.Replicas)

	deletePodsForOldHosts(c, ns, metadata.Annotations, pods, hostEntries)

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

func generatePrivateKeySecrets(c *client.Client, ns string, hostEntries []*HostEntry, rc *api.ReplicationController, podSpec *api.PodSpec, container *api.Container) error {
	secrets := map[string]string{}
	rcName := rc.ObjectMeta.Name

	for _, hostEntry := range hostEntries {
		privateKey := hostEntry.PrivateKey
		if len(privateKey) != 0 {
			volumeMount := secrets[privateKey]
			if len(volumeMount) == 0 {
				buffer, err := ioutil.ReadFile(privateKey)
				if err != nil {
					return err
				}
				hostName := hostEntry.Name
				secretName := rcName + "-" + hostName
				keyName := "sshkey"
				secret := &api.Secret{
					ObjectMeta: api.ObjectMeta{
						Name: secretName,
						Labels: rc.ObjectMeta.Labels,
					},
					Data: map[string][]byte{
						keyName: buffer,
					},
				}

				// lets create or update the secret
				secretClient := c.Secrets(ns);
				current, err := secretClient.Get(secretName)
				if err != nil || current == nil {
					_, err = secretClient.Create(secret)
				} else {
					_, err = secretClient.Update(secret)
				}
				if err != nil {
					return err
				}

				volumeMount = "/secrets/" + hostName
				secrets[privateKey] = volumeMount
				hostEntry.PrivateKey = volumeMount + "/" + keyName

				// lets add the volume mapping to the container
				secretVolumeName := "secret-" + hostName
				k8s.EnsurePodSpecHasSecretVolume(podSpec, secretVolumeName, secretName)
				k8s.EnsureContainerHasVolumeMount(container, secretVolumeName, volumeMount)
			}
		}
	}
	return nil
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


func removeHostEntry(hostEntries []*HostEntry, name string) []*HostEntry {
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

func getHostEntryByName(hostEntries []*HostEntry, name string) *HostEntry {
	for _, entry := range hostEntries {
		if entry.Name == name {
			return entry
		}
	}
	return nil
}


func deletePodsForOldHosts(c *client.Client, ns string, annotations map[string]string, pods *api.PodList, hostEntries []*HostEntry) {
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

// HostEntriesToString generates the Ansible inventory text for the host entries
func HostEntriesToString(hostEntries []*HostEntry) string {
	var buffer bytes.Buffer
	for _, hostEntry := range hostEntries {
		hostEntry.write(&buffer)
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func (hostEntry HostEntry) write(buffer *bytes.Buffer) {
	buffer.WriteString(hostEntry.Name)
	host := hostEntry.Host
	if len(host) > 0 {
		buffer.WriteString(" ")
		buffer.WriteString(AnsibleVariableHost)
		buffer.WriteString("=")
		buffer.WriteString(host)
	}
	pk := hostEntry.PrivateKey
	if len(pk) > 0 {
		buffer.WriteString(" ")
		buffer.WriteString(AnsibleVariablePrivateKey)
		buffer.WriteString("=")
		buffer.WriteString(pk)
	}
	password := hostEntry.Password
	if len(password) > 0 {
		buffer.WriteString(" ")
		buffer.WriteString(AnsibleVariablePassword)
		buffer.WriteString("=")
		buffer.WriteString(password)
	}
	runCommand := hostEntry.RunCommand
	if len(runCommand) > 0 {
		buffer.WriteString(" ")
		buffer.WriteString(AppRunCommand)
		buffer.WriteString("=")
		buffer.WriteString(runCommand)
	}
	port := hostEntry.Port
	if len(port) > 0 {
		buffer.WriteString(" ")
		buffer.WriteString(AnsibleVariablePort)
		buffer.WriteString("=")
		buffer.WriteString(port)
	}
	user := hostEntry.User
	if len(user) > 0 {
		buffer.WriteString(" ")
		buffer.WriteString(AnsibleVariableUser)
		buffer.WriteString("=")
		buffer.WriteString(user)
	}
	connection := hostEntry.Connection
	if len(connection) > 0 {
		buffer.WriteString(" ")
		buffer.WriteString(AnsibleVariableConnection)
		buffer.WriteString("=")
		buffer.WriteString(connection)
	}
}


func parseHostEntry(text string) *HostEntry {
	values := strings.Split(text, " ")
	name := ""
	user := ""
	host := ""
	port := ""
	privateKey := ""
	connection := ""
	password := ""
	runCommand := ""
	count := len(values)
	if count > 0 {
		name = values[0];

		// lets parse the key value expressions for the host name
		for _, exp := range values[1:] {
			params := strings.Split(exp, "=")
			if len(params) == 2 {
				paramValue := params[1]
				switch (params[0]) {
				case AnsibleVariableHost:
					host = paramValue
				case AnsibleVariableUser:
					user = paramValue
				case AnsibleVariablePort:
					port = paramValue
				case AnsibleVariablePrivateKey:
					privateKey = paramValue
				case AnsibleVariableConnection:
					connection = paramValue
				case AnsibleVariablePassword:
					password = paramValue
				case AppRunCommand:
					runCommand = paramValue
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
		Connection: connection,
		Password: password,
		RunCommand: runCommand,
	}
}

