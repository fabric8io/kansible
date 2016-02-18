package k8s

import (
	"io/ioutil"
	"os"

	"k8s.io/kubernetes/pkg/api"

	"github.com/ghodss/yaml"
)

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
