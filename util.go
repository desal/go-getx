package main

import (
	"fmt"
	"os"
	"runtime"
)

func ErrorExit(output string) {
	fmt.Println("Error")
	fmt.Println("Output:")
	fmt.Println(output)
	os.Exit(1)
}

func CheckPath(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
