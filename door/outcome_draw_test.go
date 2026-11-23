//go:build draw && darwin

package door

import (
	"context"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/sources"
)

// the size the city preview is drawn at, and the camera of the second
// test: the first camera needs no name, for it is the one drawn when none
// is asked for.
const (
	previewWidth  = "144"    // defined: pixels across
	previewHeight = "81"     // defined: pixels down
	streetCamera  = "street" // defined: a canal alley, lamps thick in fog
	scroogeOn     = "1"      // defined: the value that asks for scrooge
)

// futianShore is sets/futian-shore.yaml, absolute.
func futianShore(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../sets/futian-shore.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// drawnFutian draws sets/futian-shore.yaml through the door's draw verb,
// as slayer draw does with --renderer gpu and --fresh at the preview's
// size, with scrooge on and the cache folder given, which the caller
// makes. camera names the camera, or is empty for the first. --fresh
// takes up no kept rows of the picture and serves no kept still, and
// clears nothing the cache keeps of what is worked out from the sources,
// so a second draw into the same cache is warm. the set reads surveyed
// sources from the shelf the process's environment names, so the test
// declines to run where it names none, and says why in the refusal's
// words: it is skipped, not failed. it gives the path of the picture
// drawn and the wall clock time the draw took, staging included.
func drawnFutian(t *testing.T, camera, cache string) (string,
	time.Duration) {
	t.Helper()
	shelf, err := sources.Here()
	if refusal := shelfRefusal(err); refusal != "" {
		t.Skip(refusal)
	}
	t.Setenv(scrooge.Variable, scroogeOn)
	t.Setenv("SLAYER_CACHE", cache)
	directory := t.TempDir()
	picture := filepath.Join(directory, "picture.png")
	args := []string{futianShore(t), "--width", previewWidth, "--height",
		previewHeight, "--renderer", "gpu", "--fresh", "--out", picture}
	if camera != "" {
		args = append(args, "--camera", camera)
	}
	env := envIn(t, context.Background(), directory, "SLAYER_CACHE", cache,
		sources.Placed, shelf)
	began := time.Now()
	drew(t, env, args...)
	return picture, time.Since(began)
}

// peakResident is the most resident memory the test process has held, in
// bytes. on darwin getrusage gives Maxrss in bytes.
func peakResident(t *testing.T) int64 {
	t.Helper()
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		t.Fatal(err)
	}
	return usage.Maxrss
}

// the first camera of the city set draws a preview twice into one empty
// cache, and each draw is inside its time, the cold one inside the
// owner's bar and the warm one inside three times what it measured, and
// the two inside one memory limit, so a change that is right alone cannot
// make the preview slow or large again.
//
// the memory mark is the whole test process's high-water mark, not the
// draws': whatever ran before them in the same process counts. the test
// is meant to be run by itself, with -run, and its limit is one and a
// half times what the draws alone measured, which is the slack for what
// the process held before them.
func TestACityPreviewDrawsInsideItsTimeAndMemoryBudget(t *testing.T) {
	cache := t.TempDir()
	_, cold := drawnFutian(t, "", cache)
	_, warm := drawnFutian(t, "", cache)
	took := drawBudget{cold: cold, warm: warm, resident: peakResident(t)}
	if faults := budgetFaults(took, previewLimits); len(faults) != nothing {
		t.Error(budgetReport(took, faults))
	}
}

// the haze of the canal alley has few specks. a lamp in fog that is
// sampled with too great a weight leaves single pixels far brighter than
// the pixels round them, and a change that is right alone cannot bring
// them back without this test counting them.
func TestLampsInFogLeaveFewSpecksInTheHaze(t *testing.T) {
	path, _ := drawnFutian(t, streetCamera, t.TempDir())
	if count := specks(pictureOf(t, path)); count > speckLimit {
		t.Error(speckReport(count))
	}
}
