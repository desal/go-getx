package getx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/desal/cmd"
	"github.com/desal/dsutil"
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
	donePkgs   set
	install    bool
	scanMode   ScanMode
	output     cmd.Output
	goPath     []string
	cmdCtx     *cmd.Context
	gitCtx     *git.Context
	ruleSet    RuleSet
	applyHooks bool
	taggedOnly bool
}

func NewContext(install bool, scanMode ScanMode, output cmd.Output, goPath []string, ruleSet RuleSet, applyHooks, taggedOnly bool) *Context {
	return &Context{
		donePkgs:   set{},
		install:    install,
		scanMode:   scanMode,
		output:     output,
		goPath:     goPath,
		cmdCtx:     cmd.NewContext(".", output),
		gitCtx:     git.New(output),
		ruleSet:    ruleSet,
		applyHooks: applyHooks,
		taggedOnly: taggedOnly,
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

func stringInSlice(slice []string, s string) bool {
	for _, e := range slice {
		if s == e {
			return true
		}
	}
	return false
}

func (c *Context) goToMostRecentTag(goDir string) {
	tags, err := c.gitCtx.Tags(goDir, false)
	if err != nil {
		c.output.Warning("Failed to get tags %s:\n%s", goDir, err.Error())
		return
	}

	mostRecentTag, err := c.gitCtx.MostRecentTag(goDir, false)
	if err != nil {
		c.output.Warning("Failed to get most recent tag %s:\n%s", goDir, err.Error())
		return
	} else if mostRecentTag == "" {
		c.output.Warning("Package %s has no tags", goDir)
		return
	}

	if !stringInSlice(tags, mostRecentTag) {
		//this will disconnect head...
		err = c.gitCtx.Checkout(goDir, mostRecentTag, false)
		if err != nil {
			c.output.Warning("Failed to checkout tag %s (%s):\n%s", goDir, mostRecentTag, err.Error())
			return
		}
	}

}

func (c *Context) runHook(ctx *cmd.Context, goDir string, filename string) error {
	if !c.applyHooks {
		return nil
	}

	hook, err := dsutil.SanitisePath(ctx, filepath.Join(goDir, filename))
	if err != nil {
		err := fmt.Errorf("Failed to sanitise path: %s", err.Error())
		c.output.Error(err.Error())
		return err
	}

	if !dsutil.CheckPath(hook) {
		return nil
	}

	output, err := ctx.ShellExecf("%s", hook)
	if err != nil {
		return fmt.Errorf("Hook Error: %s\n%s", err.Error(), output)
	}

	return nil
}

//

func (c *Context) Get(workingDir, pkg string, depsOnly, tests bool) {
	goCtx := gocmd.New(c.output, c.goPath) //TODO why am i creating this again?
	goDir, alreadyExists := goCtx.Dir(workingDir, pkg)

	hookContext := cmd.NewContext(goDir, c.output) // TODO flags?

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
			c.output.Error("Failed to clone %s:\n%s", gitUrl, err.Error())
			os.Exit(1)

		}

		if c.taggedOnly {
			c.goToMostRecentTag(goDir)
		}

		c.donePkgs[pkg] = empty{}
		c.donePkgs[rootPkg] = empty{}

	} else if c.scanMode == ScanMode_Update || c.scanMode == ScanMode_DeepFetch {
		gitTopLevel, err := c.gitCtx.TopLevel(goDir, false)
		if err != nil {
			c.donePkgs[pkg] = empty{}
			return
		}

		var rootPkg string
		{
			saneDir, err := dsutil.SanitisePath(c.cmdCtx, goDir)
			if err != nil {
				panic(err)
			}

			if !strings.HasSuffix(saneDir, pkg) {
				panic("pkg not in path")
			}
			srcPath := strings.TrimSuffix(saneDir, pkg)

			if !strings.HasPrefix(gitTopLevel, srcPath) {
				panic("git top level not in src")
			}
			rootPkg = strings.TrimPrefix(gitTopLevel, srcPath)
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
			if err != nil {
				c.output.Error("Error running git status on %s", goDir)
				goto finishUpdate
			}

			if gitStatus != git.Clean {
				c.output.Warning("Skipping %s, git status is %s", goDir, gitStatus.String())
				goto finishUpdate
			}

			err = c.gitCtx.Checkout(goDir, "master", false)
			if err != nil {
				c.output.Warning("Couldn't checkout %s master:\n%s", goDir, err.Error())
				goto finishUpdate
			}

			err = c.runHook(hookContext, goDir, "get-before-pull.sh")
			if err != nil {
				goto finishUpdate
			}

			err = c.gitCtx.Pull(goDir, false)
			if err != nil {
				c.output.Warning("Couldn't pull %s:\n%s", goDir, err.Error())
				goto finishUpdate
			}

			if c.taggedOnly {
				c.goToMostRecentTag(goDir)
			}
		}
	finishUpdate:
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

	//Before install runs even if we don't install, Don't install if hook fails.
	err := c.runHook(hookContext, goDir, "get-before-install.sh")
	if err != nil || !c.install {
		return
	}

	//attempt to install everything; takes advantage of multiple cores
	//but will bomb out if some of the sub pkgs are particularly broken
	//if that happens, attempt installing one by one instead
	err = goCtx.Install(workingDir, pkg+"/...")
	if err != nil {
		for importPath, _ := range list {
			goCtx.Install(workingDir, importPath)
		}
	}

	//Post install always runs, even if errors (for the case where a subpkg fails)
	c.runHook(hookContext, goDir, "get-after-install.sh")
}
