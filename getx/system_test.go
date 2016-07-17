package getx

import (
	"testing"

	"github.com/desal/richtext"
	"github.com/stretchr/testify/assert"
)

func TestSingleRepo(t *testing.T) {
	format := richtext.Test(t)

	repos := NewRepos(format)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u1/p1/s2"))

	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		ctx := New(format, goPath, ruleSet, MustPanic, Install)
		ctx.Get(".", "gh/u1/p1/s2", false, false)
	})

	expected := stringSet{
		"./pkg/gh/u1/p1.a":         empty{},
		"./pkg/gh/u1/p1/s1.a":      empty{},
		"./pkg/gh/u1/p1/s2.a":      empty{},
		"./src/gh/u1/p1/gen.go":    empty{},
		"./src/gh/u1/p1/s1/gen.go": empty{},
		"./src/gh/u1/p1/s2/gen.go": empty{},
	}
	assert.Equal(t, expected, fileList)

}

func TestDependentRepo(t *testing.T) {
	format := richtext.Test(t)

	repos := NewRepos(format)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u1/p1/s2"))

	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		ctx := New(format, goPath, ruleSet, MustPanic, Install)
		ctx.Get(".", "gh/u2/p2", false, false)
	})

	expected := stringSet{
		"./pkg/gh/u1/p1.a":         empty{},
		"./pkg/gh/u1/p1/s1.a":      empty{},
		"./pkg/gh/u1/p1/s2.a":      empty{},
		"./pkg/gh/u2/p2.a":         empty{},
		"./src/gh/u1/p1/gen.go":    empty{},
		"./src/gh/u1/p1/s1/gen.go": empty{},
		"./src/gh/u1/p1/s2/gen.go": empty{},
		"./src/gh/u2/p2/gen.go":    empty{},
	}
	assert.Equal(t, expected, fileList)
}

func TestMultiDepOk(t *testing.T) {
	format := richtext.Test(t)

	repos := NewRepos(format)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u2/p1"))

	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		ctx := New(format, goPath, ruleSet, MustPanic, Install)
		ctx.Get(".", "gh/u2/p2", false, false)
	})

	expected := stringSet{
		"./pkg/gh/u1/p1.a":         empty{},
		"./pkg/gh/u1/p1/s1.a":      empty{},
		"./pkg/gh/u1/p1/s2.a":      empty{},
		"./pkg/gh/u2/p1.a":         empty{},
		"./pkg/gh/u2/p2.a":         empty{},
		"./src/gh/u1/p1/gen.go":    empty{},
		"./src/gh/u1/p1/s1/gen.go": empty{},
		"./src/gh/u1/p1/s2/gen.go": empty{},
		"./src/gh/u2/p1/gen.go":    empty{},
		"./src/gh/u2/p2/gen.go":    empty{},
	}
	assert.Equal(t, expected, fileList)
}

func TestMultiDepPartialFail(t *testing.T) {
	format := richtext.Test(t)

	repos := NewRepos(format)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1", "gh/u1/p1/missing1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u2/p1"))

	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		ctx := New(format, goPath, ruleSet, MustPanic, Install)
		ctx.Get(".", "gh/u2/p2", false, false)
	})

	expected := stringSet{
		//"./pkg/gh/u1/p1/s2.a":      empty{},
		"./pkg/gh/u1/p1.a":         empty{},
		"./pkg/gh/u1/p1/s1.a":      empty{},
		"./pkg/gh/u2/p1.a":         empty{},
		"./pkg/gh/u2/p2.a":         empty{},
		"./src/gh/u1/p1/gen.go":    empty{},
		"./src/gh/u1/p1/s1/gen.go": empty{},
		"./src/gh/u1/p1/s2/gen.go": empty{},
		"./src/gh/u2/p1/gen.go":    empty{},
		"./src/gh/u2/p2/gen.go":    empty{},
	}
	assert.Equal(t, expected, fileList)
}

func TestDepNoRootUpgrade(t *testing.T) {
	format := richtext.Test(t)

	repos := NewRepos(format)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p2/s1"))
	repos.AddRepo("gh/u1/p2",
		Pkg("gh/u1/p2/s1"))
	//Test Get
	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		{
			ctx := New(format, goPath, ruleSet)
			ctx.Get(".", "gh/u1/p1", false, false)
		}
		{
			ctx := New(format, goPath, ruleSet, Update, MustPanic, Install)
			ctx.Get(".", "gh/u1/p1", false, false)
		}
	})

	expected := stringSet{
		"./pkg/gh/u1/p1.a":         empty{},
		"./pkg/gh/u1/p2/s1.a":      empty{},
		"./src/gh/u1/p1/gen.go":    empty{},
		"./src/gh/u1/p2/s1/gen.go": empty{},
	}
	assert.Equal(t, expected, fileList)

}

func TestHooks(t *testing.T) {
	format := richtext.Test(t)

	repos := NewRepos(format)

	pkg := Pkg("hookrepo")
	repo := repos.AddRepo("hookrepo", pkg)

	repo.hookBeforeUpdate = `#!/usr/bin/env sh
if [ -e ".hook1" ]; then
	touch .hookfail
fi
touch .hook1
`
	repo.hookBeforeInstall = `#!/usr/bin/env sh
if [ -e ".hook2a" ]; then
	touch .hook2b
fi
touch .hook2a`

	repo.hookAfterInstall = `#!/usr/bin/env sh
if [ -e ".hook3a" ]; then
	touch .hook3b
fi
touch .hook3a`

	repo.gitIgnore = `
.hook*
`

	//Install
	fileList1 := repos.Test(func(goPath []string, ruleSet RuleSet) {
		ctx := New(format, goPath, ruleSet, Verbose, MustPanic, Install, ApplyHooks)
		ctx.Get(".", "hookrepo", false, false)
	})

	expected1 := stringSet{
		"./pkg/hookrepo.a":                     empty{},
		"./src/hookrepo/.gitignore":            empty{},
		"./src/hookrepo/.hook2a":               empty{},
		"./src/hookrepo/.hook3a":               empty{},
		"./src/hookrepo/get-before-update.sh":  empty{},
		"./src/hookrepo/get-before-install.sh": empty{},
		"./src/hookrepo/get-after-install.sh":  empty{},
		"./src/hookrepo/gen.go":                empty{},
	}
	assert.Equal(t, expected1, fileList1)

	//Install AND Upgrade
	fileList2 := repos.Test(func(goPath []string, ruleSet RuleSet) {
		{
			ctx := New(format, goPath, ruleSet, Verbose, MustPanic, Install, ApplyHooks)
			ctx.Get(".", "hookrepo", false, false)
		}

		{
			ctx := New(format, goPath, ruleSet, Verbose, MustPanic, Install, ApplyHooks, Update)
			ctx.Get(".", "hookrepo", false, false)
		}
	})

	expected2 := stringSet{
		"./pkg/hookrepo.a":                     empty{},
		"./src/hookrepo/.gitignore":            empty{},
		"./src/hookrepo/.hook1":                empty{}, //MISSING
		"./src/hookrepo/.hook2a":               empty{},
		"./src/hookrepo/.hook2b":               empty{}, //MISSING
		"./src/hookrepo/.hook3a":               empty{},
		"./src/hookrepo/.hook3b":               empty{}, //MISSING
		"./src/hookrepo/get-before-update.sh":  empty{},
		"./src/hookrepo/get-before-install.sh": empty{},
		"./src/hookrepo/get-after-install.sh":  empty{},
		"./src/hookrepo/gen.go":                empty{},
	}
	assert.Equal(t, expected2, fileList2)
}

//Poor test coverage:
// non-git repos
// tags
