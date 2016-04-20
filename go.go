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
	stdLibs := strings.Split(string(stdListOutput), "\n")
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

	var goListOutput []byte
	var err error
	if mustRun {
		goListOutput = MustRunCmd(workingDir, "go", args...)
	} else {
		_, goListOutput, err = RunCmd(workingDir, "go", args...)
	}

	var result []string
	if err == nil {
		result = strings.Split(string(goListOutput), "\n")
		result = result[:len(result)-1]
	}
	return result, err
}

func GoDeps(pkg string, tests bool) map[string]struct{} {
	depsRaw, _ := GoList(".", "Deps", pkg+"/...", true)
	depSet := map[string]struct{}{}
	for _, dep := range depsRaw {
		depSet[dep] = struct{}{}
	}

	if !tests {
		return depSet
	}

	testImports, _ := GoList(".", "TestImports", "./...", true)
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
	//TODO surprisingly, go list here doesn't give a vendored folder if it exists
	//work around (will need to search the vendor folder for the pkg first)
	_, goListOutput, err := RunCmd(workingDir, "go", "list", "-f", "{{.Dir}}", pkg)
	if err == nil {
		return strings.Split(string(goListOutput), "\n")[0], true
	}
	return GoPathPkg(pkg), false
}

func GoName(path string) string {
	return string(MustRunCmd(path, "go", "list", "-f", "{{.Name}}"))
}
