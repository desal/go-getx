package main

import (
	"fmt"
	"os"
)

type GitStatus int

const (
	GitStatus_Clean       = 1
	GitStatus_Uncommitted = 2
	GitStatus_Detached    = 3
	GitStatus_Unpushed    = 4
	GitStatus_NoUpstream  = 5
)

func (g GitStatus) String() string {
	switch g {
	case GitStatus_Clean:
		return "GitStatus_Clean"
	case GitStatus_Uncommitted:
		return "GitStatus_Uncommitted"
	case GitStatus_Detached:
		return "GitStatus_Detached"
	case GitStatus_Unpushed:
		return "GitStatus_Unpushed"
	case GitStatus_NoUpstream:
		return "GitStatus_NoUpstream"
	default:
		return "GitStatus_Unknown"
	}
}

func CheckGitStatus(gitPath string) GitStatus {
	unchecked := MustRunCmd(gitPath, "git", "status", "--porcelain")
	if len(unchecked) != 0 {
		return GitStatus_Uncommitted
	}

	//Detached head check
	if _, err := RunCmd(gitPath, "git", "symbolic-ref", "HEAD"); err != nil {
		return GitStatus_Detached
	}

	//Some (but not all) versions of git on windows like the curly brackets escaped.
	//Rather than attempting to decipher this from the version of git installed, I just retry the first time this
	//runs with an escape, if that works, then we switch over for remaining invokations
	if !escapeChecked {
		_, err := RunCmd(gitPath, "git", "rev-parse", "@{0}")
		if err != nil {
			_, err := RunCmd(gitPath, "git", "rev-parse", `@\{0\}`)
			if err != nil {
				fmt.Printf("Error\n")
				fmt.Printf("Could not determine if git curly braces need to be escaped\n")
				fmt.Printf("git rev-parse @{0} failed both with and without quotes in %s\n", gitPath)
				os.Exit(1)
			} else {
				escapeWindows = true
			}
		}
		escapeChecked = true
	}

	var err error
	if escapeWindows {
		_, err = RunCmd(gitPath, "git", "rev-parse", "--abrev-ref", "--symbolic-full-name", `@\{upstream\}`)
	} else {
		_, err = RunCmd(gitPath, "git", "rev-parse", "--abrev-ref", "--symbolic-full-name", "@{upstream}")
	}
	if err != nil {
		return GitStatus_NoUpstream
	}

	var unsynced string
	if escapeWindows {
		unsynced = MustRunCmd(gitPath, "git", "rev-list", `HEAD@\{upstream\}..HEAD`)
	} else {
		unsynced = MustRunCmd(gitPath, "git", "rev-list", `HEAD@{upstream}..HEAD`)
	}

	if len(unsynced) == 0 {
		return GitStatus_Clean
	}

	return GitStatus_Unpushed
}

func GitTopLevel(gitPath string) string {
	return SanitisePath(FirstLine(MustRunCmd(gitPath, "git", "rev-parse", "--show-toplevel")))
}

//These are just straight git wrappers
func GitClone(targetDir, url string) error {
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		return err
	}

	_ = MustRunCmd(targetDir, "git", "clone", url, ".")
	return nil
}

func GitPull(targetDir string) error {
	_ = MustRunCmd(targetDir, "git", "pull")
	return nil
}
