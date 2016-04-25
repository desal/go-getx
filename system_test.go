package main

import (
	"testing"
)

func TestScenario(t *testing.T) {
	verbose = true
	repos := Repos{}
	repos.AddRepo("gh/u1/p1",
		Pkg("gh/u1/p1", "gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s1"),
		Pkg("gh/u1/p1/s2", "gh/u1/p1"))
	repos.AddRepo("gh/u2/p1",
		Pkg("gh/u2/p1", "gh/u1/p1/s1"))
	repos.AddRepo("gh/u2/p2",
		Pkg("gh/u2/p2", "gh/u1/p1/s1", "gh/u1/p1", "gh/u1/p1/s2"))

	repos.Test(func() {
		g := map[string]struct{}{}
		Get(".", "gh/u1/p1/s2", false, true, true, true, g)
	})

	repos.Test(func() {
		g := map[string]struct{}{}
		Get(".", "gh/u2/p2", false, true, true, true, g)
	})
}
