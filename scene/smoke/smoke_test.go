package smoke

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/world"
)

// A smoke's Reads opens its inputs out: a path passes through, a $NAME
// becomes the files in the folder the variable names, a $NAME/file that
// one file in it, sorted, and an unset name reads nothing. Smoke
// thirty-eight declares the blue marble.
func TestASmokeSaysWhatItReads(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"black.jpg", "blue.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("SLAYER_MARBLE", dir)
	s := Smoke{Inputs: []string{"$SLAYER_MARBLE", "/etc/hosts"}}
	got, err := s.Reads()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/etc/hosts", filepath.Join(dir, "black.jpg"), filepath.Join(dir, "blue.jpg")}
	// Reads sorts, and where the temporary folder sorts against /etc
	// depends on who runs the test: under /var it comes after, under a
	// home folder before
	slices.Sort(want)
	if len(got) != len(want) {
		t.Fatalf("reads %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("reads %v, want %v", got, want)
		}
	}
	t.Setenv("SLAYER_MARBLE", "")
	if got, _ := s.Reads(); len(got) != 1 {
		t.Errorf("with the name unset, reads %v", got)
	}
	marble, _ := Get(38)
	if len(marble.Inputs) != 1 || marble.Inputs[0] != "$SLAYER_MARBLE/blue.jpg" {
		t.Errorf("the marble smoke declares %v", marble.Inputs)
	}
	stars, _ := Get(50)
	if len(stars.Inputs) != 1 || stars.Inputs[0] != "$SLAYER_STARS/ybsc5.gz" {
		t.Errorf("the smoke of Webb and its stars declares %v", stars.Inputs)
	}
	for _, s := range All() {
		if s.Number != 38 && s.Number != 50 && len(s.Inputs) > 0 {
			t.Errorf("smoke %d declares inputs %v", s.Number, s.Inputs)
		}
	}
}

// Smoke thirty-eight reads blue.jpg and nothing else in the folder, so
// that is all its Reads says: black.jpg, which only a test reads, and any
// other file there are not its inputs. The file is named whether or not
// it is there, since Reads opens nothing.
func TestTheBlueMarbleSmokeReadsBlueAlone(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"black.jpg", "blue.jpg", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, ok := Get(38)
	if !ok {
		t.Fatal("no smoke 38")
	}
	t.Setenv("SLAYER_MARBLE", dir)
	got, err := s.Reads()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "blue.jpg"); len(got) != 1 || got[0] != want {
		t.Errorf("reads %v, want just %v", got, want)
	}
	empty := t.TempDir()
	t.Setenv("SLAYER_MARBLE", empty)
	got, err = s.Reads()
	if want := filepath.Join(empty, "blue.jpg"); err != nil || len(got) != 1 || got[0] != want {
		t.Errorf("in a folder without it, reads %v, %v, want just %v", got, err, want)
	}
	t.Setenv("SLAYER_MARBLE", "")
	if got, err := s.Reads(); err != nil || len(got) != 0 {
		t.Errorf("with the name unset, reads %v, %v", got, err)
	}
}

// refusal is the error a call panics with, which is how a smoke refuses
// by name through placed; nil when the call does not panic. A panic that
// is not an error is returned as one that says so.
func refusal(call func()) (err error) {
	defer func() {
		if caught := recover(); caught != nil {
			var ok bool
			if err, ok = caught.(error); !ok {
				err = fmt.Errorf("panicked with %v, which is not an error", caught)
			}
		}
	}()
	call()
	return nil
}

// watched runs call on this goroutine and ends the whole test run, with status
// 2, if the call takes longer than limit or the heap passes 256 MB: a refusal
// that does not come would otherwise loop forever or fill the machine's memory,
// and a failing test cannot stop a loop.
func watched(limit time.Duration, call func()) {
	runtime.GC()
	start, done := time.Now(), make(chan struct{})
	defer close(done)
	go func() {
		tick := time.NewTicker(50 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.HeapAlloc > 256<<20 {
					fmt.Fprintf(os.Stderr, "watched: the heap is %d MB, past 256\n", m.HeapAlloc>>20)
					os.Exit(2)
				}
				if time.Since(start) > limit {
					fmt.Fprintf(os.Stderr, "watched: still running after %v\n", limit)
					os.Exit(2)
				}
			}
		}
	}()
	call()
}

// Smoke thirty-seven refuses a radius it cannot fall to: zero, below it, no
// number, infinite, or past where the chart carries the camera. Each is refused
// by name inside a second, where the fall to one of them would otherwise go on
// forever.
func TestSmoke37RefusesARadiusItCannotFallTo(t *testing.T) {
	for _, radius := range []float64{0, -1, math.NaN(), math.Inf(1), 0.005} {
		var err error
		watched(time.Second, func() {
			err = refusal(func() { thirtySevenFrame(radius, false, false) })
		})
		if err == nil || !strings.Contains(err.Error(), "thirtySevenFrame") {
			t.Errorf("a radius of %v: refused with %v", radius, err)
		}
	}
}

// A Find that finds nothing is refused by name, with what was looked for
// and where, and a Find that finds a tile gives that tile.
func TestAFindThatFindsNothingIsRefused(t *testing.T) {
	plain := world.New(twelveTerrain)
	toward := maths.V(0.35, 0.1, 1)
	err := refusal(func() {
		found(plain, "a tile of no biome", toward, func(world.Biome) bool { return false })
	})
	if err == nil || !strings.Contains(err.Error(), "a tile of no biome") ||
		!strings.Contains(err.Error(), fmt.Sprint(toward)) {
		t.Errorf("a want nothing meets: refused with %v", err)
	}
	tile := plain.Tiling.Nearest(toward)
	got := found(plain, "the tile itself", toward, func(b world.Biome) bool {
		return b == plain.Biome(tile)
	})
	if got != tile {
		t.Errorf("found %v, the nearest tile is %v and wants itself", got, tile)
	}
}
