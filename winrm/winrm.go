package winrm

import(
	"fmt"
	"os"

	"github.com/fabric8io/gosupervise/log"
	"github.com/masterzen/winrm/winrm"
)

// RemoteWinRmCommand runs the remote command on a windows machine
func RemoteWinRmCommand(user string, password string, host string, port string, cmd string) error {
	log.Info("Connecting to windows host over WinRM on host %s", host)
	client, err := winrm.NewClient(&winrm.Endpoint{Host: host, Port: 5985, HTTPS: false, Insecure: false}, user, password)
	if err != nil {
	    fmt.Println(err)
	}
	run, err := client.RunWithInput(cmd, os.Stdout, os.Stderr, os.Stdin)
	if err != nil {
	    fmt.Println(err)
	}
	fmt.Println(run)
	return nil
}

