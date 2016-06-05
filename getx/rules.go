package getx

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type Rule struct {
	Re      *regexp.Regexp
	Replace string
}

type RuleSet struct {
	Rules []Rule
}

func NewRule(match, replace string) Rule {
	r := Rule{}
	match = "^" + strings.TrimSpace(match)

	var err error
	r.Re, err = regexp.Compile(match)

	if err != nil {
		fmt.Println("Regexp Error:", err.Error())
		os.Exit(1)
	}

	r.Replace = strings.TrimSpace(replace)
	return r
}

func (r *Rule) tryRegex(s string) (goImport, gitUrl string, success bool) {
	subStr := r.Re.FindString(s)
	if subStr == "" {
		return "", "", false
	}
	return subStr, r.Re.ReplaceAllString(subStr, r.Replace), true
}

func (r *RuleSet) GetUrl(pkg string) (goImport, gitUrl string, err error) {
	for _, rule := range r.Rules {
		goImport, gitUrl, ok := rule.tryRegex(pkg)
		if ok {
			return goImport, gitUrl, nil
		}
	}
	return "", "", fmt.Errorf("Could not find a rule matching %s", pkg)
}

func LoadRulesFromFile(filename string) (RuleSet, error) {
	file, err := os.Open(filename)
	if err != nil {
		return RuleSet{}, err
	}
	defer file.Close()
	return LoadRules(file)
}

func LoadRules(r io.Reader) (RuleSet, error) {
	rules := []Rule{}
	scanner := bufio.NewScanner(r)
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
	return RuleSet{rules}, nil
}
