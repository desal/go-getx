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
	"github.com/desal/richtext"
)

//go:generate stringer -type Flag

type (
	empty     struct{}
	Flag      int
	flagSet   map[Flag]empty
	stringSet map[string]empty

	Context struct {
		doneGit     stringSet
		doneGo      stringSet
		format      richtext.Format
		goPath      []string
		cmdCtx      *cmd.Context
		gitCtx      *git.Context
		goCtx       *gocmd.Context
		ruleSet     RuleSet
		flags       flagSet
		gitTopCache map[string]string
	}
)

const (
	_          Flag = iota
	DeepScan        // When traversing folders, iteratively scan into present folders
	Update          // Update any found dependency to latest version
	Install         //
	Warn            //
	MustExit        //
	MustPanic       //
	Verbose         //
	CmdVerbose      // Also displays commands being executed
	ApplyHooks      //
	TaggedOnly      //
	RecurseTopLevel
)

func (fs flagSet) Checked(flag Flag) bool {
	_, ok := fs[flag]

	return ok
}

func New(format richtext.Format, goPath []string, ruleSet RuleSet, buildFlags string, flags ...Flag) *Context {
	c := &Context{
		doneGit:     stringSet{},
		doneGo:      stringSet{},
		format:      format,
		goPath:      goPath,
		ruleSet:     ruleSet,
		flags:       flagSet{},
		gitTopCache: map[string]string{},
	}

	cmdFlags := []cmd.Flag{cmd.Strict}
	var gitFlags []git.Flag
	var goFlags []gocmd.Flag

	for _, flag := range flags {
		c.flags[flag] = empty{}
		switch flag {
		case MustPanic:
			cmdFlags = append(cmdFlags, cmd.MustPanic)
			gitFlags = append(gitFlags, git.MustPanic)
		case MustExit:
			cmdFlags = append(cmdFlags, cmd.MustExit)
			gitFlags = append(gitFlags, git.MustExit)
		case Warn:
			cmdFlags = append(cmdFlags, cmd.Warn)
			gitFlags = append(gitFlags, git.Warn)
		case Verbose:
		case CmdVerbose:
			goFlags = append(goFlags, gocmd.Warn)
			cmdFlags = append(cmdFlags, cmd.Verbose)
			gitFlags = append(gitFlags, git.Verbose)
			goFlags = append(goFlags, gocmd.Verbose)
		}
	}

	if !c.flags.Checked(RecurseTopLevel) && !c.flags.Checked(DeepScan) {
		panic("Atleast one of RecurseTopLevel or DeepScan must be provided")
	}

	c.cmdCtx = cmd.New(".", format, cmdFlags...)
	c.gitCtx = git.New(format, gitFlags...)
	c.goCtx = gocmd.New(format, goPath, "", buildFlags, goFlags...)

	return c
}

func (c *Context) errorf(s string, a ...interface{}) error {
	if c.flags.Checked(MustExit) {
		c.format.ErrorLine(s, a...)
		os.Exit(1)
	} else if c.flags.Checked(MustPanic) {
		panic(fmt.Errorf(s, a...))
	} else if c.flags.Checked(Warn) || c.flags.Checked(Verbose) {
		c.format.WarningLine(s, a...)
	}
	return fmt.Errorf(s, a...)
}

func (c *Context) warnf(s string, a ...interface{}) {
	if c.flags.Checked(Warn) || c.flags.Checked(Verbose) {
		c.format.WarningLine(s, a...)
	}
}

func (c *Context) verbosef(s string, a ...interface{}) {
	if c.flags.Checked(Verbose) {
		c.format.PrintLine(s, a...)
	}
}

func pkgContains(parent, child string) bool {
	if parent == child || strings.HasPrefix(child, parent+"/") {
		return true
	}
	return false
}

func (c *Context) AlreadyDoneGit(pkg string) bool {
	for donePkg, _ := range c.doneGit {
		if pkgContains(donePkg, pkg) {
			return true
		}
	}
	return false
}

func (c *Context) AlreadyDoneGo(pkg string) bool {
	_, ok := c.doneGo[pkg]
	return ok
}

func stringInSlice(slice []string, s string) bool {
	for _, e := range slice {
		if s == e {
			return true
		}
	}
	return false
}

