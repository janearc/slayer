package manufacture

import (
	"math"

	"github.com/janearc/slayer/geom"
)

type ProperConiferParams struct {
	Height       float64
	TrunkRadius  float64
	Whorls       int
	PerWhorl     int
	CrownStart   float64
	BranchLength float64
	Seed         uint64
}

// DefaultProperConiferParams is the shape of ninja's conifer, ten high.
func DefaultProperConiferParams() ProperConiferParams {
	return ProperConiferParams{
		Height:       10,
		TrunkRadius:  0.32,
		Whorls:       16,
		PerWhorl:     7,
		CrownStart:   0.10,
		BranchLength: 0.30,
		Seed:         0,
	}
}

type Branch struct {
	Base   geom.Vec3
	Dir    geom.Vec3
	Length float64
	Radius float64
}

var goldenAngle = math.Pi * (3 - math.Sqrt(5))

const (
	barkMaterial   = 2
	needleMaterial = 25
)

// unit is a number from zero up to one, the same for the same seed and the same
// three whole numbers.
func unit(seed uint64, a, b, c int) float64 {
	z := seed + 0x9e3779b97f4a7c15*uint64(a+1) +
		0xbf58476d1ce4e5b9*uint64(b+1) + 0x94d049bb133111eb*uint64(c+1)
	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11) / (1 << 53)
}

// across is unit moved to run from minus one to one.
func across(seed uint64, a, b, c int) float64 {
	return 2*unit(seed, a, b, c) - 1
}

// along is how far up the trunk a whorl stands, from zero to one, at its
// middle.
func along(p ProperConiferParams, whorl int) float64 {
	return (float64(whorl) + 0.5) / float64(p.Whorls)
}

// BranchCount is how many branches a whorl has: fewer toward the top.
func BranchCount(p ProperConiferParams, whorl int) int {
	t := along(p, whorl)
	return max(3, int(math.Round(float64(p.PerWhorl)*(1-0.55*t))))
}

// BranchOf is the k-th branch of a whorl: where it starts, which way it points,
// how long and how thick.
func BranchOf(p ProperConiferParams, whorl, k int) Branch {
	t := along(p, whorl)
	height := p.Height * (p.CrownStart + (0.97-p.CrownStart)*t)
	count := BranchCount(p, whorl)

	turn := float64(whorl)*goldenAngle + 2*math.Pi*float64(k)/float64(count) +
		0.35*across(p.Seed, whorl, k, 0)*2*math.Pi/float64(count)

	down := (100 - 38*t + 6*across(p.Seed, whorl, k, 1)) * math.Pi / 180

	length := p.BranchLength * p.Height * (1 - 0.88*t) *
		(1 + 0.15*across(p.Seed, whorl, k, 2))

	trunk := p.TrunkRadius * (1 - height/p.Height)

	return Branch{
		Base: geom.Vec3{0, 0, height},
		Dir: geom.Vec3{
			math.Sin(down) * math.Cos(turn),
			math.Sin(down) * math.Sin(turn),
			math.Cos(down),
		},
		Length: length,
		Radius: math.Max(0.35*trunk, 0.012*p.Height),
	}
}

// ConiferProper is ninja's conifer at base: a trunk and whorls of branches.
func ConiferProper(base geom.Vec3, params ProperConiferParams) []geom.Triangle {
	triangles := make([]geom.Triangle, 0)

	trunk := Cylinder(base, params.Height, params.TrunkRadius, 0.01*params.Height, 8, 2)
	triangles = append(triangles, trunk...)

	for whorl := 0; whorl < params.Whorls; whorl++ {
		count := BranchCount(params, whorl)
		for k := 0; k < count; k++ {
			branch := BranchOf(params, whorl, k)
			triangles = append(triangles, branchGeometry(base, branch)...)
		}
	}

	return triangles
}

// orientedTube is a tube of sides from one point to another, with a radius at
// each end and no cap.
func orientedTube(from, to geom.Vec3, r0, r1 float64, sides int, material int) []geom.Triangle {
	axis := to.Sub(from).Normalize()

	var u geom.Vec3
	if math.Abs(axis[0]) < 0.9 {
		u = geom.Vec3{1, 0, 0}
	} else {
		u = geom.Vec3{0, 1, 0}
	}

	u = u.Sub(axis.Scale(u.Dot(axis))).Normalize()
	v := axis.Cross(u)

	ring := func(centre geom.Vec3, r float64, side int) geom.Vec3 {
		angle := 2 * math.Pi * float64(side) / float64(sides)
		uscale := u.Scale(math.Cos(angle) * r)
		vscale := v.Scale(math.Sin(angle) * r)
		return centre.Add(uscale).Add(vscale)
	}

	var triangles []geom.Triangle
	for side := 0; side < sides; side++ {
		b0 := ring(from, r0, side)
		b1 := ring(from, r0, side+1)
		t0 := ring(to, r1, side)
		t1 := ring(to, r1, side+1)

		triangles = append(triangles, geom.Triangle{
			V:        [3]geom.Vec3{b0, b1, t0},
			Material: material,
		})
		if r1 > 0 {
			triangles = append(triangles, geom.Triangle{
				V:        [3]geom.Vec3{b1, t1, t0},
				Material: material,
			})
		}
	}

	return triangles
}

// branchGeometry is one branch as three drooping pieces of bark, each with a
// sleeve of needles.
func branchGeometry(base geom.Vec3, b Branch) []geom.Triangle {
	triangles := make([]geom.Triangle, 0)

	const pieces = 3
	step := b.Length / pieces

	at := base.Add(b.Base)
	dir := b.Dir

	sleeveSizes := [4]float64{0.10, 0.22, 0.15, 0.0}

	for piece := 0; piece < pieces; piece++ {
		next := at.Add(dir.Scale(step))

		r0 := b.Radius * (1 - 0.25*float64(piece))
		r1 := b.Radius * (1 - 0.25*float64(piece+1))

		bark := orientedTube(at, next, r0, r1, 5, barkMaterial)
		triangles = append(triangles, bark...)

		needleR0 := sleeveSizes[piece] * b.Length
		needleR1 := sleeveSizes[piece+1] * b.Length

		needles := orientedTube(at, next, needleR0, needleR1, 6, needleMaterial)
		triangles = append(triangles, needles...)

		at = next
		dir = dir.Add(geom.Vec3{0, 0, -0.22}).Normalize()
	}

	return triangles
}

// max is the larger of two whole numbers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
