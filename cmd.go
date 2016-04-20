package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func RunCmd(dir, command string, args ...string) (string, []byte, error) {
	msg := fmt.Sprintf(" [%s] %s %s", dir, command, strings.Join(args, " "))
	if verbose {
		fmt.Println(msg)
	}
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return msg, output, err
}

func MustRunCmd(dir, command string, args ...string) []byte {
	msg, output, err := RunCmd(dir, command, args...)
	if err != nil {
		if !verbose {
			fmt.Println(msg)
		}
		ErrorExit(string(output))
	}
	return output
}
