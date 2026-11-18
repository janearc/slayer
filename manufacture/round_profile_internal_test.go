package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// coneSides is how many sides the cone in these tests is cut into.
const coneSides = 8 // picked: few, so a strip's turn is large

// unitTolerance is how far from one long a normal may be.
const unitTolerance = 1e-12 // picked: rounding of a few products

// roundCone is a round cone on the y axis, its tip and the middle of its
// base on the axis: a profile with two corners there.
func roundCone() roundProfile {
	return roundProfile{turnedProfile{points: [][coordinatePair]float64{
		{spaceZero, spaceZero}, {spaceOne, spaceZero},
		{spaceZero, spaceOne}}}}
}

// TestRoundProfileKeepsTheFaceNormalOnTheAxis: a corner on the axis
// keeps its face's own normal, and every other corner's normal is one
// long and rises along the axis as its face's own does.
func TestRoundProfileKeepsTheFaceNormalOnTheAxis(test *testing.T) {
	shape := roundCone()
	skin := shape.skin(cut{around: coneSides})
	normals := shape.normals(skin)
	if len(normals) != len(skin) {
		test.Fatalf("%d normals for %d faces", len(normals), len(skin))
	}
	for index, one := range skin {
		flat := one.normal()
		for corner, point := range one {
			normal := normals[index][corner]
			if math.Hypot(point.X, point.Z) == onAxis {
				if normal != flat {
					test.Fatalf("face %d: %v on the axis, not %v",
						index, normal, flat)
				}
				continue
			}
			if math.Abs(normal.Len()-spaceOne) > unitTolerance ||
				math.Abs(normal.Y-flat.Y) > unitTolerance {
				test.Fatalf("face %d: %v off the axis for %v",
					index, normal, flat)
			}
		}
	}
}

// TestTurnedProfileStaysFlat is the anti-test: a turned profile that is
// not round, as the octagonal bus is, carries no normals, and a thing
// made only of such parts is shaded flat as it always was.
func TestTurnedProfileStaysFlat(test *testing.T) {
	plain := roundCone().turnedProfile
	piece := part{shape: plain, fine: cut{around: coneSides},
		space: maths.Subspace{Scale: spaceOne}}
	if _, normals := piece.skinned(); normals != nil {
		test.Fatal("a plain turned profile carried normals")
	}
	grey := thing.Diffuse("test grey", maths.Grey(spaceHalf))
	made := good{parts: []part{piece}, dressed: looks{body: grey}}
	if shadings := made.shadings(); len(shadings) != int(spaceZero) {
		test.Fatalf("a plain turned profile was shaded: %v", shadings)
	}
	round := made
	round.parts = []part{{shape: roundCone(), fine: piece.fine,
		space: piece.space}}
	for name, one := range map[string]good{"plain": made,
		"round": round} {
		things, err := one.things(maths.Local())
		if err != nil {
			test.Fatal(name, err)
		}
		skin, isMesh := things[int(spaceZero)].Shape.(interface {
			Smooth() bool
		})
		if !isMesh || skin.Smooth() != (name == "round") {
			test.Fatalf("%s: shaded smooth is wrong", name)
		}
	}
}
