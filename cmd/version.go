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
