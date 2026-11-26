package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

func TestExactRayLayoutUsesEachFilmPlace(t *testing.T) {
	eye, lens := maths.Subspace{}, film.Pinhole{Wide: 50 * maths.Degree}
	for _, count := range []int{1, 4, 7, 16, 25, 64} {
		lensed := through{eye: eye, lens: lens, width: 3 * count, height: 2,
			exact: count, pictureWidth: 3, pictureHeight: 2}
		for column := 0; column < 3; column++ {
			for index := 0; index < count; index++ {
				across, down := film.Place(index, count, 0)
				want := film.LookThrough(eye, lens, column, 1, across, down, 3, 2, maths.Band{})
				got := lensed.look(column*count+index, 1)
				if math.Abs(got.Dir.X-want.Dir.X) > 1e-12 || math.Abs(got.Dir.Y-want.Dir.Y) > 1e-12 || math.Abs(got.Dir.Z-want.Dir.Z) > 1e-12 {
					t.Fatalf("count=%d column=%d index=%d: got %v want %v", count, column, index, got, want)
				}
			}
		}
	}
}
