package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var goPath []string
var stdLibs map[string]struct{}

func getStdLibs() map[string]struct{} {
	stdListOutput := MustRunCmd(".", "go", "list", "std")
	stdLibs := SplitLines(stdListOutput)
	result := map[string]struct{}{}
	for _, lib := range stdLibs {
		if lib != "" {
			result[lib] = struct{}{}
		}
	}
	return result
}

func GoIsStdLib(lib string) bool {
	_, result := stdLibs[lib]
	return result
}

func init() {
	envPath := os.Getenv("GOPATH")
	if len(envPath) == 0 {
		fmt.Println("ERROR GOPATH not set")
		os.Exit(1)
	}

	if runtime.GOOS == "windows" {
		goPath = strings.Split(envPath, ";")
	} else {
		goPath = strings.Split(envPath, ":")
	}

	if !CheckPath(goPath[0]) {
		fmt.Printf("ERROR First GOPATH element (%s) not found\n", goPath[0])
		os.Exit(1)
	}

	stdLibs = getStdLibs()
}

func GoList(workingDir, cmd, pkg string, mustRun bool) ([]string, error) {
	args := []string{"list", "-f", fmt.Sprintf(`{{join .%s "\n"}}`, cmd)}
	if pkg != "" {
		args = append(args, pkg)
	}

	var goListOutput string
	var err error
	if mustRun {
		goListOutput = MustRunCmd(workingDir, "go", args...)
	} else {
		goListOutput, err = RunCmd(workingDir, "go", args...)
	}

	var result []string
	if err == nil {
		result = SplitLines(goListOutput)
		result = result[:len(result)-1]
	}
	return result, err
}

func GoDeps(pkg string, tests bool) map[string]struct{} {
	//This isn't vendor aware, packages already covered by a vendor include will be still returned
	//To workaround this I'd have to do the recursion myself, and ignore anything that's present in the vendor.

	depsRaw, _ := GoList(".", "Deps", pkg+"/...", true)
	depSet := map[string]struct{}{}
	for _, dep := range depsRaw {
		depSet[dep] = struct{}{}
	}

	if !tests {
		return depSet
	}

	testImports, _ := GoList(".", "TestImports", pkg+"/...", true)
	for _, testImport := range testImports {
		if _, covered := depSet[testImport]; covered {
			continue
		}
		depSet[testImport] = struct{}{}

		testImportDeps, _ := GoList(".", "Deps", testImport, true)
		for _, testImportDep := range testImportDeps {
			depSet[testImportDep] = struct{}{}
		}
	}

	return depSet
}

func GoPathPkg(pkg string) string {
	return filepath.Join(goPath[0], "src", pkg)
}

func GoDir(workingDir, pkg string) (string, bool) {
	goListOutput, err := RunCmd(workingDir, "go", "list", "-f", "{{.Dir}}", pkg)
	if err == nil {
		return FirstLine(goListOutput), true
	}
	return GoPathPkg(pkg), false
}

func GoName(path string) (string, error) {
	for _, gopath := range goPath {
		relPath, err := filepath.Rel(filepath.Join(gopath, "src"), path)
		if err == nil && !strings.Contains(relPath, "..") {
			return filepath.ToSlash(relPath), nil
		}
	}
	return "", fmt.Errorf("Could not match %s to any element of gopath (%v)", path, goPath)
}
