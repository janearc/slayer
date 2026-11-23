package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// CoralSpec controls an authored branching colony in metres. Levels counts
// branching generations, not age. Dimensions describe the first branch and its base.
type CoralSpec struct {
	Height, Radius measure.Length
	Levels         int
	Corallites     bool // authored skeletal cups; not animated living polyps
	FineSurface    bool // use the same finer mesh for matched plain controls
}

// DefaultCoral gives a small branching colony for close review.
func DefaultCoral() CoralSpec {
	return CoralSpec{Height: measure.Metres(.14), Radius: measure.Metres(.020), Levels: 4}
}

// Coral constructs a seeded Manufactory recipe. Its geometry is an authored
// branching study, not a species identification or a biological growth model.
func Coral(spec CoralSpec) (Kind, error) {
	height, radius := spec.Height.InMetres(), spec.Radius.InMetres()
	if !positive(height) || height < .05 || height > 3 ||
		!positive(radius) || radius < .003 || radius > height*.3 ||
		spec.Levels < 1 || spec.Levels > 4 {
		return kind{}, fmt.Errorf("manufacture: coral needs height 0.05–3 m, radius 0.003 m to 30%% of height, and 1–4 branching levels")
	}
	return kind{name: "branching-coral", how: coralMaker{spec}}, nil
}

type coralMaker struct{ spec CoralSpec }

// made grows one colony of branching coral to the spec's height, radius
// and levels. It refuses a seed that was not made.
func (maker coralMaker) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	colony := good{dressed: looks{body: &thing.Material{
		Name: "branching coral", Colour: maths.RGB(.49, .32, .22),
		Texture: coralTexture{seed: from.number},
	}}}
	err := maker.grow(&colony, from, maths.V(0, 0, 0), maths.Up,
		maker.spec.Height.InMetres(), maker.spec.Radius.InMetres(), 0, 1)
	if err != nil {
		return good{}, err
	}
	return colony, nil
}

// grow keeps branch geometry bounded by the requested recursion depth.
func (maker coralMaker) grow(colony *good, from seed, start, direction maths.Vec,
	length, radius float64, level, index int,
) error {
	placement, err := spaceAlong(start, direction.Scale(length))
	if err != nil {
		return err
	}
	sample := .0012
	if maker.spec.Levels >= 3 {
		// An entire colony shares the mesh face budget. Close studies use
		// fewer generations and can spend those faces on the surface.
		sample = .0045
	}
	branch := coralBranch{length: length, radius: radius, seed: from.number + uint64(index),
		corallites: maker.spec.Corallites, fine: maker.spec.FineSurface || maker.spec.Corallites, sample: sample}
	colony.parts = append(colony.parts, part{shape: branch, space: placement, fine: cut{around: 20}, of: body})
	if level == maker.spec.Levels {
		return nil
	}
	tip := start.Add(direction.Scale(length * .88))
	turn := 2 * math.Pi * from.draw(level, index, 2)
	for child := 0; child < 3; child++ {
		angle := turn + 2*math.Pi*float64(child)/3
		fork := maths.V(math.Cos(angle), .35, math.Sin(angle))
		next := direction.Scale(.72).Add(fork.Scale(.58)).Unit()
		if err := maker.grow(colony, from, tip, next,
			length*(.65+.1*from.draw(level, index, child)), radius*.7,
			level+1, index*3+child); err != nil {
			return err
		}
	}
	return nil
}

type coralBranch struct {
	length, radius float64
	seed           uint64
	corallites     bool
	fine           bool
	sample         float64
}

// skin closes one curved, tapered branch. Intersecting branches are separate
// closed pieces, so the joining region is an approximation.
func (branch coralBranch) skin(_ cut) surface {
	rows, sides := 12, 20
	if branch.fine || branch.corallites {
		// Sample the small cups on the actual branch surface. A fixed upper
		// bound keeps densely branched colonies practical to construct.
		sample := branch.sample
		if sample == 0 {
			sample = .0012
		}
		rows = max(rows, min(108, int(math.Ceil(branch.length/sample))))
		sides = max(sides, min(108, int(math.Ceil(2*math.Pi*branch.radius/sample))))
	}
	rings := make([][]maths.Vec, rows+1)
	for row := range rings {
		t := float64(row) / float64(rows)
		r := branch.radius * (1 - .48*t)
		if row == rows {
			r *= .16
		}
		rings[row] = make([]maths.Vec, sides)
		for side := range rings[row] {
			a := 2 * math.Pi * float64(side) / float64(sides)
			rough := noise.Gradient(branch.seed, 4*math.Cos(a), 7*t, 4*math.Sin(a))
			roughness := .13
			if branch.fine || branch.corallites {
				roughness = .035
			}
			w := r * (1 + roughness*rough)
			if branch.corallites {
				w += coralliteRelief(a, t*branch.length, branch.radius) * (1 - maths.Smooth(.88, .98, t))
			}
			bend := .055 * branch.length * t * t
			rings[row][side] = maths.V(bend+w*math.Cos(a), branch.length*t, -w*math.Sin(a))
		}
	}
	return (alfaLoft{rings: rings}).skin(cut{})
}

// coralliteRelief gives a repeating, staggered field of small raised rims and
// central depressions. The 8 mm spacing and 1.2 mm relief are authored for
// inspection, not measured species-specific anatomy or a living polyp model.
// An integer number of cups wraps each ring, avoiding a seam at ±π.
func coralliteRelief(angle, height, radius float64) float64 {
	const spacing = .008
	around := max(3, int(math.Round(2*math.Pi*radius/spacing)))
	row := int(math.Round(height / spacing))
	distance := 10.0
	for y := row - 1; y <= row+1; y++ {
		shift := .5 * float64(y&1)
		x := math.Remainder(angle*float64(around)/(2*math.Pi)-shift, 1)
		dy := height/spacing - float64(y)
		distance = math.Min(distance, math.Hypot(x, dy))
	}
	rim := math.Exp(-math.Pow((distance-.31)/.16, 2))
	centre := math.Exp(-math.Pow(distance/.20, 2))
	return math.Min(.0012, radius*.10) * (rim - .7*centre)
}

// normals are the branch's vertex normals: creased, unless the branch is
// fine or has corallites, when each is the average of the faces that
// share its vertex.
func (branch coralBranch) normals(skin surface) [][3]maths.Vec {
	if !branch.fine && !branch.corallites {
		return creased(skin)
	}
	// Average the adjacent face normals at each shared vertex so the
	// sampled relief reads as a continuous surface rather than triangles.
	shared := make(map[maths.Vec]maths.Vec)
	for _, face := range skin {
		normal := face.normal()
		for _, point := range face {
			shared[point] = shared[point].Add(normal)
		}
	}
	normals := make([][3]maths.Vec, len(skin))
	for i, face := range skin {
		for corner, point := range face {
			normals[i][corner] = shared[point].Unit()
		}
	}
	return normals
}

// coralTexture suggests small corallite-scale mottling without claiming
// geometry for individual polyps. Coordinates are metres in colony space.
type coralTexture struct{ seed uint64 }

// At is the colony's base colour at a point, mottled by a large and a fine
// noise.
func (texture coralTexture) At(point maths.Vec) maths.Light {
	large := noise.Gradient(texture.seed+1, point.X*22, point.Y*22, point.Z*22)
	fine := noise.Gradient(texture.seed+2, point.X*180, point.Y*180, point.Z*180)
	return maths.RGB(.49, .32, .22).Scale(1 + .16*large + .12*fine)
}
