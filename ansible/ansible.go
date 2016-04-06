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

package ansible

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/fabric8io/kansible/k8s"
	"github.com/fabric8io/kansible/log"
)

const (
	// AnsibleHostPodAnnotationPrefix is the annotation prefix used on the RC to associate a host name with a pod name
	AnsibleHostPodAnnotationPrefix = "pod.kansible.fabric8.io/"

	// HostInventoryAnnotation is the list of hosts from the inventory
	HostInventoryAnnotation = "kansible.fabric8.io/host-inventory"

	// HostNameAnnotation is used to annotate a pod with the host name its processing
	HostNameAnnotation = "kansible.fabric8.io/host-name"

	// HostAddressAnnotation is used to annotate a pod with the host address its processing
	HostAddressAnnotation = "kansible.fabric8.io/host-address"

	// WinRMShellAnnotationPrefix stores the shell ID for the WinRM host name on the RC
	WinRMShellAnnotationPrefix = "winrm.shellid.kansible.fabric8.io/"

	// EnvHosts is the environment variable on a pod for specifying the Ansible hosts in the inventory
	EnvHosts = "KANSIBLE_HOSTS"

	// EnvCommand is the environment variable on a pod for specifying the command to run on each host
	EnvCommand = "KANSIBLE_COMMAND"

	// EnvRC is the environment variable on a pod for the name of the ReplicationController
	EnvRC = "KANSIBLE_RC"

	// EnvNamespace is the environment variable on a pod for the namespace to use
	EnvNamespace = "KANSIBLE_NAMESPACE"

	// EnvExportEnvVars is the space separated list of environment variables exported to the remote process
	EnvExportEnvVars = "KANSIBLE_EXPORT_ENV_VARS"

	// EnvPortForward allows port forwarding to be disabled
	EnvPortForward = "KANSIBLE_PORT_FORWARD"

	// EnvBash is the environment variable on a pod for the name of the bash script to generate on startup for
	// opening a remote shell
	EnvBash = "KANSIBLE_BASH"

	// EnvIsBashShell is used to indicate of the command running remotely on the machine is a bash shell in which case we
	// don't want to delete any previous WinRM shell
	EnvIsBashShell = "KANSIBLE_IS_BASH_SHELL"

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
	gitConfig    = ".git/config"
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
				hostNames = append(hostNames, text[1:len(text)-1])
			}
			if foundHeader {
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
func ChooseHostAndPrivateKey(thisPodName string, hosts string, c *client.Client, ns string, rcName string) (*HostEntry, *api.ReplicationController, map[string]string, error) {
	retryAttempts := 20

	for i := 0; i < retryAttempts; i++ {
		if i > 0 {
			// lets sleep before retrying
			time.Sleep(time.Duration(random(1000, 20000)) * time.Millisecond)
		}
		if c == nil {
			return nil, nil, nil, fmt.Errorf("No Kubernetes Client specified!")
		}
		rc, err := c.ReplicationControllers(ns).Get(rcName)
		if err != nil {
			return nil, nil, nil, err
		}
		if rc == nil {
			return nil, nil, nil, fmt.Errorf("No ReplicationController found for name %s", rcName)
		}

		pods, err := c.Pods(ns).List(api.ListOptions{})
		if err != nil {
			return nil, nil, nil, err
		}

		metadata := &rc.ObjectMeta
		resourceVersion := metadata.ResourceVersion
		if metadata.Annotations == nil {
			metadata.Annotations = make(map[string]string)
		}
		annotations := metadata.Annotations
		log.Info("Using ReplicationController with namespace %s name %s and version %s", ns, rcName, resourceVersion)

		hostsText := annotations[HostInventoryAnnotation]
		if len(hostsText) == 0 {
			return nil, nil, nil, fmt.Errorf("Could not find annotation %s on ReplicationController %s", HostInventoryAnnotation, rcName)
		}
		hostEntries, err := LoadHostEntriesFromText(hostsText)
		if err != nil {
			return nil, nil, nil, err
		}
		log.Info("Found %d host entries", len(hostEntries))

		// lets pick a random entry
		if len(hostEntries) > 0 {
			filteredHostEntries := hostEntries
			for annKey, podName := range annotations {
				if strings.HasPrefix(annKey, AnsibleHostPodAnnotationPrefix) {
					hostName := annKey[len(AnsibleHostPodAnnotationPrefix):]
					if k8s.PodIsRunning(pods, podName) {
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
				return nil, nil, nil, fmt.Errorf("No more hosts available to be supervised!")
			}
			log.Info("After filtering out hosts owned by other pods we have %v host entries left", count)

			pickedEntry := filteredHostEntries[random(0, count)]
			hostName := pickedEntry.Name
			if len(pickedEntry.Host) == 0 {
				return nil, nil, nil, fmt.Errorf("Could not find host name for entry %s", pickedEntry.Name)
			}
			if len(pickedEntry.User) == 0 {
				return nil, nil, nil, fmt.Errorf("Could not find User for entry %s", pickedEntry.Name)
			}

			// lets try pick this pod
			annotations[AnsibleHostPodAnnotationPrefix+hostName] = thisPodName

			rc, err = c.ReplicationControllers(ns).Update(rc)
			if err != nil {
				log.Info("Failed to update the RC, could be concurrent update failure: %s", err)
			} else {
				log.Info("Picked host " + pickedEntry.Host)

				// lets update the Pod with the host name label
				podClient := c.Pods(ns)
				pod, err := podClient.Get(thisPodName)
				if err != nil {
					return pickedEntry, nil, nil, err
				}
				metadata := &pod.ObjectMeta
				if metadata.Annotations == nil {
					metadata.Annotations = make(map[string]string)
				}
				metadata.Annotations[HostNameAnnotation] = pickedEntry.Name
				metadata.Annotations[HostAddressAnnotation] = pickedEntry.Host
				//pod.Status = api.PodStatus{}
				pod, err = podClient.UpdateStatus(pod)
				if err != nil {
					return pickedEntry, nil, nil, err
				}

				// lets export required environment variables
				exportEnvVars := os.Getenv(EnvExportEnvVars)
				envVars := make(map[string]string)
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
				return pickedEntry, rc, envVars, err
			}
		}
	}
	return nil, nil, nil, fmt.Errorf("Could not find any available hosts on the ReplicationController %s and hosts %s", rcName, hosts)
}

// forwardPorts forwards any ports that are defined in the PodSpec to the host
func forwardPorts(pod *api.Pod, hostEntry *HostEntry) error {
	disableForwarding := os.Getenv(EnvPortForward)
	if len(disableForwarding) > 0 {
		if strings.ToLower(disableForwarding) == "false" {
			return nil
		}
	}
	podSpec := pod.Spec
	host := hostEntry.Host
	for _, container := range podSpec.Containers {
		for _, port := range container.Ports {
			name := port.Name
			portNum := port.ContainerPort
			if portNum > 0 {
				address := "0.0.0.0:" + strconv.Itoa(portNum)
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
func UpdateKansibleRC(hostEntries []*HostEntry, hosts string, f *cmdutil.Factory, c *client.Client, ns string, rcFile string, replicas int) (*api.ReplicationController, error) {
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
	preStopCommands := []string{"kansible", "kill"}
	if len(podSpec.ServiceAccountName) == 0 {
		podSpec.ServiceAccountName = rcName
	}
	serviceAccountName := podSpec.ServiceAccountName
	k8s.EnsureContainerHasPreStopCommand(container, preStopCommands)
	k8s.EnsureContainerHasEnvVar(container, EnvHosts, hosts)
	k8s.EnsureContainerHasEnvVar(container, EnvRC, rcName)
	k8s.EnsureContainerHasEnvVar(container, EnvBash, "/usr/local/bin/bash")
	k8s.EnsureContainerHasEnvVarFromField(container, EnvNamespace, "metadata.namespace")
	command := k8s.GetContainerEnvVar(container, EnvCommand)
	if len(command) == 0 {
		return nil, fmt.Errorf("No environemnt variable value defined for %s in ReplicationController YAML file %s", EnvCommand, rcFile)
	}

	if len(serviceAccountName) > 0 {
		created, err := k8s.EnsureServiceAccountExists(c, ns, serviceAccountName)
		if err != nil {
			return nil, err
		}
		if created {
			err = ensureSCCExists(ns, serviceAccountName)
			if err != nil {
				return nil, err
			}
		}
	}

	isUpdate := true
	rc, err := c.ReplicationControllers(ns).Get(rcName)
	if err != nil {
		isUpdate = false
		rc = &api.ReplicationController{
			ObjectMeta: api.ObjectMeta{
				Namespace: ns,
				Name:      rcName,
			},
		}
	}
	pods, err := c.Pods(ns).List(api.ListOptions{})
	if err != nil {
		return nil, err
	}

	// merge the RC configuration to allow configuration
	originalReplicas := rc.Spec.Replicas
	rc.Spec = rcConfig.Spec

	metadata := &rc.ObjectMeta
	resourceVersion := metadata.ResourceVersion
	rcSpec := &rc.Spec
	if replicas < 0 {
		replicas = originalReplicas
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

	err = applyOtherKubernetesResources(f, c, ns, rcFile, variables)
	return rc, err
}

func applyOtherKubernetesResources(f *cmdutil.Factory, c *client.Client, ns string, rcFile string, variables map[string]string) error {
	dir := filepath.Dir(rcFile)
	if len(dir) == 0 {
		dir = "."
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		name := file.Name()
		lower := strings.ToLower(name)
		ext := filepath.Ext(lower)
		if !file.IsDir() && lower != "rc.yml" {
			resource := false
			switch ext {
			case ".json":
				resource = true
			case ".js":
				resource = true
			case ".yml":
				resource = true
			case ".yaml":
				resource = true
			}
			if resource {
				fullpath := filepath.Join(dir, name)
				err = applyOtherKubernetesResource(f, c, ns, fullpath, variables)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func applyOtherKubernetesResource(f *cmdutil.Factory, c *client.Client, ns string, file string, variables map[string]string) error {
	log.Info("applying kubernetes resource: %s", file)
	data, err := LoadFileAndReplaceVariables(file, variables)
	if err != nil {
		return err
	}
	// TODO the following should work ideally but something's wrong with the loading of versioned schemas...
	//return k8s.ApplyResource(f, c, ns, data, file)

	// lets use the `oc` binary instead
	isOc := true
	binary, err := exec.LookPath("oc")
	if err != nil {
		isOc = false
		var err2 error
		binary, err2 = exec.LookPath("kubectl")
		if err2 != nil {
			return err
		}
	}
	reader := bytes.NewReader(data)
	err = runCommand(binary, []string{"apply", "-f", "-"}, reader)
	if err != nil {
		return err
	}
	if isOc {
		// if we are a service lets try figure out the service name?
		service := api.Service{}
		if err := yaml.Unmarshal(data, &service); err != nil {
			log.Info("Probably not a service! %s", err)
			return nil
		}
		name := service.ObjectMeta.Name
		serviceType := service.Spec.Type
		if service.Kind == "Service" && len(name) > 0 && serviceType == "LoadBalancer" {
			log.Info("Checking the service %s is exposed in OpenShift", name)
			runCommand(binary, []string{"expose", "service", name}, os.Stdin)
			return nil
		}
	}
	return nil
}

func ensureSCCExists(ns string, serviceAccountName string) error {
	binary, err := exec.LookPath("oc")
	if err != nil {
		// no openshift so ignore
		return nil
	}

	text, err := getCommandOutputString(binary, []string{"export", "scc", serviceAccountName}, os.Stdin)
	if err != nil {
		log.Debug("Failed to get SecurityContextConstraints %s. %s", serviceAccountName, err)
	}
	if err != nil || len(text) == 0 {
		text = `
apiVersion: v1
kind: SecurityContextConstraints
groups:
- system:cluster-admins
- system:nodes
metadata:
  creationTimestamp: null
  name: ` + serviceAccountName + `
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: RunAsAny
supplementalGroups:
  type: RunAsAny
users:
`
	}
	// lets ensure there's a users section
	if !strings.Contains(text, "\nusers:") {
		text = text + "\nusers:\n"
	}

	line := "system:serviceaccount:" + ns + ":" + serviceAccountName

	if strings.Contains(text, line) {
		log.Info("No need to modify SecurityContextConstraints as it already contains line for namespace %s and service account %s", ns, serviceAccountName)
		return nil
	}

	text = text + "\n- " + line + "\n"
	log.Debug("created SecurityContextConstraints YAML: %s", text)

	log.Info("Applying changes for SecurityContextConstraints %s for namespace %s and ServiceAccount %s", serviceAccountName, ns, serviceAccountName)
	reader := bytes.NewReader([]byte(text))
	err = runCommand(binary, []string{"apply", "-f", "-"}, reader)
	if err != nil {
		log.Err("Failed to update OpenShift SecurityContextConstraints named %s. %s", serviceAccountName, err)
	}
	return err
}

func getCommandOutputString(binary string, args []string, reader io.Reader) (string, error) {
	cmd := exec.Command(binary, args...)
	cmd.Stdin = reader

	var out bytes.Buffer
	cmd.Stdout = &out

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("Unable to setup stderr for command %s: %v", binary, err)
	}
	go io.Copy(os.Stderr, stderr)

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}
	return out.String(), err
}

func runCommand(binary string, args []string, reader io.Reader) error {
	cmd := exec.Command(binary, args...)
	cmd.Stdin = reader

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for command %s: %v", binary, err)
	}
	go io.Copy(os.Stdout, stdout)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stderr for command %s: %v", binary, err)
	}
	go io.Copy(os.Stderr, stderr)

	err = cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}
	return err
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
						Name:   secretName,
						Labels: rc.ObjectMeta.Labels,
					},
					Data: map[string][]byte{
						keyName: buffer,
					},
				}

				// lets create or update the secret
				secretClient := c.Secrets(ns)
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
			if i < len(hostEntries)-1 {
				return append(hostEntries[:i], hostEntries[i+1:]...)
			}
			return hostEntries[:i]
		}
	}
	log.Warn("Did not find a host entry with name %s", name)
	return hostEntries
}

// GetHostEntryByName finds the HostEntry for the given host name or returns nil
func GetHostEntryByName(hostEntries []*HostEntry, name string) *HostEntry {
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
			if k8s.PodIsRunning(pods, podName) {
				hostEntry := GetHostEntryByName(hostEntries, hostName)
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
	return rand.Intn(max-min) + min
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
		name = values[0]

		// lets parse the key value expressions for the host name
		for _, exp := range values[1:] {
			params := strings.Split(exp, "=")
			if len(params) == 2 {
				paramValue := params[1]
				switch params[0] {
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
		Name:       name,
		Host:       host,
		Port:       port,
		User:       user,
		PrivateKey: privateKey,
		Connection: connection,
		Password:   password,
		RunCommand: runCommand,
	}
}
