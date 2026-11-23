package set_test

import (
	"testing"

	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// filmRead is the canal's set with its film's rays line changed to now,
// or the refusal it meets.
func filmRead(t *testing.T, now string) (set.Set, error) {
	t.Helper()
	return rowRead(t, "    rays: 1\n", now)
}

// filterOf is the filter of the film of a set that read.
func filterOf(t *testing.T, read set.Set, err error) thing.Filter {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	film, has := read.Film()
	if !has {
		t.Fatal("the set has no film")
	}
	return film.Filter()
}

// A film that says no filter has the tent, and one that says a filter
// has the one it says.
func TestAFilmsFilterIsTheTentUnlessItSaysAnother(t *testing.T) {
	for _, each := range []struct {
		now  string
		want thing.Filter
	}{{"    rays: 4\n", thing.TentFilter},
		{"    rays: 4\n    filter: tent\n", thing.TentFilter},
		{"    rays: 4\n    filter: trapezoid\n", thing.TrapezoidFilter},
		{"    rays: 4\n    filter: box\n", thing.BoxFilter},
		{"    rays: 4\n    filter: grid\n", thing.GridFilter}} {
		read, err := filmRead(t, each.now)
		if got := filterOf(t, read, err); got != each.want {
			t.Errorf("with %q the film's filter is %d, want %d",
				each.now, got, each.want)
		}
	}
}

// Anti: a filter that is none of the four is refused, and not read as
// the tent.
func TestAFilmsFilterIsOneOfTheTwo(t *testing.T) {
	_, err := filmRead(t, "    rays: 4\n    filter: gauss\n")
	if err == nil {
		t.Error("a film whose filter is gauss was read")
	}
}
