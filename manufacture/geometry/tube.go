package geometry

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

var (
	ErrCut        = fmt.Errorf("manufacture: a ring is cut into %d to %d sides", 3, 64)
	ErrTubeLength = errors.New("manufacture: a tube needs a length " +
		"above nought, with an end")
	ErrTubeFoot = errors.New("manufacture: a tube needs a radius at " +
		"its foot above nought, with an end; a cone narrows to its tip")
	ErrTubeTip = errors.New("manufacture: a tube's radius at its tip " +
		"is nought or above, with an end")
)

// CheckTube preserves the maker's dimensional refusals.
func CheckTube(length, foot, tip float64) error {
	positive := func(n float64) bool { return n > 0 && !math.IsInf(n, 0) }
	if !positive(length) {
		return ErrTubeLength
	}
	if !positive(foot) {
		return ErrTubeFoot
	}
	if tip != 0 && !positive(tip) {
		return ErrTubeTip
	}
	return nil
}

// TubeRing gives the ordered corners of a round ring or flat blade.
func TubeRing(radius, height float64, sides int, flat bool) []maths.Vec {
	if flat {
		return []maths.Vec{maths.V(radius, height, 0), maths.V(-radius, height, 0)}
	}
	corners := make([]maths.Vec, sides)
	for side := range corners {
		angle := 2 * math.Pi * float64(side) / float64(sides)
		corners[side] = maths.V(math.Cos(angle)*radius, height, -math.Sin(angle)*radius)
	}
	return corners
}

// TubeWall preserves the order of the original tube's side faces.
func TubeWall[F Triangle](length, footRadius, tipRadius float64, sides int, flat bool) []F {
	foot, tip := TubeRing(footRadius, 0, sides, flat), TubeRing(tipRadius, length, sides, flat)
	count := len(foot)
	if flat {
		count = 1
	}
	var skin []F
	for side := 0; side < count; side++ {
		next := (side + 1) % len(foot)
		skin = append(skin, F{foot[side], foot[next], tip[side]})
		if tipRadius != 0 {
			skin = append(skin, F{foot[next], tip[next], tip[side]})
		}
	}
	return skin
}

// TubeCaps preserves the order of the original tube's end faces.
func TubeCaps[F Triangle](length, footRadius, tipRadius float64, sides int, flat bool) []F {
	if flat {
		return nil
	}
	foot, tip := TubeRing(footRadius, 0, sides, false), TubeRing(tipRadius, length, sides, false)
	var skin []F
	for side := range foot {
		next := (side + 1) % len(foot)
		skin = append(skin, F{{}, foot[next], foot[side]})
		if tipRadius != 0 {
			skin = append(skin, F{maths.V(0, length, 0), tip[side], tip[next]})
		}
	}
	return skin
}

// TubeMesh builds the checked flat-shaded mesh of a round tube.
func TubeMesh(length, foot, tip float64, sides int) (*mesh.Mesh, error) {
	if err := CheckTube(length, foot, tip); err != nil {
		return nil, err
	}
	skin := append(TubeWall[[3]maths.Vec](length, foot, tip, sides, false), TubeCaps[[3]maths.Vec](length, foot, tip, sides, false)...)
	return FaceMesh(skin, nil)
}
