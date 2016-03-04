package utils

// this is code taken from the v2 branch of github.com/deis/deis/tests/utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// RunCommandWithStdoutStderr execs a command and returns its output.
func RunCommandWithStdoutStderr(cmd *exec.Cmd) (bytes.Buffer, bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	stderrPipe, err := cmd.StderrPipe()
	stdoutPipe, err := cmd.StdoutPipe()

	cmd.Env = os.Environ()
	if err != nil {
		fmt.Println("error at io pipes")
	}

	err = cmd.Start()
	if err != nil {
		fmt.Println("error at command start")
	}

	go func() {
		streamOutput(stdoutPipe, &stdout, os.Stdout)
	}()
	go func() {
		streamOutput(stderrPipe, &stderr, os.Stderr)
	}()
	err = cmd.Wait()
	if err != nil {
		fmt.Println("error at command wait")
	}
	return stdout, stderr, err
}

// streamOutput from a source to a destination buffer while also printing
func streamOutput(src io.Reader, dst *bytes.Buffer, out io.Writer) error {
	mw := io.MultiWriter(dst, out)
	s := bufio.NewReader(src)
	for {
		var line []byte
		line, err := s.ReadSlice('\n')
		if err == io.EOF && len(line) == 0 {
			break // done
		}
		if err == io.EOF {
			return fmt.Errorf("Improper termination: %v", line)
		}
		if err != nil {
			return err
		}

		// append to the buffer and out at once
		mw.Write(line)
	}

	return nil
}
