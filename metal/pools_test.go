//go:build draw && darwin

package metal

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// The sentinel for a whole class of fault. A thread that calls into
// Apple's frameworks is to have an autorelease pool, and a thread of
// Go's has none of its own: a bridge function that touches an object
// outside a pool leaks whatever the object hands back autoreleased, for
// as long as the process runs. One such function kept every buffer of
// every closed session, found on 2026-11-11.
//
// The Objective-C runtime will say so itself. With OBJC_DEBUG_MISSING_POOLS
// set it writes a line for every object autoreleased with no pool in
// place. So this package's tests are run once more with it set, and any
// such line fails: for every bridge function the tests reach, the ones
// not written yet among them.

// poolSwitch is the runtime's switch for saying when an object is
// autoreleased with no pool in place, and poolShout what it says.
const (
	poolSwitch = "OBJC_DEBUG_MISSING_POOLS"
	poolShout  = "MISSING POOLS"
)

// sentinelName is this file's own test, left out of the watched run so
// the run does not start another of itself.
const sentinelName = "TestNothingIsAutoreleasedWithNoPool"

// leaked is the class of every object a run's output says was
// autoreleased with no pool in place, each class once, sorted.
func leaked(output string) []string {
	var classes []string
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.Contains(line, poolShout) {
			continue
		}
		class := "an object the runtime did not name"
		if _, after, named := strings.Cut(line, " of class "); named {
			class, _, _ = strings.Cut(after, " ")
		}
		if !slices.Contains(classes, class) {
			classes = append(classes, class)
		}
	}
	slices.Sort(classes)
	return classes
}

// Every test of this package, run again with the runtime watching: no
// object is autoreleased with no pool in place.
func TestNothingIsAutoreleasedWithNoPool(t *testing.T) {
	watched := exec.Command(os.Args[0], "-test.run", ".",
		"-test.skip", "^"+sentinelName+"$", "-test.count", "1")
	watched.Env = append(os.Environ(), poolSwitch+"=YES")
	var said bytes.Buffer
	watched.Stderr = &said
	watched.Stdout = io.Discard
	ran := watched.Run()
	if classes := leaked(said.String()); len(classes) != 0 {
		t.Errorf("objects were autoreleased with no pool in place, and leak: %s. "+
			"A bridge function Go calls touches them outside @autoreleasepool.",
			strings.Join(classes, ", "))
	}
	if ran != nil {
		// the tests' own failures are the ordinary run's to report; a
		// watched run that could not run at all would watch nothing
		if _, exited := ran.(*exec.ExitError); !exited {
			t.Fatalf("the watched run did not run: %v", ran)
		}
	}
}

// the anti-test of the one above: the sentinel reads the runtime's own
// words. This is the line it wrote on 2026-11-11 against the bridge as
// it then was, and a second of another class.
func TestTheSentinelReadsTheRuntimesShout(t *testing.T) {
	night := "objc[38767]: MISSING POOLS: (0x1f4a3c000) Object 0x7fa3d2c04a80 of class AGXG15XFamilyBuffer autoreleased with no pool in place - just leaking - break on objc_autoreleaseNoPool() to debug\n" +
		"ok so far\n" +
		"objc[38767]: MISSING POOLS: (0x1f4a3c000) Object 0x7fa3d2c04b00 of class AGXG15XFamilyBuffer autoreleased with no pool in place - just leaking - break on objc_autoreleaseNoPool() to debug\n" +
		"objc[38767]: MISSING POOLS: (0x1f4a3c000) Object 0x7fa3d2c05000 of class __NSArrayM autoreleased with no pool in place - just leaking\n"
	got := leaked(night)
	want := []string{"AGXG15XFamilyBuffer", "__NSArrayM"}
	if !slices.Equal(got, want) {
		t.Errorf("the sentinel read %q out of the runtime's shouts, not %q", got, want)
	}
	if quiet := leaked("PASS\nok  \tgithub.com/janearc/slayer/metal\t2.672s\n"); len(quiet) != 0 {
		t.Errorf("the sentinel heard a leak in a clean run: %q", quiet)
	}
	if unnamed := leaked("objc[1]: MISSING POOLS: something the runtime words another way\n"); len(unnamed) != 1 {
		t.Errorf("a shout that names no class was not counted: %q", unnamed)
	}
}
