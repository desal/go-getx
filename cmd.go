package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func runCmd(dir, command string, args []string) (string, string, error) {
	msg := fmt.Sprintf(" [%s] %s %s", dir, command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return msg, string(output), err
}

func RunCmd(dir, command string, args ...string) (string, error) {
	msg, output, err := runCmd(dir, command, args)
	if verbose {
		fmt.Println(msg)
	}
	return output, err
}

func MustRunCmd(dir, command string, args ...string) string {
	msg, output, err := runCmd(dir, command, args)
	if err != nil || verbose {
		fmt.Println(msg)
	}
	if err != nil {
		ErrorExit(output)
	}

	return output
}
