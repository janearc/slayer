package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// the numbers Boarded is drawn with, each picked and held in
// picked-numbers.yaml
var (
	// boardTone is how far a board's own tone may differ from the
	// concrete's, as a share of the stain's difference from it.
	boardTone = pickedNumber("board-tone")
	// grainAlong is how many board heights long a streak of grain is.
	grainAlong = pickedNumber("grain-along")
	// grainAcross is how many streaks of grain fit in a board's height.
	grainAcross = pickedNumber("grain-across")
	// grainShare is how much of the stain's difference the darkest grain
	// takes.
	grainShare = pickedNumber("grain-share")
	// jointShare is how much of the stain's difference a joint between
	// two boards takes.
	jointShare = pickedNumber("joint-share")
	// grainOctaves is how many sizes of grain lie one on another.
	grainOctaves = int(pickedNumber("grain-octaves"))
	// grainRange is about how far either side of nought the noise runs.
	grainRange = pickedNumber("grain-range")
	// tiny keeps a width from being nought where it is divided by.
	tiny = pickedNumber("width-floor")
)

// Boarded is board-marked concrete: concrete that was poured against
// timber boards and keeps their imprint, the grain of each board, a seam
// where two meet, and a tone a little different from board to board. The
// boards run level, along x, and are Board tall in the thing's own space;
// Joint is how wide a seam is, in the same units; Seed picks the tones.
// Stain is the colour the seams and the grain darken toward.
type Boarded struct {
	Concrete, Stain maths.Light
	Board, Joint    float64
	Seed            uint64
}

// board is how far up its board a height is, nought at the foot of a
// board and one at its top, and which board it is.
func (boarded Boarded) board(height float64) (float64, int64) {
	row := math.Floor(height / boarded.Board)
	return height/boarded.Board - row, int64(row)
}

// At is the concrete at the point: the board's tone, the grain on it, and
// the seam if the point is within a joint of the board's edge.
func (boarded Boarded) At(point maths.Vec) maths.Light {
	if !(boarded.Board > nought) {
		return boarded.Concrete
	}
	within, row := boarded.board(point.Y)
	tone := (noise.Hash(boarded.Seed, row)*pair - unity) * boardTone
	grain := noise.Fractal(boarded.Seed+unity,
		point.X/(grainAlong*boarded.Board),
		point.Y/(boarded.Board/grainAcross),
		point.Z/(grainAlong*boarded.Board), grainOctaves)
	lit := math.Min(unity, math.Max(nought,
		(grain+grainRange)/(pair*grainRange)))
	dark := math.Max(nought, tone) + grainShare*lit
	seam := boarded.Joint / boarded.Board
	if within < seam || within > unity-seam {
		dark = jointShare
	}
	return boarded.Concrete.Scale(unity - dark).Add(boarded.Stain.Scale(dark))
}

// Within is the concrete at the point for a pixel footprint wide: seams
// narrower than a pixel fade to the share of the surface they cover, and
// the grain, which is finer, to its average.
func (boarded Boarded) Within(point maths.Vec, footprint float64) maths.Light {
	return kept(boarded.At(point), boarded.Average(),
		Detail(footprint, boarded.Joint))
}

// Average is the colour of the surface seen from so far off that its seams
// and its grain are finer than a pixel: the concrete darkened toward the
// stain by the share of the surface the seams cover, each a joint's share,
// and by the grain's share on average, which is half of its darkest.
func (boarded Boarded) Average() maths.Light {
	seams := math.Min(unity, pair*boarded.Joint/math.Max(boarded.Board, tiny))
	dark := seams*jointShare + grainShare/pair
	return boarded.Concrete.Scale(unity - dark).Add(boarded.Stain.Scale(dark))
}
