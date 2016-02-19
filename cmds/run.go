package cmds

import (
	"github.com/codegangsta/cli"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/log"
	"github.com/fabric8io/kansible/ssh"
	"github.com/fabric8io/kansible/winrm"
)

func Run(c *cli.Context) {
	port, err := osExpandAndVerifyGlobal(c, "port")
	if err != nil {
		fail(err)
	}
	command, err := osExpandAndVerify(c, "command")
	if err != nil {
		fail(err)
	}
	host, err := osExpandAndVerify(c, "host")
	if err != nil {
		fail(err)
	}
	user, err := osExpandAndVerify(c, "user")
	if err != nil {
		fail(err)
	}
	connection := c.String("connection")
	if connection == ansible.ConnectionWinRM {
		password, err := osExpandAndVerify(c, "password")
		if err != nil {
			fail(err)
		}
		err = winrm.RemoteWinRmCommand(user, password, host, port, command)
	} else {
		privatekey, err := osExpandAndVerify(c, "privatekey")
		if err != nil {
			fail(err)
		}
		err = ssh.RemoteSshCommand(user, privatekey, host, port, command)
	}
	if err != nil {
		log.Err("Failed: %v", err)
	}
}
