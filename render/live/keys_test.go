package live

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"go.uber.org/mock/gomock"
)

// idle is a watcher of a job that has read and is staged, not running,
// so a test can press its keys one at a time.
func idle(t *testing.T, text string) *Watcher {
	t.Helper()
	control := gomock.NewController(t)
	now := &clock{now: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)}
	jobPath := jobFile(t, text)
	stager := NewMockStager(control)
	screen := screenOf(control, make(chan terminal.Event), nil, nil)
	watching := watcherOn(t, jobPath, stager,
		screen, now,
		make(chan time.Time))
	watching.reread()
	frames := filmFrames
	if watching.still() {
		frames = one
	}
	facts := factsOf(stageOf(control, frames, nil))
	watching.opened(facts)
	watching.pending = Unchanged
	return watching
}

// pressed is the key of bytes as a terminal sends them.
func pressed(t *testing.T, bytes string) terminal.Key {
	t.Helper()
	var decoder terminal.Decoder
	keys := decoder.Feed([]byte(bytes))
	if len(keys) != one {
		t.Fatalf("%q decoded as %d keys", bytes, len(keys))
	}
	return keys[none]
}

// left and right step a film by a frame, held at its ends; [ and ] by a
// second of frames.
func TestKeysStepAFilm(t *testing.T) {
	watching := idle(t, filmJob)
	if got := watching.key(pressed(t, "\x1b[C")); got != Redraw ||
		watching.view.index != 1 {
		t.Errorf("right gave %v at frame %d", got, watching.view.index)
	}
	if got := watching.key(pressed(t, "]")); got != Redraw ||
		watching.view.index != 1+filmRate {
		t.Errorf("] gave %v at frame %d", got, watching.view.index)
	}
	watching.key(pressed(t, "]"))
	if watching.view.index != filmFrames-one {
		t.Errorf("past the end is frame %d", watching.view.index)
	}
	watching.key(pressed(t, "\x1b[D"))
	if watching.view.index != filmFrames-pair {
		t.Errorf("left is frame %d", watching.view.index)
	}
}

// anti-test of TestKeysStepAFilm
// at the start, left changes nothing and costs nothing; a
// still's left is a note, not a step.
func TestKeysHoldAtTheEnds(t *testing.T) {
	watching := idle(t, filmJob)
	if got := watching.key(pressed(t, "\x1b[D")); got != Unchanged ||
		watching.view.index != none {
		t.Errorf("left at the start gave %v at %d", got,
			watching.view.index)
	}
	still := idle(t, stillJob)
	moved := still.key(pressed(t, "\x1b[C"))
	footing := strings.Join(still.foot.lines(), " ")
	if moved != Unchanged || !strings.Contains(footing, "one frame") {
		t.Errorf("right on a still gave %v, %q", moved, still.foot.lines())
	}
	if got := still.key(pressed(t, " ")); got != Unchanged {
		t.Errorf("space on a still gave %v", got)
	}
}

// a still's [ and ] move its moment by a second and restage, never
// before nought.
func TestKeysMoveAStillsMoment(t *testing.T) {
	watching := idle(t, stillJob)
	if got := watching.key(pressed(t, "]")); got != Restage ||
		watching.view.moment != time.Second {
		t.Errorf("] gave %v at %v", got, watching.view.moment)
	}
	if at, said := watching.wanted.Clock().At(); !said ||
		at != time.Second {
		t.Errorf("the job shown is at %v, %v", at, said)
	}
	watching.key(pressed(t, "["))
	watching.key(pressed(t, "["))
	if watching.view.moment != none {
		t.Errorf("two back from a second is %v", watching.view.moment)
	}
}

