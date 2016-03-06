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

package cmd

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fabric8io/kansible/version"
)

var versionInfoTmpl = `
kansible, version {{.version}} (branch: {{.branch}}, revision: {{.revision}})
  build user:       {{.buildUser}}
  build date:       {{.buildDate}}
  go version:       {{.goVersion}}
`

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Output version information and exit",
	Long:  `Output version information and exit.`,
	Run: func(cmd *cobra.Command, args []string) {
		t := template.Must(template.New("version").Parse(versionInfoTmpl))

		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "version", version.Map); err != nil {
			panic(err)
		}
		fmt.Fprintln(os.Stdout, strings.TrimSpace(buf.String()))
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
