package cmds

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"

	"github.com/fabric8io/kansible/log"
)

func osExpand(c *cli.Context, name string) string {
	flag := c.String(name)
	value := os.ExpandEnv(flag)
	log.Debug("flag %s is %s", name, value)
	return value
}

func osExpandAndVerify(c *cli.Context, name string) (string, error) {
	value := osExpand(c, name)
	if len(value) == 0 {
		return "", fmt.Errorf("No parameter supplied for: %s", name)
	}
	return value, nil
}

func osExpandAndVerifyGlobal(c *cli.Context, name string) (string, error) {
	flag := c.GlobalString(name)
	value := os.ExpandEnv(flag)
	if len(value) == 0 {
		return "", fmt.Errorf("No parameter supplied for: %s", name)
	}
	log.Debug("flag %s is %s", name, value)
	return value, nil
}

func fail(err error) {
	log.Die("Failed: %s", err)
}

