package ansible

import (
	"os"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/fabric8io/kansible/k8s"
)

const (
	// AnsibleGlobalVariablesFile is the prefix file name for the Ansible global variables file
	AnsibleGlobalVariablesFile = "group_vars/"

)
// LoadAnsibleVariables loads the global variables from the Ansible playbook
// so that we can search and replace them inside other files like the RC.yml
func LoadAnsibleVariables(hosts string) (map[string]string, error) {
	path := AnsibleGlobalVariablesFile + hosts
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	variables := map[string]string{}
	data, err := k8s.ReadBytesFromFile(path)
	if err != nil {
		return variables, err
	}
	err = yaml.Unmarshal(data, &variables)
	if err != nil {
		return variables, err
	}
	for k, v := range variables {
		variables[k] = ReplaceVariables(v, variables)
	}
	return variables, nil
}

// ReplaceVariables replaces variables in the given string using the Ansible variable syntax of
// `{{ name }}`
func ReplaceVariables(text string, variables map[string]string) string {
	for k, v := range variables {
		replace := "{{ " + k + " }}"
		text = strings.Replace(text, replace, v, -1)
	}
	return text
}

// LoadFileAndReplaceVariables loads the given file and replaces all the Ansible variable expressions
// and then returns the data
func LoadFileAndReplaceVariables(filename string, variables map[string]string) ([]byte, error)  {
	data, err := k8s.ReadBytesFromFile(filename)
	if err != nil {
		return nil, err
	}
	// TODO replace the variables!
	text := ReplaceVariables(string(data), variables)
	return []byte(text), nil
}