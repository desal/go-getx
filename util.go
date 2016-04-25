package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func SplitLines(s string) []string {
	split := strings.Split(s, "\n")
	for i, s := range split {
		split[i] = strings.Replace(s, "\r", "", -1)
	}
	return split
}

func FirstLine(s string) string {
	return strings.Replace(strings.Split(s, "\n")[0], "\r", "", -1)
}

func SanitisePath(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}
	result := filepath.ToSlash(FirstLine(MustRunCmd(".", "cygpath", "-w", path)))
	fmt.Println(result)
	return result
}
