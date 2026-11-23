package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// names maps a package's import path to its test names
type names map[string][]string

// count is the number of names in all packages
func count(n names) int {
	total := 0
	for _, list := range n {
		total += len(list)
	}
	return total
}

// sortedPackages is the package paths of n, in order
func sortedPackages(n names) []string {
	var pkgs []string
	for pkg := range n {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)
	return pkgs
}

// format is n as `package<TAB>name` lines, in order
func format(n names) string {
	var b strings.Builder
	for _, pkg := range sortedPackages(n) {
		list := append([]string(nil), n[pkg]...)
		sort.Strings(list)
		for _, name := range list {
			fmt.Fprintf(&b, "%s\t%s\n", pkg, name)
		}
	}
	return b.String()
}

// readSaved reads the lines format wrote
func readSaved(file string) (names, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	n := names{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		pkg, name, ok := strings.Cut(sc.Text(), "\t")
		if !ok || pkg == "" || name == "" {
			return nil, fmt.Errorf("%s: not `package<TAB>name`: %q", file, sc.Text())
		}
		n[pkg] = append(n[pkg], name)
	}
	return n, sc.Err()
}

// parseList reads the output of `go test -list . ./...`: each package's
// names come before its `ok` line. A package with no tests, or with
// none under this tag, has no names.
func parseList(output []byte) (names, error) {
	n := names{}
	var pending []string
	sc := bufio.NewScanner(bytes.NewReader(output))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		switch {
		case len(fields) >= 2 && fields[0] == "ok":
			if len(pending) > 0 {
				n[fields[1]] = append(n[fields[1]], pending...)
			}
			pending = nil
		case len(fields) >= 2 && (fields[0] == "?" || fields[0] == "FAIL"):
			if fields[0] == "FAIL" {
				return nil, fmt.Errorf("a package failed: %s", line)
			}
			pending = nil
		case len(fields) == 1:
			pending = append(pending, fields[0])
		default:
			return nil, fmt.Errorf("cannot read %q", line)
		}
	}
	return n, sc.Err()
}

// listTree lists every test of the packages matching patterns, built
// with the tag, or with none
func listTree(tag string, patterns []string) (names, error) {
	args := []string{"test", "-vet=off", "-list", ".", "-count=1"}
	if tag != "" {
		args = append(args, "-tags", tag)
	}
	args = append(args, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(stderr.String(), "build constraints exclude all Go files") {
			return names{}, nil // every file of the package is of the other tier
		}
		return nil, fmt.Errorf("go %s: %v\n%s%s", strings.Join(args, " "), err, output, stderr.String())
	}
	return parseList(output)
}

// drawOnly is the names in all that plain lacks
func drawOnly(all, plain []string) []string {
	have := map[string]bool{}
	for _, name := range plain {
		have[name] = true
	}
	var rest []string
	for _, name := range all {
		if !have[name] {
			rest = append(rest, name)
		}
	}
	return rest
}

// compare is every way the tiers differ from the saved names: a saved
// name the draw build lacks (lost), a draw name not saved (added), a
// plain name the draw build lacks (misplaced), and a name twice in one
// list (duplicated)
func compare(saved, all, plain names) []string {
	var problems []string
	pkgs := map[string]bool{}
	for _, n := range []names{saved, all, plain} {
		for pkg := range n {
			pkgs[pkg] = true
		}
	}
	var order []string
	for pkg := range pkgs {
		order = append(order, pkg)
	}
	sort.Strings(order)
	for _, pkg := range order {
		problems = append(problems, setDiff(pkg, "lost", saved[pkg], all[pkg])...)
		problems = append(problems, setDiff(pkg, "added", all[pkg], saved[pkg])...)
		problems = append(problems, setDiff(pkg, "in neither tier but plain", plain[pkg], all[pkg])...)
		problems = append(problems, repeats(pkg, "draw build", all[pkg])...)
		problems = append(problems, repeats(pkg, "plain build", plain[pkg])...)
	}
	return problems
}