func (c *Context) goToMostRecentTag(pkg, goDir string) error {
	tags, err := c.gitCtx.Tags(goDir)
	if err != nil {
		return c.errorf("Failed to get git tags for package %s (%s): %s",
			pkg, goDir, err.Error())
	}

	mostRecentTag, err := c.gitCtx.MostRecentTag(goDir)
	if err != nil {
		return c.errorf("Failed to get most recent tag for package %s (%s): %s",
			pkg, goDir, err.Error())
	} else if mostRecentTag == "" {
		c.warnf("Package %s (%s) has no tags", pkg, goDir)
		return nil
	}

	if stringInSlice(tags, mostRecentTag) {
		//Already pointing to latest tag
		return nil
	}

	err = c.gitCtx.Checkout(goDir, mostRecentTag)
	if err != nil {
		return c.errorf("Failed to checkout tag %s for pacakge %s (%s): %s",
			mostRecentTag, pkg, goDir, err.Error())

	} else {
		return nil
	}
}

func (c *Context) clone(workingDir, pkg, goDir string, depsOnly, tests bool) (bool, error) {
	if c.AlreadyDoneGit(pkg) {
		return false, nil
	}

	rootPkg, gitUrl, err := c.ruleSet.GetUrl(pkg)

	if c.AlreadyDoneGit(rootPkg) {
		c.doneGit[pkg] = empty{}
		return false, nil
	}
	if err != nil {
		return true, c.errorf("%s", err.Error())
	}

	if rootPkg != pkg {
		if c.flags.Checked(RecurseTopLevel) {
			return false, c.Get(workingDir, rootPkg, depsOnly, tests)
		} else {
			//This is a bit of a hack
			//one other option would be to just recreate the path from GOPATH.
			goDirSlash := filepath.ToSlash(goDir)
			if !strings.HasSuffix(goDirSlash, pkg) {
				return false, c.errorf("couldn't map %s in %s", pkg, goDir)
			}
			goDirNoPkg := strings.TrimSuffix(goDirSlash, pkg)
			goDirRootPkg := goDirNoPkg + rootPkg
			goDir = filepath.FromSlash(goDirRootPkg)
		}
	}

	err = c.gitCtx.Clone(goDir, gitUrl)
	if err != nil {
		return true, c.errorf("Failed to clone %s:\n%s", gitUrl, err.Error())
	}

	if c.flags.Checked(TaggedOnly) {
		err := c.goToMostRecentTag(pkg, goDir)
		if err != nil {
			return true, err
		}
	}

	c.doneGit[pkg] = empty{}
	c.doneGit[rootPkg] = empty{}
	return true, nil
}

func (c *Context) gitTopLevel(goDir string) (string, error) {
	if cached, hasCache := c.gitTopCache[goDir]; hasCache {
		return cached, nil
	}
	v, err := c.gitCtx.TopLevel(goDir)
	if err != nil {
		c.gitTopCache[goDir] = v
	}
	return v, err
}

