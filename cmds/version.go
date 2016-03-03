package cmds

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/codegangsta/cli"

	"github.com/fabric8io/kansible/version"
)

var versionInfoTmpl = `
kansible, version {{.version}} (branch: {{.branch}}, revision: {{.revision}})
  build user:       {{.buildUser}}
  build date:       {{.buildDate}}
  go version:       {{.goVersion}}
`

// Version outputs the version & exits
func Version(c *cli.Context) {
	t := template.Must(template.New("version").Parse(versionInfoTmpl))

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "version", version.Map); err != nil {
		panic(err)
	}
	fmt.Fprintln(os.Stdout, strings.TrimSpace(buf.String()))
}
