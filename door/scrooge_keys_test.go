package door

import (
	"image"
	"testing"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// A still's rows and finished picture are kept and found under the same
// scrooge setting, including a still that asks for a region.
func TestStillKeysKeepScroogeSetting(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "still.yaml", stillJob(t, ""))
	wanted, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	region, err := wanted.With(job.Region(image.Rect(1, 1, 3, 3)))
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range []job.Job{wanted, region} {
		for _, keyOf := range []func(job.Job) diskcache.Key{rowsKey, pictureKey} {
			t.Setenv(scrooge.Variable, "1")
			first := keyOf(each)
			again := keyOf(each)
			if first.Err() != nil || again.Err() != nil {
				t.Fatalf("key refusal: %v, %v", first.Err(), again.Err())
			}
			if first.Name() != again.Name() {
				t.Error("the same setting gave different keys")
			}
			store, err := diskcache.Open(t.TempDir(), first.Kind())
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Put(first, []byte("kept")); err != nil {
				t.Fatal(err)
			}
			hit, err := store.Get(again)
			if err != nil || !hit.Found() {
				t.Fatalf("same setting missed: %v", err)
			}
		}
	}
}

// Anti: a setting that changes geometry cannot take up the other setting's
// rows or finished picture; every non-1 value means the same disabled mode.
func TestStillKeysSeparateScroogeModes(t *testing.T) {
	directory := t.TempDir()
	wanted, err := job.Read(written(t, directory, "still.yaml", stillJob(t, "")))
	if err != nil {
		t.Fatal(err)
	}
	for _, keyOf := range []func(job.Job) diskcache.Key{rowsKey, pictureKey} {
		t.Setenv(scrooge.Variable, "1")
		on := keyOf(wanted)
		t.Setenv(scrooge.Variable, "")
		off := keyOf(wanted)
		t.Setenv(scrooge.Variable, "other")
		alsoOff := keyOf(wanted)
		if on.Err() != nil || off.Err() != nil || alsoOff.Err() != nil {
			t.Fatalf("key refusal: %v, %v, %v", on.Err(), off.Err(), alsoOff.Err())
		}
		if on.Name() == off.Name() {
			t.Error("on and off share a key")
		}
		if off.Name() != alsoOff.Name() {
			t.Error("two disabled settings differ")
		}
		store, err := diskcache.Open(t.TempDir(), on.Kind())
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Put(on, []byte("kept")); err != nil {
			t.Fatal(err)
		}
		hit, err := store.Get(off)
		if err != nil || hit.Found() {
			t.Fatalf("other setting found an entry: %v", err)
		}
	}
}
