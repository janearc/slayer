//go:build draw

package hillside_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/hillside"
)

// -cpuprofile writes a profile of the drawing where it is told, and a
// profile with nowhere to go is a hillside not drawn, which gives back
// one.
func TestTheDrawingIsProfiledWhereAsked(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "draw.pprof")
	var said, wrong bytes.Buffer
	small := append(onlyKinds(map[string]string{"boulder": "2"}), "-w", "16",
		"-h", "10", "-n", "1", "-out", filepath.Join(dir, "hill.png"),
		"-cpuprofile")
	if code := hillside.Run(append(small, profile), &said, &wrong, hillside.Stamped("", "", "", "")); code != 0 {
		t.Fatalf("it gave back %d: %s", code, wrong.String())
	}
	if file, err := os.Stat(profile); err != nil || file.Size() == 0 {
		t.Errorf("no profile at %s: %v", profile, err)
	}
	nowhere := filepath.Join(dir, "no", "such", "draw.pprof")
	if code := hillside.Run(append(small, nowhere), &said, &wrong, hillside.Stamped("", "", "", "")); code != 1 {
		t.Errorf("a profile with nowhere to go gave back %d", code)
	}
}

// kindFlags is every kind of thing the command line can ask for a count of.
var kindFlags = []string{"bonfire", "boulder", "campfire", "can", "cobble",
	"fern", "home", "log", "maser", "norway", "pebble", "spotlight", "spruce"}

// onlyKinds is the arguments that ask for none of any kind but these.
func onlyKinds(counts map[string]string) []string {
	var arguments []string
	for _, name := range kindFlags {
		count, asked := counts[name]
		if !asked {
			count = "0"
		}
		arguments = append(arguments, "-"+name, count)
	}
	return arguments
}