// e and E move exposure by thirds and only develop again; c chooses the
// next camera and restages; f draws full size; ? helps.
func TestKeysChangeTheView(t *testing.T) {
	watching := idle(t, stillJob)
	if got := watching.key(pressed(t, "E")); got != Redevelop ||
		watching.view.thirds != one {
		t.Errorf("E gave %v at %d thirds", got, watching.view.thirds)
	}
	watching.key(pressed(t, "e"))
	watching.key(pressed(t, "e"))
	if watching.view.thirds != -one {
		t.Errorf("e twice is %d thirds", watching.view.thirds)
	}
	if got := watching.key(pressed(t, "c")); got != Restage ||
		watching.view.camera != "near" {
		t.Errorf("c gave %v, camera %q", got, watching.view.camera)
	}
	watching.key(pressed(t, "c"))
	watching.key(pressed(t, "c"))
	if watching.view.camera != "near" {
		t.Errorf("c round the cameras is %q", watching.view.camera)
	}
	if got := watching.key(pressed(t, "f")); got != Redraw ||
		watching.view.mode != full {
		t.Errorf("f gave %v in mode %d", got, watching.view.mode)
	}
	if got := watching.key(pressed(t, "x")); got != Redraw ||
		watching.view.mode != previewing {
		t.Errorf("a key after f gave %v in mode %d", got,
			watching.view.mode)
	}
	watching.key(pressed(t, "?"))
	if got := watching.key(pressed(t, "\x1b[C")); got != Unchanged ||
		watching.view.mode != previewing {
		t.Errorf("a key after ? gave %v in mode %d", got,
			watching.view.mode)
	}
}

// anti-test of TestKeysChangeTheView
// anti-test of TestKeysQuit
// byte 0x1d, the host's key, does nothing in any mode: it does
// not quit, step, or leave the help; nor does a key with no action.
func TestTheHostsKeyIsNeverBound(t *testing.T) {
	watching := idle(t, filmJob)
	host := pressed(t, "\x1d")
	if host.Name() != terminal.KeyHost {
		t.Fatalf("0x1d decoded as %v", host.Name())
	}
	before := watching.view
	if got := watching.key(host); got != Unchanged ||
		watching.view != before || watching.quit {
		t.Errorf("the host's key gave %v", got)
	}
	watching.key(pressed(t, "?"))
	watching.key(host)
	if watching.view.mode != helping {
		t.Error("the host's key left the help")
	}
	if got := watching.key(pressed(t, "\x1b[5~")); got != Unchanged {
		t.Errorf("page-up gave %v", got)
	}
}

// q and control-c quit, from any mode.
func TestKeysQuit(t *testing.T) {
	for _, bytes := range []string{"q", "\x03"} {
		watching := idle(t, stillJob)
		watching.key(pressed(t, "?"))
		watching.key(pressed(t, bytes))
		if !watching.quit {
			t.Errorf("%q did not quit", bytes)
		}
	}
}

// anti-test of TestKeysChangeTheView
// anti-test of TestKeysQuit
// a scene with no cameras by name says so and changes
// nothing; Q is not q.
func TestKeysThatCannotActSaySo(t *testing.T) {
	watching := idle(t, stillJob)
	watching.facts.cameras = nil
	if got := watching.key(pressed(t, "c")); got != Unchanged ||
		watching.view.camera != "" {
		t.Errorf("c with no cameras gave %v", got)
	}
	watching.key(pressed(t, "Q"))
	if watching.quit {
		t.Error("Q quit")
	}
}

// space plays: the frame shown is where the clock says, frames that
// could not be drawn in time dropped, round past the end.
func TestPlayDropsFramesItCannotDraw(t *testing.T) {
	watching := idle(t, filmJob)
	now := &clock{now: watching.given.Now()}
	watching.given.Now = now.Now
	if got := watching.key(pressed(t, " ")); got != Redraw ||
		!watching.view.playing {
		t.Fatalf("space gave %v", got)
	}
	now.pass(time.Second)
	if got := watching.playedTo(); got != filmRate {
		t.Errorf("a second in is frame %d, want %d", got, filmRate)
	}
	now.pass(2 * time.Second)
	if got := watching.playedTo(); got != filmRate {
		t.Errorf("past the end is frame %d, want round to %d", got,
			filmRate)
	}
}

// anti-test of TestPlayDropsFramesItCannotDraw
// play does not step a frame at a time when time has gone
// further; and space again stops it.
func TestPlayIsNotAStep(t *testing.T) {
	watching := idle(t, filmJob)
	now := &clock{now: watching.given.Now()}
	watching.given.Now = now.Now
	watching.key(pressed(t, " "))
	now.pass(time.Second / pair)
	if got := watching.playedTo(); got == one {
		t.Error("half a second in is the next frame, not two on")
	}
	watching.key(pressed(t, " "))
	if watching.view.playing {
		t.Error("space again did not stop play")
	}
}
