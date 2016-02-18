package winrm

import(
	"fmt"
	"os"

	"github.com/fabric8io/kansible/log"
	"github.com/masterzen/winrm/winrm"
	"strconv"
)

// RemoteWinRmCommand runs the remote command on a windows machine
func RemoteWinRmCommand(user string, password string, host string, port string, cmd string) error {
	portNumber, err := strconv.Atoi(port)
	if err != nil {
	    log.Err("Failed to convert port number text `%s` to a number: %s", port, err)
		return nil
	}
	log.Info("Connecting to windows host over WinRM on host %s and port %d with user %s with command `%s`", host, portNumber, user, cmd)
	client, err := winrm.NewClient(&winrm.Endpoint{Host: host, Port: portNumber, HTTPS: false, Insecure: false}, user, password)
	if err != nil {
	    fmt.Println(err)
		return nil
	}
	run, err := client.RunWithInput(cmd, os.Stdout, os.Stderr, os.Stdin)
	if err != nil {
	    fmt.Println(err)
		return nil
	}
	fmt.Println(run)
	return nil
}

