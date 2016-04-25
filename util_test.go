package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Package struct {
	ImportPath string
	Imports    []string
}

type Repo struct {
	BasePath string
	Packages []Package
}

type Repos struct {
	Repos []Repo
}

func Pkg(importPath string, imports ...string) Package {
	return Package{importPath, imports}
}

func (r *Repos) AddRepo(basePath string, packages ...Package) {
	r.Repos = append(r.Repos, Repo{basePath, packages})
}

func (r *Repos) Test(f func()) {
	gitDir, err := ioutil.TempDir("", "gogetx_test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(gitDir)

	rules := []rule{}
	for _, repo := range r.Repos {
		rules = append(rules, MockPackageBareGit(gitDir, repo))
	}

	mockGoPath, err := ioutil.TempDir("", "gogetx_test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(mockGoPath)

	err = os.MkdirAll(filepath.Join(mockGoPath, "src"), 0644)
	if err != nil {
		panic(err)
	}
	MockEnv(mockGoPath, rules, f)
}

func mockPackage(dir, pkgName string, imports []string) {
	os.MkdirAll(dir, 0644)
	f, err := os.OpenFile(filepath.Join(dir, "gen.go"), os.O_CREATE|os.O_TRUNC|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintln(f, "package", path.Base(pkgName))
	for _, pkg := range imports {
		fmt.Fprintf(f, `import _ "%s"`, pkg)
		fmt.Fprintf(f, "\n")
	}
}

//creates a mocked package in a bare git repo.
func MockPackageBareGit(rootDir string, repo Repo) rule {
	escapedPkg := strings.Replace(repo.BasePath, "/", "_", -1)
	barePath := filepath.Join(rootDir, escapedPkg+".git")
	repoPath := filepath.Join(rootDir, escapedPkg)
	os.MkdirAll(barePath, 0644)
	MustRunCmd(barePath, "git", "--bare", "init")
	MustRunCmd(rootDir, "git", "clone", escapedPkg+".git")

	for _, pkg := range repo.Packages {
		//Check the package actually belongs in the repo
		if !strings.HasPrefix(pkg.ImportPath, repo.BasePath) {
			panic(fmt.Sprintf("%s is not part of repo %s", pkg.ImportPath, repo.BasePath))
		}
		relativePkg := pkg.ImportPath[len(repo.BasePath):]
		mockPackage(filepath.Join(repoPath, relativePkg), pkg.ImportPath, pkg.Imports)
	}
	MustRunCmd(repoPath, "git", "add", "-A")
	MustRunCmd(repoPath, "git", "commit", "-m", `"init"`)
	MustRunCmd(repoPath, "git", "push")

	return NewRule(repo.BasePath, barePath)
}

func MockEnv(mockGoPath string, mockRules []rule, f func()) {
	//This clearly shows the pain i've caused myself by hanging onto globals/directly fetching environment.
	//TODO split out the "Go()" func so that this is all encapsulated, and remove this function.
	goPathActual := goPath
	rulesActual := rules
	wdActual, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	goPath = []string{mockGoPath}
	os.Setenv("GOPATH", strings.Join(goPath, ":"))
	rules = mockRules
	err = os.Chdir(filepath.Join(mockGoPath, "src"))
	if err != nil {
		panic(err)
	}

	f()

	goPath = goPathActual
	os.Setenv("GOPATH", strings.Join(goPath, ":"))
	rules = rulesActual
	err = os.Chdir(wdActual)
	if err != nil {
		panic(err)
	}
}
