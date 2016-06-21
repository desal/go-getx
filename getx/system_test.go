package getx

import (
	"testing"

	"github.com/desal/cmd"
	"github.com/stretchr/testify/assert"
)

func TestSingleRepo(t *testing.T) {
	output := cmd.NewTestOutput(t)

	repos := NewRepos(output)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u1/p1/s2"))

	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		ctx := NewContext(true, ScanMode_Skip, output, goPath, ruleSet)
		ctx.Get(".", "gh/u1/p1/s2", false, false, false)
	})

	expected := set{
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
	output := cmd.NewTestOutput(t)

	repos := NewRepos(output)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u1/p1/s2"))

	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		ctx := NewContext(true, ScanMode_Skip, output, goPath, ruleSet)
		ctx.Get(".", "gh/u2/p2", false, false, false)
	})

	expected := set{
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
	output := cmd.NewTestOutput(t)

	repos := NewRepos(output)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u2/p1"))

	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		ctx := NewContext(true, ScanMode_Skip, output, goPath, ruleSet)
		ctx.Get(".", "gh/u2/p2", false, false, false)
	})

	expected := set{
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
	output := cmd.NewTestOutput(t)

	repos := NewRepos(output)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1", "gh/u1/p1/missing1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u2/p1"))

	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		ctx := NewContext(true, ScanMode_Skip, output, goPath, ruleSet)
		ctx.Get(".", "gh/u2/p2", false, false, false)
	})

	expected := set{
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
	output := cmd.NewTestOutput(t)

	repos := NewRepos(output)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p2/s1"))
	repos.AddRepo("gh/u1/p2",
		Pkg("gh/u1/p2/s1"))
	//Test Get
	fileList := repos.Test(func(goPath []string, ruleSet RuleSet) {
		{
			ctx := NewContext(true, ScanMode_Skip, output, goPath, ruleSet)
			ctx.Get(".", "gh/u1/p1", false, false, false)
		}
		{
			ctx := NewContext(true, ScanMode_Update, output, goPath, ruleSet)
			ctx.Get(".", "gh/u1/p1", false, false, false)
		}
	})

	expected := set{
		"./pkg/gh/u1/p1.a":         empty{},
		"./pkg/gh/u1/p2/s1.a":      empty{},
		"./src/gh/u1/p1/gen.go":    empty{},
		"./src/gh/u1/p2/s1/gen.go": empty{},
	}
	assert.Equal(t, expected, fileList)

}
