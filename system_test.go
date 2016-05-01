package main

import (
	"testing"

	"github.com/desal/cmd"
	"github.com/desal/richtext"
)

func TestScenario(t *testing.T) {
	output := cmd.NewStdOutput(true, richtext.Ansi())

	repos := NewRepos(output)

	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u1/p1/s2"))

	repos.Test(func(goPath []string) {
		ctx := NewContext(true, ScanMode_Skip, output, goPath)
		ctx.Get(".", "gh/u1/p1/s2", false, false)
	})

	repos.Test(func(goPath []string) {
		ctx := NewContext(true, ScanMode_Skip, output, goPath)
		ctx.Get(".", "gh/u2/p2", false, false)
	})
}
