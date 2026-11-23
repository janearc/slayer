package session

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/mesh"
)

func TestAuthoredOrcaIsRebuiltAtSceneTime(t *testing.T) {
	given := mustParse(t)
	given.set = "../sets/bioforms/orca-clean-05.yaml"
	initial, _, stagedGiven, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if stagedGiven.live == nil || len(stagedGiven.live.deformations) != 1 {
		t.Fatal("orca is not retained as deforming")
	}
	first, _, err := stagedGiven.live.frameAt(0)
	if err != nil {
		t.Fatal(err)
	}
	later, _, err := stagedGiven.live.frameAt(750 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	again, _, err := stagedGiven.live.frameAt(0)
	if err != nil {
		t.Fatal(err)
	}
	a := first.Things[0].Shape.(*mesh.Mesh).Points()
	b := later.Things[0].Shape.(*mesh.Mesh).Points()
	c := again.Things[0].Shape.(*mesh.Mesh).Points()
	if reflect.DeepEqual(a, b) || !reflect.DeepEqual(a, c) || !reflect.DeepEqual(a, initial.Things[0].Shape.(*mesh.Mesh).Points()) {
		t.Fatal("authored orca frame changed incorrectly")
	}
}

func TestAuthoredOrcaSessionTimesAndShutterRefusal(t *testing.T) {
	path, err := filepath.Abs("../sets/bioforms/orca-clean-05.yaml")
	if err != nil {
		t.Fatal(err)
	}
	prefix := "scene: {set: " + path + "}\nrenderer: cpu\n"
	film := jobFrom(t, prefix+"fps: 2\nduration: {value: 1, unit: s}\n")
	session, err := Open(context.Background(), film, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	source := session.source.(timedFramer)
	a, err := source.inspect(0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := source.inspect(500 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(a.Things[0].Shape.(*mesh.Mesh).Points(), b.Things[0].Shape.(*mesh.Mesh).Points()) {
		t.Fatal("film session froze the orca")
	}
	if session.given.sections.keep {
		t.Fatal("deforming film kept image sections")
	}
	shuttered := jobFrom(t, prefix+"fps: 2\nduration: {value: 1, unit: s}\nshutter: {angle: {value: 180, unit: deg}, glimpses: 2}\n")
	if _, err := Open(context.Background(), shuttered, Stamp{}); err == nil || !strings.Contains(err.Error(), "shutter") {
		t.Fatalf("shutter was not refused: %v", err)
	}
	gpuJob := jobFrom(t, "scene: {set: "+path+"}\nrenderer: gpu\n")
	if _, err := Open(context.Background(), gpuJob, Stamp{}); err == nil || !strings.Contains(err.Error(), "CPU") {
		t.Fatalf("GPU was not refused: %v", err)
	}
}
