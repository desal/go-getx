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
)

type Package struct {
	ImportPath        string
	Imports           []string
	hookBeforePull    string
	hookBeforeInstall string
	hookAfterInstall  string
	gitIgnore         string
}

type Repo struct {
	BasePath string
	Packages []Package
}

type Repos struct {
	output cmd.Output
	Repos  []Repo
}

func NewRepos(output cmd.Output) Repos { return Repos{output: output} }

func Pkg(importPath string, imports ...string) Package {
	return Package{importPath, imports, "", "", ""}
}

func (r *Repos) AddRepo(basePath string, packages ...Package) {
	r.Repos = append(r.Repos, Repo{basePath, packages})
}

func (r *Repos) Test(f func(goPath []string, ruleSet RuleSet)) set {
	gitDir, err := ioutil.TempDir("", "gogetx_test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(gitDir)

	rules := []Rule{}
	for _, repo := range r.Repos {
		rules = append(rules, MockPackageBareGit(gitDir, repo, r.output))
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
	MockEnv(mockGoPath, RuleSet{rules}, f)

	cmdCtx := cmd.NewContext(mockGoPath, r.output, cmd.Must)

	res, _ := cmdCtx.Execf("find . -not ( -type d -name .git -prune ) -type f")

	res_noarch := strings.Replace(res, "./pkg/"+runtime.GOOS+"_"+runtime.GOARCH, "./pkg", -1)
	res_noexe := strings.Replace(res_noarch, ".exe", "", -1)
	res_clean := dsutil.SplitLines(res_noexe, true)

	result := set{}
	for _, e := range res_clean {
		result[e] = empty{}
	}
	return result
}

func mockPackage(dir, pkgName string, imports []string) {
	os.MkdirAll(dir, 0644)
	f, err := os.OpenFile(filepath.Join(dir, "gen.go"), os.O_CREATE|os.O_TRUNC, 0644)
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
func MockPackageBareGit(rootDir string, repo Repo, output cmd.Output) Rule {
	escapedPkg := strings.Replace(repo.BasePath, "/", "_", -1)
	barePath := filepath.Join(rootDir, escapedPkg+".git")
	repoPath := filepath.Join(rootDir, escapedPkg)
	os.MkdirAll(barePath, 0644)

	bareCtx := cmd.NewContext(barePath, output, cmd.Must)
	rootCtx := cmd.NewContext(rootDir, output, cmd.Must)
	repoCtx := cmd.NewContext(repoPath, output, cmd.Must)

	bareCtx.Execf("git --bare init")
	rootCtx.Execf("git clone %s.git", escapedPkg)

	for _, pkg := range repo.Packages {
		//Check the package actually belongs in the repo
		if !strings.HasPrefix(pkg.ImportPath, repo.BasePath) {
			panic(fmt.Sprintf("%s is not part of repo %s", pkg.ImportPath, repo.BasePath))
		}
		relativePkg := pkg.ImportPath[len(repo.BasePath):]
		mockPackage(filepath.Join(repoPath, relativePkg), pkg.ImportPath, pkg.Imports)
		if pkg.gitIgnore != "" {
			gitIgnore, err := os.OpenFile(filepath.Join(repoPath, ".gitignore"), os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				panic(err)
			}
			fmt.Fprint(gitIgnore, pkg.gitIgnore)
			gitIgnore.Close()
		}

		if pkg.hookBeforePull != "" {
			hookBeforePull, err := os.OpenFile(filepath.Join(repoPath, "get-before-pull.sh"), os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				panic(err)
			}
			fmt.Fprint(hookBeforePull, pkg.hookBeforePull)
			hookBeforePull.Close()
		}

		if pkg.hookBeforeInstall != "" {
			hookBeforeInstall, err := os.OpenFile(filepath.Join(repoPath, "get-before-install.sh"), os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				panic(err)
			}
			fmt.Fprint(hookBeforeInstall, pkg.hookBeforeInstall)
			hookBeforeInstall.Close()
		}

		if pkg.hookAfterInstall != "" {
			hookAfterInstall, err := os.OpenFile(filepath.Join(repoPath, "get-after-install.sh"), os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				panic(err)
			}
			fmt.Fprint(hookAfterInstall, pkg.hookAfterInstall)
			hookAfterInstall.Close()
		}

	}

	repoCtx.Execf("git add -A")
	repoCtx.Execf(`git commit -m "init"`)
	repoCtx.Execf("git push")

	return NewRule(repo.BasePath, barePath)
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