// setDiff is a problem line for each name in from that is not in other
func setDiff(pkg, what string, from, other []string) []string {
	have := map[string]bool{}
	for _, name := range other {
		have[name] = true
	}
	var out []string
	for _, name := range from {
		if !have[name] {
			out = append(out, fmt.Sprintf("%s: %s: %s", pkg, what, name))
		}
	}
	return out
}

// repeats is a problem line for each name that list holds more than once
func repeats(pkg, where string, list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range list {
		if seen[name] {
			out = append(out, fmt.Sprintf("%s: duplicated in the %s: %s", pkg, where, name))
		}
		seen[name] = true
	}
	return out
}

// only is the saved names of the packages the patterns match
func only(saved names, patterns []string) (names, error) {
	args := append([]string{"list", "-e"}, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go %s: %v", strings.Join(args, " "), err)
	}
	kept := names{}
	for _, pkg := range strings.Fields(string(output)) {
		if list, ok := saved[pkg]; ok {
			kept[pkg] = list
		}
	}
	return kept, nil
}

// readHot is the import paths in a hot list: one a line, a comment after
// a hash, blank lines and comment lines left out
func readHot(file string) ([]string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var pkgs []string
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		switch len(fields) {
		case 0:
		case 1:
			pkgs = append(pkgs, fields[0])
		default:
			return nil, fmt.Errorf("%s: more than one word before the comment: %q", file, line)
		}
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("%s: no packages", file)
	}
	return pkgs, nil
}

// hot runs the plain tests of the packages in the hot list, and returns
// go test's exit code
func hot(file string, out, errOut io.Writer) int {
	pkgs, err := readHot(file)
	if err != nil {
		fmt.Fprintln(errOut, "tiers:", err)
		return exitUsage
	}
	cmd := exec.Command("go", append([]string{"test", "-count=1"}, pkgs...)...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout, cmd.Stderr = out, errOut
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintln(errOut, "tiers:", err)
		return exitUsage
	}
	return 0
}

// drawTimeout is how long one package's draw tier may run: the session's
// drawing tests alone take 7 to 25 minutes on this laptop
const drawTimeout = "60m"

// exitFailed is draw's exit when a package's draw tests failed
const exitFailed = 1 // defined here, as go test's own failure

// drawArgs is the go test arguments that run only the named tests of one
// package with the draw tag, or nil when there are none to run
func drawArgs(pkg string, tests []string) []string {
	if len(tests) == 0 {
		return nil
	}
	quoted := make([]string, len(tests))
	for i, name := range tests {
		quoted[i] = regexp.QuoteMeta(name)
	}
	return []string{"test", "-count=1", "-tags", "draw", "-p", "1",
		"-timeout", drawTimeout, "-run", "^(" + strings.Join(quoted, "|") + ")$", pkg}
}

// draw runs the draw tier alone: per package, the tests the draw build
// has and the plain build lacks, one package after another. It returns 1
// if any package failed, and names each one that did
func draw(patterns []string, out, errOut io.Writer) int {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	all, err := listTree("draw", patterns)
	if err != nil {
		fmt.Fprintln(errOut, "tiers:", err)
		return exitUsage
	}
	plain, err := listTree("", patterns)
	if err != nil {
		fmt.Fprintln(errOut, "tiers:", err)
		return exitUsage
	}
	var failed []string
	for _, pkg := range sortedPackages(all) {
		args := drawArgs(pkg, drawOnly(all[pkg], plain[pkg]))
		if args == nil {
			continue
		}
		cmd := exec.Command("go", args...)
		cmd.Env = append(os.Environ(), "GOWORK=off")
		cmd.Stdout, cmd.Stderr = out, errOut
		if err := cmd.Run(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				fmt.Fprintln(errOut, "tiers:", err)
				return exitUsage
			}
			failed = append(failed, pkg)
		}
	}
	for _, pkg := range failed {
		fmt.Fprintln(out, "tiers: draw failed in", pkg)
	}
	if len(failed) > 0 {
		return exitFailed
	}
	return 0
}
