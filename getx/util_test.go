package getx

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/desal/cmd"
	"github.com/desal/dsutil"
	"github.com/desal/richtext"
)

type Package struct {
	ImportPath string
	Imports    []string
}

type Repo struct {
	BasePath          string
	Packages          []Package
	hookBeforeUpdate  string
	hookBeforeInstall string
	hookAfterInstall  string
	gitIgnore         string
}

type Repos struct {
	format richtext.Format
	Repos  []Repo
}

func NewRepos(format richtext.Format) Repos { return Repos{format: format} }

func Pkg(importPath string, imports ...string) Package {
	return Package{ImportPath: importPath, Imports: imports}
}

func (r *Repos) AddRepo(basePath string, packages ...Package) *Repo {
	r.Repos = append(r.Repos, Repo{BasePath: basePath, Packages: packages})
	return &r.Repos[len(r.Repos)-1]
}

func (r *Repos) Test(f func(goPath []string, ruleSet RuleSet)) stringSet {
	gitDir, err := ioutil.TempDir("", "gogetx_test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(gitDir)

	rules := []Rule{}
	for _, repo := range r.Repos {
		rules = append(rules, MockPackageBareGit(gitDir, repo, r.format))
	}

	mockGoPath, err := ioutil.TempDir("", "gogetx_test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(mockGoPath)

	err = os.MkdirAll(filepath.Join(mockGoPath, "src"), 0755)
	if err != nil {
		panic(err)
	}
	MockEnv(mockGoPath, RuleSet{rules}, f)

	cmdCtx := cmd.New(mockGoPath, r.format, cmd.Warn)

	o, _, err := cmdCtx.Execf(`find . -not \( -type d -name .git -prune \) -type f`)
	if err != nil {
		panic(err)
	}
	res_noarch := strings.Replace(o, "./pkg/"+runtime.GOOS+"_"+runtime.GOARCH, "./pkg", -1)
	res_noexe := strings.Replace(res_noarch, ".exe", "", -1)
	res_clean := dsutil.SplitLines(res_noexe, true)

	result := stringSet{}
	for _, e := range res_clean {
		result[e] = empty{}
	}
	return result
}

func mockPackage(dir, pkgName string, imports []string) {
	os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(filepath.Join(dir, "gen.go"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
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

func mockFile(dir, filename, contents string) {
	if contents == "" {
		return
	}

	os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(filepath.Join(dir, filename), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprint(f, contents)
}

//creates a mocked package in a bare git repo.
func MockPackageBareGit(rootDir string, repo Repo, format richtext.Format) Rule {
	escapedPkg := strings.Replace(repo.BasePath, "/", "_", -1)
	barePath := filepath.Join(rootDir, escapedPkg+".git")
	repoPath := filepath.Join(rootDir, escapedPkg)
	os.MkdirAll(barePath, 0755)

	bareCtx := cmd.New(barePath, format, cmd.Warn)
	rootCtx := cmd.New(rootDir, format, cmd.Warn)
	repoCtx := cmd.New(repoPath, format, cmd.Warn)

	bareCtx.Execf("git --bare init")
	_, _, err := rootCtx.Execf("git clone %s.git", dsutil.PosixPath(escapedPkg))
	if err != nil {
		panic(err)
	}

	for _, pkg := range repo.Packages {
		//Check the package actually belongs in the repo
		if !strings.HasPrefix(pkg.ImportPath, repo.BasePath) {
			panic(fmt.Sprintf("%s is not part of repo %s", pkg.ImportPath, repo.BasePath))
		}
		relativePkg := pkg.ImportPath[len(repo.BasePath):]
		mockPackage(filepath.Join(repoPath, relativePkg), pkg.ImportPath, pkg.Imports)
	}
	mockFile(repoPath, "get-before-update.sh", repo.hookBeforeUpdate)
	mockFile(repoPath, "get-before-install.sh", repo.hookBeforeInstall)
	mockFile(repoPath, "get-after-install.sh", repo.hookAfterInstall)
	mockFile(repoPath, ".gitignore", repo.gitIgnore)

	repoCtx.Execf("git add -A")
	repoCtx.Execf(`git commit -m "init"`)
	repoCtx.Execf("git push")

	return NewRule(repo.BasePath, dsutil.PosixPath(barePath))
}

func MockEnv(mockGoPath string, ruleSet RuleSet, f func(goPath []string, ruleSet RuleSet)) {
	//This clearly shows the pain i've caused myself by hanging onto globals/directly fetching environment.
	//TODO split out the "Go()" func so that this is all encapsulated, and remove this function.
	wdActual, err := os.Getwd()
	goPathActual := os.Getenv("GOPATH")
	if err != nil {
		panic(err)
	}

	os.Setenv("GOPATH", mockGoPath)
	err = os.Chdir(filepath.Join(mockGoPath, "src"))
	if err != nil {
		panic(err)
	}

	f([]string{mockGoPath}, ruleSet)

	err = os.Chdir(wdActual)
	if err != nil {
		panic(err)
	}
	os.Setenv("GOPATH", goPathActual)
}
