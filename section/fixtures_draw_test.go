//go:build draw

package section_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// place is the vector of a named row of places.csv.
func place(t *testing.T, name string) maths.Vec {
	t.Helper()
	each := row(t, placesFile, name)
	return maths.V(float(t, each, "x"), float(t, each, "y"),
		float(t, each, "z"))
}

// shade is the light of a named row of colours.csv.
func shade(t *testing.T, name string) maths.Light {
	t.Helper()
	each := row(t, coloursFile, name)
	return maths.RGB(float(t, each, "red"), float(t, each, "green"),
		float(t, each, "blue"))
}

// placed is the subspace of a named place: where it is, turned no way and
// at no scale, which reads as one.
func placed(t *testing.T, name string) maths.Subspace {
	t.Helper()
	var upright maths.Orientation
	return maths.NewSubspace(place(t, name), upright, nothing)
}
