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