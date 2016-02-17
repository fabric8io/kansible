package k8s

import (
	"io/ioutil"
	"os"

	"k8s.io/kubernetes/pkg/api"

	"github.com/ghodss/yaml"
)

// ReadReplicationControllerFromFile reads the ReplicationController object from the given file name
func ReadReplicationControllerFromFile(filename string) (*api.ReplicationController, error) {
	data, err := readBytesFromFile(filename)
	if err != nil {
		return nil, err
	}
	rc := api.ReplicationController{}
	// TODO(jackgr): Replace with a call to testapi.Codec().Decode().
	if err := yaml.Unmarshal(data, &rc); err != nil {
		return nil, err
	}
	return &rc, nil
}

func readBytesFromFile(filename string) ([]byte, error) {
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
	if len(podSpec.Containers) == 0 {
		podSpec.Containers[0] = api.Container{}
	}
	return &podSpec.Containers[0];
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