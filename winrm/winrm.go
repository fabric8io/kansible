package winrm

import(
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/fabric8io/kansible/ansible"
	"github.com/fabric8io/kansible/log"

	"github.com/masterzen/winrm/winrm"

	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"strings"
)

// RemoteWinRmCommand runs the remote command on a windows machine
func RemoteWinRmCommand(user string, password string, host string, port string, commandText string, c *client.Client, rc *api.ReplicationController, hostName string) error {
	portNumber, err := parsePortNumber(port)
	if err != nil {
		return err
	}
	log.Info("Connecting to windows host over WinRM on host %s and port %d with user %s with command `%s`", host, portNumber, user, commandText)
	client, err := winrm.NewClient(&winrm.Endpoint{Host: host, Port: portNumber, HTTPS: false, Insecure: false}, user, password)
	if err != nil {
		return fmt.Errorf("Could not create WinRM client: %s", err)
	}

	isBash := false
	isBashShellText := os.Getenv(ansible.EnvIsBashShell)
	if len(isBashShellText) > 0 && strings.ToLower(isBashShellText) == "false" {
		isBash = true
	}
	if rc.ObjectMeta.Annotations != nil && !isBash {
		oldShellID := rc.ObjectMeta.Annotations[ansible.WinRMShellAnnotationPrefix + hostName];
		if len(oldShellID) > 0 {
			// lets close the previously running shell on this machine
			log.Info("Closing the old WinRM Shell %s", oldShellID)
			shell := client.NewShell(oldShellID)
			err = shell.Close()
			if err != nil {
				log.Warn("Failed to close shell %s. Error: %s", oldShellID, err)
			}
		}
	}

	shell, err := client.CreateShell()
	if err != nil {
	  return fmt.Errorf("Impossible to create WinRM shell: %s", err)
	}
	defer shell.Close()
	shellID := shell.ShellId
	log.Info("Created WinRM Shell %s", shellID)

	if rc != nil && c != nil && !isBash {
		rc.ObjectMeta.Annotations[ansible.WinRMShellAnnotationPrefix + hostName] = shellID
		_, err = c.ReplicationControllers(rc.ObjectMeta.Namespace).UpdateStatus(rc)
		if err != nil {
			return err
		}
	}

	var cmd *winrm.Command
	cmd, err = shell.Execute(commandText)
	if err != nil {
		return fmt.Errorf("Impossible to create Command %s\n", err)
	}

	go io.Copy(cmd.Stdin, os.Stdin)
	go io.Copy(os.Stdout, cmd.Stdout)
	go io.Copy(os.Stderr, cmd.Stderr)

	cmd.Wait()
	shell.Close()
	return nil
}

// CloseShell closes the given WinRM Shell terminating any processes created within it
func CloseShell(user string, password string, host string, port string, shellID string) error {
	portNumber, err := parsePortNumber(port)
	if err != nil {
		return err
	}
	client, err := winrm.NewClient(&winrm.Endpoint{Host: host, Port: portNumber, HTTPS: false, Insecure: false}, user, password)
	if err != nil {
		return fmt.Errorf("Could not create WinRM client: %s", err)
	}

	log.Info("Closing shell %s", shellID)
	shell := client.NewShell(shellID)
	return shell.Close()
}

func parsePortNumber(port string) (int, error) {
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return 0, fmt.Errorf("Failed to convert port number text `%s` to a number: %s", port, err)
	}
	return portNumber, nil
}