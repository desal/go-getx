package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/desal/cmd"
	"github.com/desal/dsutil"
	"github.com/desal/go-getx/getx"
	"github.com/desal/gocmd"
	"github.com/desal/richtext"
	"github.com/jawher/mow.cli"
)

//Example config:
//a/hats=http://server/special_repos/hats.git
//a/([^/]+)=http://server/repos/$1.git
//b/([^/]+)=http://other/repos/$1.git

func main() {
	app := cli.App("go-getx", "go get extended")

	app.Spec = "[-d] [-v] [-i] [-f | -u] [-t] [PKG...]"

	var (
		dependencies = app.BoolOpt("d deps-only", false, "Do not fetch named packages, only their dependencies")
		verbose      = app.BoolOpt("v verbose", false, "Verbose output")
		install      = app.BoolOpt("i install", false, "Install all fetched packages (will continue if package fails to compile)")
		fetch        = app.BoolOpt("f fetch-missing", false, "Performs a deep search for any missing dependencies and fetches them")
		update       = app.BoolOpt("u update", false, "Updates package, and all transisitive depnediencs where possible")
		tests        = app.BoolOpt("t tests", false, "Fetches tests for the named packages")

		pkgs = app.StringsArg("PKG", nil, "Packages")
	)

	app.Action = func() {
		if len(*pkgs) == 0 {
			app.PrintHelp()
			os.Exit(0)
		}

		ruleSet, err := getx.LoadRulesFromFile(filepath.Join(dsutil.UserHomeDir(), ".go-getx-map"))
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		output := cmd.NewStdOutput(*verbose, richtext.Ansi())

		var scanMode getx.ScanMode
		if *fetch {
			scanMode = getx.ScanMode_DeepFetch
		} else if *update {
			scanMode = getx.ScanMode_Update
		} else {
			scanMode = getx.ScanMode_Skip
		}

		goPath := gocmd.FromEnv(output)
		ctx := getx.NewContext(*install, scanMode, output, goPath, ruleSet)
		for _, pkg := range *pkgs {
			ctx.Get(".", pkg, *dependencies, *tests, false)
		}
	}

	app.Run(os.Args)
}
