package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// options are what one proof is asked to do
type options struct {
	file, old, new, run, pkg, dir string
	count                         int
}

// outcome is what the tests did under the break
type outcome int

const (
	testsFailed outcome = iota
	testsPassed
)

// stuckError is a file that could not be put back
type stuckError struct{ file string }

// Error says the file is left changed and that git checkout puts it back.
func (e stuckError) Error() string {
	return fmt.Sprintf("%s is left changed; git checkout puts it back",
		e.file)
}

// runner runs the tests and returns their output and whether they
// passed; it stops them and returns an error if a signal arrives
type runner func(o options, signals <-chan os.Signal) (string, bool, error)

// parseFlags reads args into options. It wants exactly one package, -file,
// -old, and -run given, -count at least one, and -old different from -new.
func parseFlags(args []string) (options, error) {
	var o options
	flags := flag.NewFlagSet("antiproof", flag.ContinueOnError)
	flags.StringVar(&o.file, "file", "", "file to break")
	flags.StringVar(&o.old, "old", "", "text to replace, once")
	flags.StringVar(&o.new, "new", "", "text to put in its place")
	flags.StringVar(&o.run, "run", "", "go test -run pattern")
	flags.IntVar(&o.count, "count", 1, "go test -count")
	flags.StringVar(&o.dir, "C", "", "directory to run go test in")
	if err := flags.Parse(args); err != nil {
		return o, err
	}
	if flags.NArg() != 1 {
		return o, errors.New("want one package")
	}
	o.pkg = flags.Arg(0)
	if o.file == "" || o.old == "" || o.run == "" || o.count < 1 {
		return o, errors.New("-file, -old and -run are needed, " +
			"-count at least 1")
	}
	if o.old == o.new {
		return o, errors.New("-old and -new are the same: no break")
	}
	return o, nil
}

// prove breaks the file, runs the tests and puts the file back, however
// it ends
func prove(o options, run runner, signals <-chan os.Signal) (
	out outcome, output string, err error) {
	original, err := os.ReadFile(o.file)
	if err != nil {
		return 0, "", err
	}
	if n := bytes.Count(original, []byte(o.old)); n != 1 {
		return 0, "", fmt.Errorf("%q occurs %d times in %s, want once",
			o.old, n, o.file)
	}
	info, err := os.Stat(o.file)
	if err != nil {
		return 0, "", err
	}
	broken := bytes.Replace(original, []byte(o.old), []byte(o.new), 1)
	defer func() {
		// put back first, then let a panic go on
		if werr := os.WriteFile(o.file, original,
			info.Mode().Perm()); werr != nil {
			err = stuckError{o.file}
		}
		if p := recover(); p != nil {
			if _, stuck := err.(stuckError); !stuck {
				err = fmt.Errorf("panic: %v", p)
			}
		}
	}()
	if err := os.WriteFile(o.file, broken, info.Mode().Perm()); err != nil {
		os.WriteFile(o.file, original, info.Mode().Perm())
		return 0, "", err
	}
	text, passed, err := run(o, signals)
	if err != nil {
		return 0, text, err
	}
	if passed {
		return testsPassed, text, nil
	}
	return testsFailed, text, nil
}

// goTest runs go test in its own process group, and on a signal kills the
// group and waits for it before returning
func goTest(o options, signals <-chan os.Signal) (string, bool, error) {
	cmd := exec.Command("go", "test", "-count="+strconv.Itoa(o.count),
		"-run", o.run, o.pkg)
	cmd.Dir = o.dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var text bytes.Buffer
	cmd.Stdout, cmd.Stderr = &text, &text
	if err := cmd.Start(); err != nil {
		return "", false, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return judge(text.String(), err)
	case sig := <-signals:
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return text.String(), false, fmt.Errorf("interrupted by %v", sig)
	}
}

// judge reads a finished go test: a test failure is a result; a build
// that failed, or no tests matching, proves nothing
func judge(text string, err error) (string, bool, error) {
	if strings.Contains(text, "[build failed]") ||
		strings.Contains(text, "[setup failed]") {
		return text, false, errors.New("the broken code does not build")
	}
	if err == nil {
		if strings.Contains(text, "no tests to run") {
			return text, false, errors.New("-run matched no tests")
		}
		return text, true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return text, false, nil
	}
	return text, false, err
}

// report prints what happened and gives the exit code
func report(out outcome, text string, err error, stdout,
	stderr io.Writer) int {
	var stuck stuckError
	fmt.Fprint(stdout, text)
	switch {
	case errors.As(err, &stuck):
		fmt.Fprintln(stderr, "antiproof:", err)
		return exitStuck
	case err != nil:
		fmt.Fprintln(stderr, "antiproof:", err)
		return exitToolFail
	case out == testsFailed:
		fmt.Fprintln(stdout, "antiproof: the tests failed under the "+
			"break; the proof holds")
		return exitFailed
	}
	fmt.Fprintln(stdout, "antiproof: the tests PASSED under the break; "+
		"they do not catch it")
	return exitPassed
}
