// Command antitests counts anti-tests, package by package, so the rule
// "every positive test is answered by at least one anti-test" can be held
// as a number.
//
// usage, from slayer's root:
//
//	antitests [-v] [-min RATIO] PKG...
//
// PKG is a package directory; dir/... is every package below dir. An
// anti-test is a test function with a line `// anti-test of TestX`
// directly above it; the line must match exactly, alone on its line. A
// positive is a test function with no such line above it. The ratio is the
// positives named by at least one marker (above a function or inside one)
// over the positives.
//
// A marker naming a test that is not in the package is
// an error. Old markers (`// anti-test:`, `// anti:`) name no test; they
// are counted and listed.
//
// exit 0, or 1 if a marker names a missing test or a package is below
// -min, or 2 if the packages cannot be read.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	exitBelow = 1 // defined here: a missing name or a ratio under -min
	exitUsage = 2 // defined here: arguments or files that cannot be read
)

// main runs the audit on the command line arguments and exits with its code.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run audits the packages named in args and returns the exit code
func run(args []string, out, errOut io.Writer) int {
	flags := flag.NewFlagSet("antitests", flag.ContinueOnError)
	flags.SetOutput(errOut)
	min := flags.Float64("min", -1, "least ratio a package may have")
	verbose := flags.Bool("v", false, "list unanswered positives")
	if flags.Parse(args) != nil {
		return exitUsage
	}
	if flags.NArg() == 0 {
		fmt.Fprintln(errOut, "antitests: no packages")
		return exitUsage
	}
	dirs, err := expand(flags.Args())
	if err != nil {
		fmt.Fprintln(errOut, "antitests:", err)
		return exitUsage
	}
	total := report{dir: "total"}
	code := 0
	for _, dir := range dirs {
		rep, err := auditDir(dir)
		if err != nil {
			fmt.Fprintln(errOut, "antitests:", err)
			return exitUsage
		}
		show(out, rep, *verbose)
		if len(rep.missing) > 0 || (*min >= 0 && rep.ratio() < *min) {
			code = exitBelow
		}
		total.add(rep)
	}
	show(out, total, false)
	return code
}

// add folds the counts, missing markers, and old markers of o into r. It
// does not carry over o's unanswered places.
func (r *report) add(o report) {
	r.tests += o.tests
	r.antis += o.antis
	r.positives += o.positives
	r.named += o.named
	r.missing = append(r.missing, o.missing...)
	r.old = append(r.old, o.old...)
}

// show prints the package's line, then the markers that name no test and
// the old markers, then with verbose the unanswered positives
func show(out io.Writer, r report, verbose bool) {
	fmt.Fprintf(out, "%s: tests %d, anti-tests %d, positives %d, "+
		"named %d, ratio %.2f, missing %d, old %d\n", r.dir, r.tests,
		r.antis, r.positives, r.named, r.ratio(), len(r.missing),
		len(r.old))
	for _, m := range r.missing {
		fmt.Fprintf(out, "  missing: %s names %s, which does not exist\n",
			m.at, m.name)
	}
	for _, at := range r.old {
		fmt.Fprintf(out, "  old marker: %s\n", at)
	}
	if verbose {
		for _, at := range r.unanswered {
			fmt.Fprintf(out, "  unanswered: %s\n", at)
		}
	}
}
