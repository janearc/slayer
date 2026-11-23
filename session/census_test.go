package session

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/janearc/slayer/scene/thing"
)

// a set's still is counted from the world draw traces, and its camera
// is the eye the still is drawn from, without drawing.
func TestTheCensusOfAStillIsItsStagedWorld(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	setPath, err := filepath.Abs("../sets/widow.yaml")
	if err != nil {
		t.Fatal(err)
	}
	session, err := Open(context.Background(), jobFrom(t,
		"scene: {set: "+setPath+"}\n"), Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	got, err := session.Census(0)
	if err != nil {
		t.Fatal(err)
	}
	still := session.source.(stillFramer)
	want, err := thing.Count(still.world)
	if err != nil || !reflect.DeepEqual(got.World, want) {
		t.Fatalf("census %+v differs from the staged world's %+v (%v)",
			got.World, want, err)
	}
	if got.Eye == nil || got.Unseen != "" ||
		got.Eye.Position != still.given.eye.Origin ||
		got.Eye.Look != still.given.eye.Forward() {
		t.Fatalf("the camera %+v (%s) is not the set's eye %+v", got.Eye,
			got.Unseen, still.given.eye)
	}
	if !reflect.DeepEqual(got.World.Total.Storage,
		mustInspect(t, session).Geometry) {
		t.Errorf("the census and Inspect disagree")
	}
}

// mustInspect is the session's Inspect of its first frame.
func mustInspect(t *testing.T, session *Session) SceneComplexity {
	t.Helper()
	inspected, err := session.Inspect(0)
	if err != nil {
		t.Fatal(err)
	}
	return inspected
}

// a named film is counted at the job's exact time, from the eye its
// frame would be drawn from, and nothing is opened on the gpu.
func TestTheCensusOfANamedFilmIsAtItsTime(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	session, err := Open(context.Background(), jobFrom(t,
		"scene: {named: orbital, with: {far: true}}\nrenderer: cpu\n"+
			"film: {width: 24, height: 14, rays: 1}\nfps: 2\n"+
			"start: {value: 0.5, unit: s}\n"+
			"duration: {value: 1, unit: s}\n"), Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	got, err := session.Census(1)
	if err != nil || got.At != time.Second || got.Eye == nil {
		t.Fatalf("census at %v, eye %+v (%v)", got.At, got.Eye, err)
	}
	source := session.source.(*namedFramer)
	if source.frames != nil {
		t.Fatal("the census opened a gpu drawer")
	}
	if got.Eye.Position != source.built.Eye().Origin {
		t.Errorf("the eye stands at %v, not the scene's %v",
			got.Eye.Position, source.built.Eye().Origin)
	}
}

// the anti-test: a source that cannot say its camera says why and is
// still counted; a closed session counts nothing.
func TestTheCensusSaysWhenItCannotSeeTheCamera(t *testing.T) {
	session := &Session{source: inspectTestSource{world: &thing.World{
		Things: []*thing.Thing{}}}, times: []time.Duration{0}}
	got, err := session.Census(0)
	if err != nil || got.Eye != nil || got.Unseen != unseen {
		t.Fatalf("eye %+v, unseen %q, %v", got.Eye, got.Unseen, err)
	}
	session.closed = true
	if _, err := session.Census(0); err != errClosed {
		t.Errorf("a closed session counted: %v", err)
	}
}
