package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/fabric8io/gosupervise/log"
	"github.com/masterzen/winrm/winrm"
)

// version is the version of the app.
//
// This value is overwritten by the linker during build. The default version
// here is SemVer 2, but basically indicates that this was a one-off build
// and should not be trusted.
var version = "0.1.0-unstable"

func main() {
	app := cli.NewApp()
	app.Name = "gowindows"
	app.Usage = `Go Supervise

This command supervises a remote process inside a Pod inside Kubernetes to make
it look and feel like legacy processes running outside of Kubernetes are really
running inside Docker inside Kubernetes.

`
	app.Version = version
	app.EnableBashCompletion = true
	app.After = func(c *cli.Context) error {
		if log.ErrorState {
			return errors.New("Exiting with errors")
		}

		return nil
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "user",
			Value:  "$GOWINRM_USER",
			Usage:  "WINRM user to use on the remote machine",
			EnvVar: "GOWINRM_USER",
		},
		cli.StringFlag{
			Name:   "password",
			Value:  "$GOWINRM_PASSWORD",
			Usage:  "WINRM password",
			EnvVar: "GOWINRM_PASSWORD",
		},
		cli.StringFlag{
			Name:   "host",
			Value:  "127.0.0.1",
			Usage:  "WINRM hostname",
			EnvVar: "GOWINRM_HOST",
		},
		cli.StringFlag{
			Name:   "port",
			Value:  "5985",
			Usage:  "WINRM port",
			EnvVar: "GOWINRM_PORT",
		},
		cli.StringFlag{
			Name:   "command",
			Value:  "$GOSUPERVISE_COMAND",
			Usage:  "The remote command to invoke over WINRN",
		},
	}

	app.CommandNotFound = func(c *cli.Context, command string) {
		log.Err("No matching command '%s'", command)
		cli.ShowAppHelp(c)
		log.Die("")
	}

	app.Commands = []cli.Command{
		{
			Name:    "run",
			Usage:   "Runs the supervisor.",
			Description: `This commmand will begin running the supervisor on an avaiable host.`,
			ArgsUsage: "",
			Action: run,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.IsDebugging = c.Bool("debug")
		return nil
	}

	app.RunAndExitOnError()
}

func osExpandAndVerify(c *cli.Context, name string) (string, error) {
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

func run(c *cli.Context) {
	log.Info("Running GoSuperviseWindows!")
	user, err := osExpandAndVerify(c, "user")
	if err != nil {
		fail(err)
	}
	password, err := osExpandAndVerify(c, "password")
	if err != nil {
		fail(err)
	}
	host, err := osExpandAndVerify(c, "host")
	if err != nil {
		fail(err)
	}
	port, err := osExpandAndVerify(c, "port")
	if err != nil {
		fail(err)
	}
	portInt, err := strconv.Atoi(port)

	command, err := osExpandAndVerify(c, "command")
	if err != nil {
		fail(err)
	}
	err = remoteWinRmCommand(user, password, host, portInt, command)
	if err != nil {
		log.Err("Failed: %v", err)
	}
}

func remoteWinRmCommand(user string, password string, host string, port int, cmd string) error {

	client, err := winrm.NewClient(&winrm.Endpoint{Host: "localhost", Port: 5985, HTTPS: false, Insecure: false}, user, password)
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
