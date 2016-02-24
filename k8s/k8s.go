package k8s

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/util/strategicpatch"

	"github.com/ghodss/yaml"

	"github.com/fabric8io/kansible/log"
)


// GetThisPodName returns this pod name via the `HOSTNAME` environment variable
func GetThisPodName() (string, error) {
	var err error
	thisPodName := os.Getenv("HOSTNAME")
	if len(thisPodName) == 0 {
		thisPodName, err = os.Hostname()
		if err != nil {
			return "", err
		}
	}
	if len(thisPodName) == 0 {
		return "", fmt.Errorf("Could not find the pod name using $HOSTNAME!")
	}
	return thisPodName, nil
}
// ReadReplicationControllerFromFile reads the ReplicationController object from the given file name
func ReadReplicationControllerFromFile(filename string) (*api.ReplicationController, error) {
	data, err := ReadBytesFromFile(filename)
	if err != nil {
		return nil, err
	}
	return ReadReplicationController(data)
}

// ReadReplicationController loads a ReplicationController from the given data
func ReadReplicationController(data []byte) (*api.ReplicationController, error) {
	rc := api.ReplicationController{}
	if err := yaml.Unmarshal(data, &rc); err != nil {
		return nil, err
	}
	return &rc, nil
}

// ReadBytesFromFile loads the given file into memory
func ReadBytesFromFile(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, nil
}


// PodIsRunning returns true if the given pod is running in the given list of all pods
func PodIsRunning(pods *api.PodList, podName string) bool {
	for _, pod := range pods.Items {
		if pod.ObjectMeta.Name == podName {
			return true
		}
	}
	return false
}

// GetFirstContainerOrCreate returns the first Container in the PodSpec for this ReplicationController
// lazily creating structures as required
func GetFirstContainerOrCreate(rc *api.ReplicationController) *api.Container {
	podSpec := GetOrCreatePodSpec(rc)
	if len(podSpec.Containers) == 0 {
		podSpec.Containers[0] = api.Container{}
	}
	return &podSpec.Containers[0];
}

// GetOrCreatePodSpec returns the PodSpec for this ReplicationController
// lazily creating structures as required
func GetOrCreatePodSpec(rc *api.ReplicationController) *api.PodSpec {
	spec := &rc.Spec
	if spec == nil {
		rc.Spec = api.ReplicationControllerSpec{}
		spec = &rc.Spec
	}
	template := spec.Template
	if template == nil {
		spec.Template = &api.PodTemplateSpec{}
		template = spec.Template
	}
	podSpec := &template.Spec
	if podSpec == nil {
		template.Spec = api.PodSpec{}
		podSpec = &template.Spec
	}
	return podSpec
}

// GetContainerEnvVar returns the environment variable value for the given name in the Container
func GetContainerEnvVar(container *api.Container, name string) string {
	if container != nil {
		for _, env := range container.Env {
			if env.Name == name {
				return env.Value
			}
		}
	}
	return ""
}

// EnsureContainerHasEnvVar if there is an existing EnvVar for the given name then lets update it
// with the given value otherwise lets add a new entry.
// Returns true if there was already an existing environment variable
func EnsureContainerHasEnvVar(container *api.Container, name string, value string) bool {
	for _, env := range container.Env {
		if env.Name == name {
			env.Value = value
			return true
		}
	}
	container.Env = append(container.Env, api.EnvVar{
		Name: name,
		Value: value,
	})
	return false
}

// EnsureContainerHasEnvVarFromField if there is an existing EnvVar for the given name then lets update it
// with the given fieldPath otherwise lets add a new entry.
// Returns true if there was already an existing environment variable
func EnsureContainerHasEnvVarFromField(container *api.Container, name string, fieldPath string) bool {
	from := &api.EnvVarSource{
		FieldRef: &api.ObjectFieldSelector{
			FieldPath: fieldPath,
		},
	}
	for _, env := range container.Env {
		if env.Name == name {
			env.ValueFrom = from
			env.Value = ""
			return true
		}
	}
	container.Env = append(container.Env, api.EnvVar{
		Name: name,
		ValueFrom: from,
	})
	return false
}


// EnsureContainerHasPreStopCommand ensures that the given container has a `preStop` lifecycle hook
// to invoke the given commands
func EnsureContainerHasPreStopCommand(container *api.Container, commands []string) {
	if container.Lifecycle == nil {
		container.Lifecycle = &api.Lifecycle{}
	}
	lifecycle := container.Lifecycle
	if lifecycle.PreStop == nil {
		lifecycle.PreStop = &api.Handler{}
	}
	preStop := lifecycle.PreStop
	preStop.Exec = &api.ExecAction{
		Command: commands,
	}
}

// EnsureContainerHasVolumeMount ensures that there is a volume mount of the given name with the given values
// Returns true if there was already a volume mount
func EnsureContainerHasVolumeMount(container *api.Container, name string, mountPath string) bool {
	for _, vm := range container.VolumeMounts {
		if vm.Name == name {
			vm.MountPath = mountPath
			return true
		}
	}
	container.VolumeMounts = append(container.VolumeMounts, api.VolumeMount{
		Name: name,
		MountPath: mountPath,
	})
	return false
}


