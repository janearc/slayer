// Command tiers proves that sorting Slayer's tests into tiers lost,
// duplicated or added none.
//
// usage, from slayer's root:
//
//	tiers snapshot [-o FILE]   write every test name, built with -tags draw
//	tiers hot [-list FILE]     run the hot path tier: the plain tests of the
//	                           packages in tools/tiers/hot.txt
//	tiers draw [PATTERN...]    run the draw tier alone: in each package, only
//	                           the tests the plain build lacks, one package
//	                           at a time, -timeout 60m; every package is run
//	                           even after one fails
//	tiers check [-before FILE] [-v] [PATTERN...]
//	                           list the tree both ways and compare; with
//	                           patterns (./session), only those packages
//
// The default FILE is tools/tiers/before.txt, one `package<TAB>name` per
// line. check lists the tree with `go test -list` twice, once plain (the
// non draw tier) and once with -tags draw (every test), and exits 1 if
// the draw list is not the saved list name for name, or if the plain
// list holds a name the draw list does not. The names the plain list
// lacks are the draw tier, and are printed per package with -v.
//
// exit 0, or 1 on a lost, added or misplaced name, or 2 if the tree
// cannot be listed.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	exitDiffer = 1 // defined here: a name lost, added or misplaced
	exitUsage  = 2 // defined here: arguments, files or the build failed
)

const (
	defaultFile = "tools/tiers/before.txt"
	defaultHot  = "tools/tiers/hot.txt"
)

// main runs the command in the arguments
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run does the command named in args and returns the exit code
func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "tiers: want snapshot, hot, draw or check")
		return exitUsage
	}
	flags := flag.NewFlagSet("tiers "+args[0], flag.ContinueOnError)
	flags.SetOutput(errOut)
	file := flags.String("o", defaultFile, "the file of saved names")
	flags.StringVar(file, "before", defaultFile, "the file of saved names")
	list := flags.String("list", defaultHot, "the file of hot path packages")
	verbose := flags.Bool("v", false, "list the draw tier per package")
	if flags.Parse(args[1:]) != nil {
		return exitUsage
	}
	switch args[0] {
	case "snapshot":
		return snapshot(*file, out, errOut)
	case "hot":
		return hot(*list, out, errOut)
	case "draw":
		return draw(flags.Args(), out, errOut)
	case "check":
		return check(*file, *verbose, flags.Args(), out, errOut)
	}
	fmt.Fprintf(errOut, "tiers: unknown command %q\n", args[0])
	return exitUsage
}

// snapshot writes the names of every test in the tree, drawing ones too
func snapshot(file string, out, errOut io.Writer) int {
	all, err := listTree("draw", []string{"./..."})
	if err != nil {
		fmt.Fprintln(errOut, "tiers:", err)
		return exitUsage
	}
	formatted := format(all)
	if err := os.WriteFile(file, []byte(formatted), 0o644); err != nil {
		fmt.Fprintln(errOut, "tiers:", err)
		return exitUsage
	}
	fmt.Fprintf(out, "tiers: %d names in %s\n", count(all), file)
	return 0
}

// check compares the tree, listed both ways, to the saved names
func check(file string, verbose bool, patterns []string, out, errOut io.Writer) int {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	saved, err := readSaved(file)
	if err != nil {
		fmt.Fprintln(errOut, "tiers:", err)
		return exitUsage
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
	if len(patterns) != 1 || patterns[0] != "./..." {
		saved, err = only(saved, patterns)
		if err != nil {
			fmt.Fprintln(errOut, "tiers:", err)
			return exitUsage
		}
	}
	problems := compare(saved, all, plain)
	for _, p := range problems {
		fmt.Fprintln(out, p)
	}
	fmt.Fprintf(out, "tiers: %d saved, %d with the draw tag, %d without, %d draw only, %d problems\n",
		count(saved), count(all), count(plain), count(all)-count(plain), len(problems))
	if verbose {
		for _, pkg := range sortedPackages(all) {
			if d := drawOnly(all[pkg], plain[pkg]); len(d) > 0 {
				fmt.Fprintf(out, "%s\t%d draw\t%d plain\n", pkg, len(d), len(plain[pkg]))
			}
		}
	}
	if len(problems) > 0 {
		return exitDiffer
	}
	return 0
}
