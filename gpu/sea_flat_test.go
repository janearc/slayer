//go:build darwin

package gpu

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/scene/thing"
)

// seaAskedMember is where the fourth number of the row's `sea` member is:
// after the touch, the most steps and the bands, four bytes each.
const seaAskedMember = seaRowMember + 12

// packedAsked is the fourth number a trade sea's row carries, packed with
// scrooge's variable set to the given text.
func packedAsked(t *testing.T, asked string) float64 {
	t.Helper()
	t.Setenv(scrooge.Variable, asked)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: newTradeSea(t),
		Material: &thing.Material{Name: "sea"}})
	encoded, err := encode(world, maths.Local(), film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	return rowFloat(encoded.fields.bytes, seaAskedMember)
}

// The row says one when scrooge is asked for, so the kernel takes the far
// sea as its mean plane.
func TestASeaRowSaysScroogeWhenAsked(t *testing.T) {
	if got := packedAsked(t, "1"); got != 1 {
		t.Errorf("asked, the row's fourth number is %g, want 1", got)
	}
}

// Anti-test: unset, empty or anything but one, the row says nought, and
// the kernel searches the whole way as it did before.
func TestASeaRowSaysNothingWhenNotAsked(t *testing.T) {
	for _, asked := range []string{"", "0", "yes"} {
		if got := packedAsked(t, asked); got != 0 {
			t.Errorf("with %q the row's fourth number is %g, want 0",
				asked, got)
		}
	}
}

// Anti-test: the other numbers of the row are the ones they were: the
// fourth is added to them and moves none.
func TestAskingScroogeMovesNoOtherNumberOfTheRow(t *testing.T) {
	world := func(asked string) []byte {
		t.Setenv(scrooge.Variable, asked)
		made := thing.NewWorld()
		made.Add(thing.Thing{Shape: newTradeSea(t),
			Material: &thing.Material{Name: "sea"}})
		encoded, err := encode(made, maths.Local(), film.Pinhole{}, 0, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		return encoded.fields.bytes
	}
	plain, asked := world("0"), world("1")
	if len(plain) != len(asked) {
		t.Fatalf("rows of %d and %d bytes", len(plain), len(asked))
	}
	for at := range plain {
		if at >= seaAskedMember && at < seaAskedMember+4 {
			continue
		}
		if plain[at] != asked[at] {
			t.Fatalf("byte %d of the row moved when scrooge was asked", at)
		}
	}
}

// The kernel's text takes a sea flat only where its row asks, takes the
// side of a flat hit as sea_flat chose it, and every call of first passes
// the way come it knows. This reads the text, so a rename shows here.
func TestTheKernelTakesASeaFlatOnlyWhereTheRowAsks(t *testing.T) {
	source, err := os.ReadFile("trace.metal")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, want := range []string{
		"sea.sea.w == 0",
		"flat_from < nowhere &&",
		"hit.flat_side, lost",
		"first(world, at, way, 0, leaving_sphere, come)",
		"first(world, from, bent, 0, -1, come)",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the kernel's text lacks %q", want)
		}
	}
}
