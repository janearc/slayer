// Command mutate is mutation testing for slayer's go: it changes one
// thing at a time in a file (an operator, a true, a number), runs the
// tests, and says which changes the tests did not notice. a change the
// tests survive is a test that is missing.
//
// usage, from slayer's root, on files git tracks with nothing
// uncommitted:
//
//	mutate [-flags "-modfile=..."] [-timeout 90s] [-procs 2]
//	       file[:first-last] ...
//
// the tests run on at most procs processors, two unless told otherwise:
// it sets GOMAXPROCS and go test's -p both, since a run is hundreds of
// test runs one after another.
//
// it overwrites each file in turn and puts it back. interrupted, hung up
// on or terminated, it kills the tests it started and puts the file back
// before it exits. if it is killed outright, git checkout puts the file
// back; if the file cannot be put back, it says so and exits 3.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

// main mutates the files named, one change at a time, and reports the
// changes the tests did not notice
func main() {
	flags := flag.String("flags", "", "extra flags for go test, spaced")
	timeout := flag.String("timeout", "90s", "how long one run may take")
	procs := flag.Int("procs", 2, "most processors the tests may use")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "mutate: no files")
		os.Exit(2)
	}
	os.Setenv("GOMAXPROCS", strconv.Itoa(*procs))
	var current atomic.Pointer[session]
	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt, syscall.SIGTERM,
		syscall.SIGHUP)
	go func() {
		<-interrupted
		if s := current.Load(); s != nil {
			stopOrSay(s)
		}
		os.Exit(130)
	}()
	survived := 0
	for _, spec := range flag.Args() {
		lost, err := mutateFile(spec, strings.Fields(*flags), *timeout,
			*procs, &current)
		if err != nil {
			fmt.Fprintln(os.Stderr, "mutate:", err)
			os.Exit(2)
		}
		survived += lost
	}
	if survived > 0 {
		os.Exit(1)
	}
}

// stopOrSay stops a session, and if its file could not be put back, says
// how to put it back and exits 3.
func stopOrSay(s *session) {
	if err := s.stop(); err != nil {
		fmt.Fprintf(os.Stderr, "mutate: %s is left changed (%v); "+
			"git checkout -- %s puts it back\n", s.path, err,
			s.path)
		os.Exit(3)
	}
}

// errStopped is a write or a run asked for after the session stopped.
var errStopped = errors.New("mutate: stopped")

// session is one file being mutated: what it held, and the tests running
// on it, if any are. the main goroutine and a signal both use it, so one
// lock keeps them from writing the file at once, and once stopped,
// nothing more is written or run.
type session struct {
	path     string
	original []byte
	lock     sync.Mutex
	tests    *exec.Cmd
	stopped  bool
}

// write puts content in the file, unless the session has stopped.
func (s *session) write(content []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.stopped {
		return errStopped
	}
	return os.WriteFile(s.path, content, 0o644)
}

// run runs a command in a process group of its own, so that stopping
// kills what it starts as well (go test starts the test binary), and
// gives back what it said. the error is its failing.
func (s *session) run(name string, args ...string) (string, error) {
	var said bytes.Buffer
	command := exec.Command(name, args...)
	command.Stdout, command.Stderr = &said, &said
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	s.lock.Lock()
	if s.stopped {
		s.lock.Unlock()
		return "", errStopped
	}
	err := command.Start()
	if err == nil {
		s.tests = command
	}
	s.lock.Unlock()
	if err != nil {
		return "", err
	}
	err = command.Wait()
	s.lock.Lock()
	defer s.lock.Unlock()
	s.tests = nil
	if s.stopped {
		return "", errStopped
	}
	return said.String(), err
}

// stop kills the running tests and everything they started, and puts
// the file back. stopping twice puts it back twice, which is harmless.
func (s *session) stop() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.stopped = true
	if s.tests != nil {
		syscall.Kill(-s.tests.Process.Pid, syscall.SIGKILL)
	}
	return os.WriteFile(s.path, s.original, 0o644)
}

// mutateFile tries every change to one file, and gives back how many the
// tests survived.
func mutateFile(spec string, flags []string, timeout string, procs int,
	current *atomic.Pointer[session]) (int, error) {
	path, first, last, err := parseSpec(spec)
	if err != nil {
		return 0, err
	}
	if err := clean(path); err != nil {
		return 0, err
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := &session{path: path, original: original}
	current.Store(s)
	defer stopOrSay(s)
	args := append([]string{"test", "-count=1", "-timeout", timeout,
		"-p", strconv.Itoa(procs)}, flags...)
	args = append(args, "./"+filepath.Dir(path)+"/")
	if _, err := s.run("go", args...); err != nil {
		return 0, fmt.Errorf("tests fail before any change: %v", err)
	}
	return try(s, changes(original, first, last), args)
}

// parseSpec splits file:first-last, the lines being optional.
func parseSpec(spec string) (string, int, int, error) {
	path, lines, found := strings.Cut(spec, ":")
	if !found {
		return path, 0, 1 << 30, nil
	}
	low, high, _ := strings.Cut(lines, "-")
	first, errFirst := strconv.Atoi(low)
	last, errLast := strconv.Atoi(high)
	if errFirst != nil || errLast != nil {
		return "", 0, 0, fmt.Errorf("%q is not file:first-last", spec)
	}
	return path, first, last, nil
}

// clean refuses a file git does not track, or one with changes git has
// not got, so that git can always put it back.
func clean(path string) error {
	tracked := exec.Command("git", "ls-files", "--error-unmatch", "--",
		path)
	if tracked.Run() != nil {
		return fmt.Errorf("%s is not tracked by git", path)
	}
	if exec.Command("git", "diff", "--quiet", "--", path).Run() != nil {
		return fmt.Errorf("%s has uncommitted changes", path)
	}
	return nil
}

// try puts each change in the file and runs the tests, saying which
// survived. a change that does not compile says nothing of the tests.
func try(s *session, all []change, args []string) (int, error) {
	killed, invalid, survived := 0, 0, 0
	for _, one := range all {
		if err := s.write(one.apply(s.original)); err != nil {
			return 0, err
		}
		said, failed := s.run("go", args...)
		switch {
		case errors.Is(failed, errStopped):
			return 0, failed
		case failed == nil:
			survived++
			fmt.Printf("SURVIVED %s:%d  %s -> %s\n", s.path,
				one.line, one.from, one.to)
		case strings.Contains(said, "[build failed]") ||
			strings.Contains(said, "[setup failed]"):
			invalid++
		default:
			killed++
		}
	}
	fmt.Printf("%s: %d changes, %d killed, %d survived, %d did not "+
		"compile\n", s.path, len(all), killed, survived, invalid)
	return survived, nil
}
