package main

import (
	"fmt"
	"os"
)

//Example config:
//a/hats=http://server/special_repos/hats.git
//a/([^/]+)=http://server/repos/$1.git
//b/([^/]+)=http://other/repos/$1.git

var (
	escapeWindows = false
	escapeChecked = false
	verbose       = false
)

func Get(workingDir, pkg string, depsOnly, install, tests, update bool, gotten map[string]struct{}) {
	importPath, alreadyExists := GoDir(workingDir, pkg)
	var rootPkg string

	if depsOnly {
		if !alreadyExists {
			fmt.Printf("ERROR: Can't get package %s with -d, pkg does not exist\n", pkg)
			os.Exit(1)
		}
		rootPkg = pkg

	} else {
		if !alreadyExists {
			var gitUrl string
			var err error
			rootPkg, gitUrl, err = GetUrl(pkg)
			if _, got := gotten[rootPkg]; got {
				gotten[pkg] = struct{}{}
				return
			}
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			rootPkgPath := GoPathPkg(rootPkg)
			GitClone(rootPkgPath, gitUrl)

		} else if update {
			var err error

			gitTopLevel := GitTopLevel(importPath)
			rootPkg, err = GoName(gitTopLevel)
			if err != nil {
				panic(err)
			}
			if _, got := gotten[rootPkg]; got {
				gotten[pkg] = struct{}{}
				return
			}

			gitStatus := CheckGitStatus(importPath)
			if gitStatus != GitStatus_Clean {
				gotten[pkg] = struct{}{}
				return
			}

			GitPull(importPath)
		} else { //alreadyExists, don't need to update
			gotten[pkg] = struct{}{}
			return
		}
	}

	gotten[rootPkg] = struct{}{}
	importPath, alreadyExists = GoDir(workingDir, pkg)
	if !alreadyExists {
		panic("Fetch without error, but no output")
	}

	for dep, _ := range GoDeps(rootPkg, tests) {
		if GoIsStdLib(dep) {
			continue
		}
		_, got := gotten[dep]
		if !got {
			Get(importPath, dep, false, install, false, update, gotten)
		} else {
		}
	}

	if install {
		_ = MustRunCmd(workingDir, "go", "install", rootPkg+"/...")
	}
}

func Usage() {

	fmt.Printf(`
Usage: go-getx [option...] packages

Options
  -d, --dependencies-only
             Will only fetch dependencies, does not try to fetch the named
             packages themselves.

  -v, --verbose
             Verbose

  -i, --install
             Runs go install ./... after git checkout

  -t, --tests
             Fetches deps required to run tests

  -u, --update
             Update the named packages and dependencies. By default go-getx
             will only get missing packages.
`)
}

func main() {
	if len(os.Args) == 1 {
		Usage()
		os.Exit(0)
	}
	err := LoadRules()
	if err != nil {
		fmt.Println("ERROR Failed to load rules")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	//command line args
	var pkgs []string

	depsOnly := false
	install := false
	tests := false
	update := false

	for _, arg := range os.Args[1:] {
		switch arg {
		case "-d":
			fallthrough
		case "--dependencies-only":
			depsOnly = true
		case "-v":
			fallthrough
		case "--verbose":
			verbose = true
		case "-i":
			fallthrough
		case "--install":
			install = true
		case "-t":
			fallthrough
		case "--tests":
			tests = true
		case "-u":
			fallthrough
		case "--update":
			update = true
		default:
			pkgs = append(pkgs, arg)
		}
	}
	if len(pkgs) == 0 && depsOnly {
		pkgs = append(pkgs, ".")
	} else if len(pkgs) == 0 {
		Usage()
		os.Exit(1)
	}

	gotten := map[string]struct{}{}
	for _, pkg := range pkgs {
		Get(".", pkg, depsOnly, install, tests, update, gotten)
	}
}