func (c *Context) inspect(workingDir, pkg, goDir string, depsOnly, tests bool) (bool, error) {
	//never inspect twice
	c.doneGo[pkg] = empty{}

	isGit := c.gitCtx.IsGit(goDir)
	if !isGit {
		return true, c.errorf("Package %s (%s) is not a git repository", pkg, goDir)
	}

	var rootPkg string
	if c.flags.Checked(RecurseTopLevel) {
		gitTopLevel, err := c.gitTopLevel(goDir)
		if err != nil {
			return true, c.errorf("%s", err.Error())
		}

		nativePkg := filepath.FromSlash(pkg)

		if !strings.HasSuffix(goDir, nativePkg) {
			return true, c.errorf("Package %s (%s) not part of path (%s)", pkg, nativePkg, goDir)
		}
		srcPath := strings.TrimSuffix(goDir, nativePkg)

		if !strings.HasPrefix(gitTopLevel, srcPath) {
			return true, c.errorf("Git top level (%s) of package %s, not below src (%s)",
				gitTopLevel, pkg, srcPath)
		}

		rootPkg = filepath.ToSlash(strings.TrimPrefix(gitTopLevel, srcPath))
	} else {
		rootPkg = pkg
	}

	if rootPkg != pkg {
		if c.AlreadyDoneGo(rootPkg) {
			return false, nil
		}
		c.doneGo[rootPkg] = empty{}

		//Costs an extra call out to git, but keeps the code way more managable
		err := c.Get(workingDir, rootPkg, depsOnly, tests)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	//Updates are only done if possible. Not an error to fail.
	if c.flags.Checked(Update) {
		err := c.runHook(pkg, goDir, "get-before-update.sh")
		if err != nil {
			return true, err
		} else if gitStatus, err := c.gitCtx.Status(goDir); err != nil {
			return true, c.errorf("Failed to get git status for package %s (%s): %s",
				pkg, goDir, err.Error())
		} else if gitStatus != git.Clean {
			c.warnf("Not updating package %s (%s), git status is %s",
				pkg, goDir, gitStatus.String())
		} else if err := c.gitCtx.Checkout(goDir, "master"); err != nil {
			c.warnf("Not updating package %s (%s), Couldn't checkout master: %s",
				pkg, goDir, err.Error())
		} else if err := c.gitCtx.Pull(goDir); err != nil {
			c.warnf("Not updating package %s (%s), Couldn't pull: %s",
				pkg, goDir, err.Error())
		} else if c.flags.Checked(TaggedOnly) {
			err := c.goToMostRecentTag(pkg, goDir)
			if err != nil {
				return true, err
			}
		}
	}

	return true, nil
}

func (c *Context) runHook(pkg, goDir, filename string) error {
	if !c.flags.Checked(ApplyHooks) {
		return nil
	}

	hookFile := filepath.Join(goDir, filename)

	if !dsutil.CheckPath(hookFile) {
		return nil
	}

	output, _, err := c.cmdCtx.Execf("cd %s; %s", pkg, dsutil.PosixPath(hookFile))
	if err != nil {
		return c.errorf("Failed to run hook script '%s' for package %s (%s): %s\n%s",
			filename, pkg, goDir, err.Error(), output)
	}

	return nil
}

func (c *Context) Get(workingDir, pkg string, depsOnly, tests bool) error {
	return c.getList(workingDir, pkg, pkg, depsOnly, tests)
}

func (c *Context) getList(workingDir, pkg, listPkg string, depsOnly, tests bool) error {
	goDir, alreadyExists := c.goCtx.Dir(workingDir, pkg)
	c.doneGo[pkg] = empty{}

	if depsOnly && !alreadyExists {
		return c.errorf("Can not get dependencies only for package %s, package does not exist", pkg)
	} else if depsOnly {
		//Can do nothing

	} else if !alreadyExists {
		shouldContinue, err := c.clone(workingDir, pkg, goDir, depsOnly, tests)
		if err != nil {
			return err
		} else if !shouldContinue {
			return nil
		}
	} else if !c.flags.Checked(Update) && !c.flags.Checked(DeepScan) {
		return nil
	} else {
		shouldContinue, err := c.inspect(workingDir, pkg, goDir, depsOnly, tests)
		if err != nil {
			return err
		} else if !shouldContinue {
			return nil
		}
	}

	var listPkgStr = listPkg
	if c.flags.Checked(RecurseTopLevel) {
		listPkgStr = listPkg + "/..."
	}
	list, err := c.goCtx.List(workingDir, listPkgStr)
	if err != nil {
		return err
	}

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

			if !c.goCtx.IsStdLib(imp) && !c.AlreadyDoneGo(imp) {
				err := c.Get(workingDir, imp, false, false)
				if err != nil {
					return err
				}
			}
		}
		if tests {
			for _, impInt := range testImports {
				imp := impInt.(string)
				if !c.goCtx.IsStdLib(imp) && !c.AlreadyDoneGo(imp) {
					err := c.Get(workingDir, imp, false, false)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	//Before install hook runs even if the install flag hasn't been set
	err = c.runHook(pkg, goDir, "get-before-install.sh")
	if err != nil {
		return err
	}

	failed := []string{}
	if c.flags.Checked(Install) {
		if !c.flags.Checked(RecurseTopLevel) {
			err := c.goCtx.Install(workingDir, pkg)
			if err != nil {
				c.warnf("%s Failed", pkg)
			}
		} else {
			//attempt to install everything; takes advantage of multiple cores
			//but will bomb out if some of the sub pkgs are particularly broken
			//if that happens, attempt installing one by one instead
			err := c.goCtx.Install(workingDir, pkg+"/...")
			if err != nil {
				for importPath, _ := range list {
					err := c.goCtx.Install(workingDir, importPath)
					if err != nil {
						//TODO check this part works:
						if strings.HasPrefix(importPath, pkg+"/") {
							failed = append(failed, ".../"+strings.TrimPrefix(importPath, pkg+"/"))
						} else {
							failed = append(failed, importPath)
						}
					}
				}
				c.warnf("%s/... [Failed: %s]", pkg, strings.Join(failed, ", "))
			}
		}
	}

	//Post install always runs, even if errors (for the case where a subpkg fails)
	err = c.runHook(pkg, goDir, "get-after-install.sh")
	if err != nil {
		return err
	}

	if len(failed) == 0 {
		c.verbosef("%s", pkg)
	}

	return nil
}
