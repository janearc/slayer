//go:build draw

package session

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAuthoredOrcaActuallyDrawsDifferentFilmFrames(t *testing.T) {
	path, err := filepath.Abs("../sets/bioforms/orca-clean-05.yaml")
	if err != nil {
		t.Fatal(err)
	}
	wanted := jobFrom(t, "scene: {set: "+path+"}\nrenderer: cpu\nfilm: {width: 32, height: 20, rays: 1}\nfps: 2\nduration: {value: 1, unit: s}\n")
	session, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	a, err := session.Frame(context.Background(), 0, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := session.Frame(context.Background(), 1, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	again, err := session.Frame(context.Background(), 0, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(a.lit.drawn.Pix, b.lit.drawn.Pix) || !reflect.DeepEqual(a.lit.drawn.Pix, again.lit.drawn.Pix) {
		t.Fatal("film frame pixels did not change with time and replay")
	}
}