// EnsurePodSpecHasGitVolume ensures that there is a volume with the given name and git repo and revision
func EnsurePodSpecHasGitVolume(podSpec *api.PodSpec, name string, gitRepo string, gitRevision string) bool {
	for _, vm := range podSpec.Volumes {
		if vm.Name == name {
			vm.GitRepo = &api.GitRepoVolumeSource{
				Repository: gitRepo,
				Revision: gitRevision,
			}
			return true
		}
	}
	podSpec.Volumes = append(podSpec.Volumes, api.Volume{
		Name: name,
		VolumeSource: api.VolumeSource{
			GitRepo: &api.GitRepoVolumeSource{
				Repository: gitRepo,
				Revision: gitRevision,
			},
		},
	})
	return false
}


// EnsurePodSpecHasSecretVolume ensures that there is a volume with the given name and secret
func EnsurePodSpecHasSecretVolume(podSpec *api.PodSpec, name string, secretName string) bool {
	for _, vm := range podSpec.Volumes {
		if vm.Name == name {
			vm.Secret = &api.SecretVolumeSource{
				SecretName: secretName,
			}
			return true
		}
	}
	podSpec.Volumes = append(podSpec.Volumes, api.Volume{
		Name: name,
		VolumeSource: api.VolumeSource{
			Secret: &api.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	})
	return false
}



// EnsureServiceAccountExists ensures that there is a service account created for the given name
func EnsureServiceAccountExists(c *client.Client, ns string, serviceAccountName string) error {
	saClient := c.ServiceAccounts(ns)
	sa, err := saClient.Get(serviceAccountName)
	if err != nil || sa == nil {
		// lets try create the SA
		sa = &api.ServiceAccount{
			ObjectMeta: api.ObjectMeta{
				Name: serviceAccountName,
			},
		}
		log.Info("Creating ServiceAccount %s", serviceAccountName)
		_, err = saClient.Create(sa)
	}
	return nil
}

// ApplyResource applies the given data as a kubernetes resource
func ApplyResource(f *cmdutil.Factory, c *client.Client, ns string, data []byte, name string) error {
	schemaCacheDir := "/tmp/kubectl.schema"
	validate := true
	schema, err := f.Validator(validate, schemaCacheDir)
	if err != nil {
		log.Info("Failed to load kubernetes schema: %s", err)
		return err
	}

	mapper, typer := f.Object()
	r := resource.NewBuilder(mapper, typer, f.ClientMapperForCommand()).
	Schema(schema).
	ContinueOnError().
	NamespaceParam(ns).DefaultNamespace().
	Stream(bytes.NewReader(data), name).
	Flatten().
	Do()
	err = r.Err()
	if err != nil {
		log.Info("Failed to load mapper!")
		return err
	}

	count := 0
	err = r.Visit(func(info *resource.Info, err error) error {
		// In this method, info.Object contains the object retrieved from the server
		// and info.VersionedObject contains the object decoded from the input source.
		if err != nil {
			return err
		}

		// Get the modified configuration of the object. Embed the result
		// as an annotation in the modified configuration, so that it will appear
		// in the patch sent to the server.
		modified, err := kubectl.GetModifiedConfiguration(info, true)
		if err != nil {
			return cmdutil.AddSourceToErr(fmt.Sprintf("retrieving modified configuration from:\n%v\nfor:", info), info.Source, err)
		}

		if err := info.Get(); err != nil {
			return cmdutil.AddSourceToErr(fmt.Sprintf("retrieving current configuration of:\n%v\nfrom server for:", info), info.Source, err)
		}

		// Serialize the current configuration of the object from the server.
		current, err := info.Mapping.Codec.Encode(info.Object)
		if err != nil {
			return cmdutil.AddSourceToErr(fmt.Sprintf("serializing current configuration from:\n%v\nfor:", info), info.Source, err)
		}

		// Retrieve the original configuration of the object from the annotation.
		original, err := kubectl.GetOriginalConfiguration(info)
		if err != nil {
			return cmdutil.AddSourceToErr(fmt.Sprintf("retrieving original configuration from:\n%v\nfor:", info), info.Source, err)
		}

		// Compute a three way strategic merge patch to send to server.
		patch, err := strategicpatch.CreateThreeWayMergePatch(original, modified, current, info.VersionedObject, false)
		if err != nil {
			format := "creating patch with:\noriginal:\n%s\nmodified:\n%s\ncurrent:\n%s\nfrom:\n%v\nfor:"
			return cmdutil.AddSourceToErr(fmt.Sprintf(format, original, modified, current, info), info.Source, err)
		}

		helper := resource.NewHelper(info.Client, info.Mapping)
		_, err = helper.Patch(info.Namespace, info.Name, api.StrategicMergePatchType, patch)
		if err != nil {
			return cmdutil.AddSourceToErr(fmt.Sprintf("applying patch:\n%s\nto:\n%v\nfor:", patch, info), info.Source, err)
		}

		count++
		cmdutil.PrintSuccess(mapper, false, os.Stdout, info.Mapping.Resource, info.Name, "configured")
		return nil
	})

	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("no objects passed to apply")
	}
	return nil
}
