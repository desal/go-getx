package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type rule struct {
	re      *regexp.Regexp
	replace string
}

var rules []rule

func NewRule(match, replace string) rule {
	r := rule{}
	match = "^" + strings.TrimSpace(match)

	var err error
	r.re, err = regexp.Compile(match)

	if err != nil {
		fmt.Println("Regexp Error:", err.Error())
		os.Exit(1)
	}

	r.replace = strings.TrimSpace(replace)
	return r
}

func (r *rule) tryRegex(s string) (goImport, gitUrl string, success bool) {
	subStr := r.re.FindString(s)
	if subStr == "" {
		return "", "", false
	}
	return subStr, r.re.ReplaceAllString(subStr, r.replace), true
}

func GetUrl(pkg string) (goImport, gitUrl string, err error) {
	for _, rule := range rules {
		goImport, gitUrl, ok := rule.tryRegex(pkg)
		if ok {
			return goImport, gitUrl, nil
		}
	}
	return "", "", fmt.Errorf("Could not find a rule matching %s", pkg)
}

func LoadRules() error {
	filename := filepath.Join(UserHomeDir(), ".go-getx-map")
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if string(text[0]) == "#" {
			continue
		}

		ruleDef := strings.SplitN(text, "=", 2)
		if len(ruleDef) < 2 {
			continue
		}
		rules = append(rules, NewRule(ruleDef[0], ruleDef[1]))
	}
	return nil
}
