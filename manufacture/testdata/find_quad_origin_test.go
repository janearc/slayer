package manufacture

import (
	"math"
	"math/big"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// exactTriangleCrossSquared interprets the supplied binary64 coordinates as
// exact rationals. It compares area without rounding a square root.
func exactTriangleCrossSquared(a, b, c maths.Vec) *big.Rat {
	values := func(v maths.Vec) [3]*big.Rat {
		return [3]*big.Rat{new(big.Rat).SetFloat64(v.X),
			new(big.Rat).SetFloat64(v.Y), new(big.Rat).SetFloat64(v.Z)}
	}
	aa, bb, cc := values(a), values(b), values(c)
	var u, v [3]*big.Rat
	for i := range 3 {
		u[i] = new(big.Rat).Sub(bb[i], aa[i])
		v[i] = new(big.Rat).Sub(cc[i], aa[i])
	}
	sum := new(big.Rat)
	for i := range 3 {
		j, k := (i+1)%3, (i+2)%3
		cross := new(big.Rat).Sub(new(big.Rat).Mul(u[j], v[k]),
			new(big.Rat).Mul(u[k], v[j]))
		sum.Add(sum, new(big.Rat).Mul(cross, cross))
	}
	return sum
}

// Search actual axial-pole quads from the public optical recipe. The
// independent rational comparison certifies the retained boundary example.
func TestDiagnosticFindQuadCrossOriginBoundary(t *testing.T) {
	const floor = 1e-14
	limit := new(big.Rat).SetFloat64(floor)
	limit.Mul(limit, limit)
	opening := measure.Degrees(30).InRadians()
	for _, sides := range []int{12, 20, 40, 80} {
		levels := max(3, sides/4)
		for side := range sides {
			points := func(radius float64) [4]maths.Vec {
				point := func(side int) maths.Vec {
					angle := opening/2 + (fullTurn-opening)*float64(side)/float64(sides)
					up := math.Pi / 2 * float64(levels-1) / float64(levels)
					return maths.V(radius*math.Cos(up)*math.Sin(angle),
						radius*math.Sin(up), -radius*math.Cos(up)*math.Cos(angle))
				}
				pole := maths.V(0, radius, 0)
				return [4]maths.Vec{point(side), point(side + 1), pole, pole}
			}
			unit := points(1)
			area := unit[1].Sub(unit[0]).Cross(unit[2].Sub(unit[0])).Len()
			if !(area > 0) {
				continue
			}
			critical := math.Sqrt(floor / area)
			for offset := -32; offset <= 32; offset++ {
				radius := critical
				toward := math.Inf(1)
				if offset < 0 {
					toward = 0
				}
				for range int(math.Abs(float64(offset))) {
					radius = math.Nextafter(radius, toward)
				}
				p := points(radius)
				first := p[1].Sub(p[0]).Cross(p[2].Sub(p[0])).Len()
				second := p[1].Sub(p[0]).Cross(p[2].Sub(p[1])).Len()
				if first > floor && second <= floor &&
					exactTriangleCrossSquared(p[0], p[1], p[2]).Cmp(limit) > 0 {
					if got := len(quad(nil, p[0], p[1], p[2], p[3])); got != 1 {
						t.Fatalf("certified pole quad has %d faces", got)
					}
					t.Logf("sides=%d side=%d radius=%.17g first=%.17g second=%.17g points=%#v",
						sides, side, radius, first, second, p)
					return
				}
			}
		}
	}
	t.Log("no certified boundary example found")
}
