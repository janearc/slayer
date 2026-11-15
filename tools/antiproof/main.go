// Command antiproof proves an anti-test can fail: it breaks one place in
// one file, runs the named tests, and puts the file back.
//
// usage, from slayer's root:
//
//	antiproof -file F -old OLD -new NEW -run REGEX [-count N] [-C DIR] PKG
//
// OLD must occur in F exactly once; it is replaced by NEW, then
// `go test -count=N -run REGEX PKG` runs. The file is put back byte for
// byte on every path: tests that fail or pass, a build that fails, a
// panic, an interrupt (the tests are killed first). If the file cannot be
// put back, it says so and exits 3.
//
// exit 0 if the tests failed under the break (the proof holds), 1 if they
// passed (they would not notice the fault), 2 if the tool could not do
// its work: no single match, a build that did not compile, no tests run.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const (
	exitFailed   = 0 // defined here: the tests failed under the break
	exitPassed   = 1 // defined here: the tests passed under the break
	exitToolFail = 2 // defined here: nothing was proven either way
	exitStuck    = 3 // defined here: the file could not be put back
)

// main reads the flags, runs the proof with interrupt, terminate, and
// hangup signals passed to it, and exits with the code report gives. A flag
// error exits with exitToolFail.
func main() {
	opts, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "antiproof:", err)
		os.Exit(exitToolFail)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	out, text, err := prove(opts, goTest, signals)
	os.Exit(report(out, text, err, os.Stdout, os.Stderr))
}
