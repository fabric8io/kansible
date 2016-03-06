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
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
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
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	variables := map[string]string{}
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
func LoadFileAndReplaceVariables(filename string, variables map[string]string) ([]byte, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// TODO replace the variables!
	text := ReplaceVariables(string(data), variables)
	return []byte(text), nil
}
