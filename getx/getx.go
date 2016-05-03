package getx

import (
	"os"
	"strings"

	"github.com/desal/cmd"
	"github.com/desal/git"
	"github.com/desal/gocmd"
)

type empty struct{}
type set map[string]empty
type ScanMode int

const (
	ScanMode_Skip      ScanMode = iota //See a folder, assume it's good
	ScanMode_DeepFetch                 //See a folder, check recursively for missing deps
	ScanMode_Update                    //Update all deps we see if possible
)

type Context struct {
	donePkgs set
	install  bool
	scanMode ScanMode
	output   cmd.Output
	goPath   []string
	gitCtx   *git.Context
	ruleSet  RuleSet
}

func NewContext(install bool, scanMode ScanMode, output cmd.Output, goPath []string, ruleSet RuleSet) *Context {
	return &Context{
		donePkgs: set{},
		install:  install,
		scanMode: scanMode,
		output:   output,
		goPath:   goPath,
		gitCtx:   git.New(output),
		ruleSet:  ruleSet,
	}
}

func pkgContains(parent, child string) bool {
	if parent == child || strings.HasPrefix(child, parent+"/") {
		return true
	}
	return false
}

func (c *Context) AlreadyDone(pkg string) bool {
	for donePkg, _ := range c.donePkgs {
		if pkgContains(donePkg, pkg) {
			return true
		}
	}
	return false
}

func (c *Context) Get(workingDir, pkg string, depsOnly, tests bool) {
	goCtx := gocmd.New(c.output, c.goPath) //TODO why am i creating this again?
	goDir, alreadyExists := goCtx.Dir(workingDir, pkg)

	if depsOnly && !alreadyExists {
		c.output.Error("Can not get dependencies only for package %s, package does not exist", pkg)
		os.Exit(1)
	} else if depsOnly {
		//Can do nothing
		c.donePkgs[pkg] = empty{}
	} else if !alreadyExists {
		rootPkg, gitUrl, err := c.ruleSet.GetUrl(pkg)
		if c.AlreadyDone(rootPkg) {
			c.donePkgs[pkg] = empty{}
			return
		}
		if err != nil {
			c.output.Error(err.Error())
			os.Exit(1)
		}
		if rootPkg != pkg {
			c.Get(workingDir, rootPkg, depsOnly, tests)
			return
		}

		err = c.gitCtx.Clone(goDir, gitUrl, false)
		if err != nil {
			c.donePkgs[pkg] = empty{}
			c.donePkgs[rootPkg] = empty{}
			return
		}
		c.donePkgs[pkg] = empty{}
		c.donePkgs[rootPkg] = empty{}

	} else if c.scanMode == ScanMode_Update || c.scanMode == ScanMode_DeepFetch {
		gitTopLevel, err := c.gitCtx.TopLevel(goDir, false)
		if err != nil {
			c.donePkgs[pkg] = empty{}
			return
		}

		list, err := goCtx.List(gitTopLevel, "")
		if err != nil {
			c.donePkgs[pkg] = empty{}
			return
		}

		if len(list) != 1 {
			panic("go list with no args should return a single package")
		}

		var rootPkg string
		for e, _ := range list { //jump into the only element
			rootPkg = e
		}

		if c.AlreadyDone(rootPkg) {
			c.donePkgs[pkg] = empty{}
			return
		} else if rootPkg != pkg {
			c.Get(workingDir, rootPkg, depsOnly, tests)
			c.donePkgs[pkg] = empty{}
			return
		}

		if c.scanMode == ScanMode_Update {
			gitStatus, err := c.gitCtx.Status(goDir, false)
			if err == nil && gitStatus == git.Clean {
				c.gitCtx.Pull(goDir, false)
			}
		}

		c.donePkgs[rootPkg] = empty{}
		c.donePkgs[pkg] = empty{}

	} else if c.scanMode == ScanMode_Skip {
		c.donePkgs[pkg] = empty{}
		return
	}

	list, _ := goCtx.List(workingDir, pkg+"/...")
	for _, e := range list {
		//Only check imports, because the recursive nature of this tool
		//will get the transisitive dependencies.
		var imports []interface{}
		var testImports []interface{}
		if importsInt, ok := e["Imports"]; ok {
			imports = importsInt.([]interface{})
		}

		if testImportsInt, ok := e["TestImports"]; ok {
			testImports = testImportsInt.([]interface{})
		}

		for _, impInt := range imports {
			imp := impInt.(string)
			if !goCtx.IsStdLib(imp) && !c.AlreadyDone(imp) {
				c.Get(workingDir, imp, false, false)
			}
		}
		if tests {
			for _, impInt := range testImports {
				imp := impInt.(string)
				if !goCtx.IsStdLib(imp) && !c.AlreadyDone(imp) {
					c.Get(workingDir, imp, false, false)
				}
			}
		}
	}
	if c.install {
		//attempt to install everything; takes advantage of multiple cores
		//but will bomb out if some of the sub pkgs are particularly broken
		//if that happens, attempt installing one by one instead
		err := goCtx.Install(workingDir, pkg+"/...")
		if err != nil {
			for importPath, _ := range list {
				goCtx.Install(workingDir, importPath)
			}
		}

	}
}
