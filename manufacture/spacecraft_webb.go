package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// triangleCorners and hexagonCorners are defined polygon vertex counts.
// quadraticFour is defined by the discriminant in the quadratic formula.
const triangleCorners, hexagonCorners, quadraticFour = 3, 6, 4

// millimetresPerMetre is defined: the paper's table of mirrors is in
// millimetres, and the research file keeps each number as its page has it.
const millimetresPerMetre = 1000

// shieldTop includes the last membrane as well as the inter-layer gaps.
func (row spacecraft) shieldTop(size spacecraftSize) float64 {
	layers := research.Value("webb-shield-layers")
	return (layers-spaceOne)*size.aperture*spacecraftValue("shield-gap") +
		size.aperture*spacecraftValue("shield-thickness")
}

// exposed is Webb's three parts as its papers lay them out (McElwain et
// al. 2023, PASP 135, 058001, section 3): the sunshield; on its sun's
// side, -y, the spacecraft bus with the solar array and the antenna; and
// on its cold side the telescope, its primary facing -z, standing on a
// tower that comes up through the shield from the bus, with the
// instrument module on the primary's back.
//
// Which side each part is on, and the secondary's place and size, are
// the papers'. Layer spacing, the mirror's figure, and the size of the
// bus, the tower, the array, the antenna and the instrument module are
// authored: the papers give none of them.
func (row spacecraft) exposed(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	row.shields(assembly, size)
	aperture := size.aperture
	stand := row.shieldTop(size) + aperture*spacecraftValue("webb-tower-rise")
	row.warmSide(assembly, size, stand)
	row.instruments(assembly, size, stand)
	centre := maths.V(spaceZero, stand+aperture*spaceHalf, -size.radius)
	orientation := maths.About(maths.XAxis, -math.Pi*spaceHalf)
	space := maths.NewSubspace(centre, orientation, spaceOne)
	panels := int(research.Value("webb-primary-segments"))
	rings := hexRings(panels)
	primary := hexagonalPrimary{rings: rings, diameter: aperture,
		fill:      spacecraftValue("tile-fill"),
		depth:     aperture * spacecraftValue("mirror-depth"),
		thickness: aperture * spacecraftValue("mirror-thickness")}
	assembly.add(primary, space, spaceMirror)
	row.secondary(assembly, size, space)
}

// warmSide hangs the bus under the shield, on the sun's side, with the
// solar array out behind it and the antenna under it, and stands the
// tower on the bus, up through the shield to where the telescope stands.
func (row spacecraft) warmSide(assembly *instrumentAssembly,
	size spacecraftSize, stand float64,
) {
	aperture := size.aperture
	height := aperture * spacecraftValue("webb-bus-height")
	top := -aperture * spacecraftValue("webb-bus-drop")
	base := top - height
	diameter := size.radius / spaceHalf
	assembly.box(diameter, height, diameter,
		maths.V(-size.radius, base, -size.radius), spaceStructure)
	assembly.rod(maths.V(spaceZero, top, spaceZero),
		maths.V(spaceZero, stand, spaceZero),
		aperture*spacecraftValue("webb-tower-radius"), spaceStructure)
	row.sunSide(assembly, size, base)
	dish := mirrorDish{
		radius: aperture * spacecraftValue("webb-antenna-radius"),
		depth:  aperture * spacecraftValue("mirror-depth")}
	under := maths.V(-size.radius*spaceHalf,
		base-aperture*spacecraftValue("webb-antenna-drop"),
		-size.radius*spaceHalf)
	assembly.add(dish, maths.NewSubspace(under,
		maths.About(maths.XAxis, math.Pi), spaceOne), spaceStructure)
}

// flat is a rectangle lying in a thing's own xz plane, about its own
// middle: so wide along x and so long along z. A plate's faces are
// drawn round its own zero, so its outline has to go round it.
func flat(wide, long, thick float64) instrumentPlate {
	x, z := wide*spaceHalf, long*spaceHalf
	return instrumentPlate{outline: []maths.Vec{
		maths.V(-x, spaceZero, -z), maths.V(x, spaceZero, -z),
		maths.V(x, spaceZero, z), maths.V(-x, spaceZero, z)},
		thickness: thick}
}

// sunSide is what hangs under the shield beside the bus, laid out by eye
// from NASA's picture of Webb's sun-facing side, and no dimension of it
// from any paper.
//
// It is two long pallets down the shield's middle, fore and aft of the
// bus; a boom out to each side corner; a row of five solar panels
// running aft from the bus, off to one side; a radiator panel on each
// side of the bus; the round interface under it; and the flap at the
// aft tip, tipped down toward the sun.
//
// The pallets and booms wear the picked white of a telescope's housing
// and not a metal: what they are made of is not in hand, and under the
// shield a mirror would show black, since the tracer follows one
// reflection and what a mirror there mirrors is another mirror.
func (row spacecraft) sunSide(assembly *instrumentAssembly,
	size spacecraftSize, base float64,
) {
	aperture := size.aperture
	thin := aperture * spacecraftValue("shield-thickness")
	under := -aperture * spacecraftValue("webb-bus-drop")
	along := size.shieldLength * spaceHalf
	// the pallets
	wide := aperture * spacecraftValue("webb-pallet-width")
	deep := aperture * spacecraftValue("webb-pallet-depth")
	long := along*spacecraftValue("webb-pallet-reach") - size.radius
	assembly.box(wide, deep, long,
		maths.V(-wide*spaceHalf, under-deep, size.radius), body)
	assembly.box(wide, deep, long,
		maths.V(-wide*spaceHalf, under-deep, -size.radius-long), body)
	// the booms to the side corners
	corner := size.span * spaceHalf * spacecraftValue("webb-boom-reach")
	boom := aperture * spacecraftValue("webb-boom-radius")
	for _, side := range []float64{-spaceOne, spaceOne} {
		assembly.rod(maths.V(side*size.radius, under-boom, spaceZero),
			maths.V(side*corner, under-boom, spaceZero), boom, body)
	}
	// the solar panels, a row running aft
	panelWide := aperture * spacecraftValue("webb-panel-width")
	panelLong := aperture * spacecraftValue("webb-panel-length")
	gap := aperture * spacecraftValue("webb-panel-gap")
	aside := wide*spaceHalf + gap
	for panel := range int(spacecraftValue("webb-panels")) {
		from := size.radius + gap + float64(panel)*(panelLong+gap)
		assembly.add(flat(panelWide, panelLong, thin), maths.NewSubspace(
			maths.V(aside+panelWide*spaceHalf, base,
				from+panelLong*spaceHalf), maths.Identity, spaceOne), leaf)
	}
	// the radiators on the bus's two sides
	height := aperture * spacecraftValue("webb-bus-height")
	diameter := size.radius / spaceHalf
	for _, side := range []float64{-size.radius - thin, size.radius} {
		assembly.box(thin, height, diameter,
			maths.V(side, base, -size.radius), spaceFoil)
	}
	// the round interface under the bus
	ring := aperture * spacecraftValue("webb-ring-radius")
	assembly.rod(maths.V(spaceZero, base, spaceZero),
		maths.V(spaceZero, base-thin/spaceHalf, spaceZero), ring,
		spaceStructure)
	// the flap at the aft tip
	flapWide := aperture * spacecraftValue("webb-flap-width")
	flapLong := aperture * spacecraftValue("webb-flap-length")
	row.seams(assembly, size, thin)
	// it hangs from its forward edge: its middle is half its length
	// further aft and that much lower as its tip allows
	tip := spacecraftValue("webb-flap-tip") * maths.Degree
	middle := maths.V(spaceZero,
		under-math.Sin(tip)*flapLong*spaceHalf,
		along*spacecraftValue("webb-flap-at")+
			math.Cos(tip)*flapLong*spaceHalf)
	assembly.add(flat(flapWide, flapLong, thin), maths.NewSubspace(middle,
		maths.About(maths.XAxis, tip), spaceOne), spaceFoil)
}

// instruments stands the instrument module on the back of the primary,
// on the cold side, where the papers put it. Neither gives its size, nor
// where each of the four instruments sits in it, so it is one closed box
// of an authored size.
func (row spacecraft) instruments(assembly *instrumentAssembly,
	size spacecraftSize, stand float64,
) {
	aperture := size.aperture
	width := aperture * spacecraftValue("webb-isim-width")
	height := aperture * spacecraftValue("webb-isim-height")
	depth := aperture * spacecraftValue("webb-isim-depth")
	behind := -size.radius + aperture*spacecraftValue("webb-isim-gap")
	assembly.box(width, height, depth,
		maths.V(-width*spaceHalf, stand, behind), spaceStructure)
}

// seams are the lines across the sun-facing membrane where its gores
// are joined: a grid of thin strips just proud of its sun side, some
// across the shield and some along it, each as long as the kite is wide
// or long where it lies, less a margin.
//
// NASA's picture shows the
// membrane seamed in a grid; how far apart the seams are, how wide, and
// that they run straight across are by eye from it. The real gores are
// not all one size and the real seams do not all run through.
func (row spacecraft) seams(assembly *instrumentAssembly,
	size spacecraftSize, proud float64,
) {
	aperture := size.aperture
	step := aperture * spacecraftValue("shield-seam-step")
	wide := aperture * spacecraftValue("shield-seam-width")
	inside := spacecraftValue("shield-seam-inside")
	across, along := size.span*spaceHalf, size.shieldLength*spaceHalf
	for out := step; out < along*inside; out += step {
		half := across * (spaceOne - out/along) * inside
		for _, side := range []float64{-spaceOne, spaceOne} {
			assembly.box(half/spaceHalf, proud, wide, maths.V(-half, -proud,
				side*out-wide*spaceHalf), spaceStructure)
		}
	}
	for out := step; out < across*inside; out += step {
		half := along * (spaceOne - out/across) * inside
		for _, side := range []float64{-spaceOne, spaceOne} {
			assembly.box(wide, proud, half/spaceHalf, maths.V(
				side*out-wide*spaceHalf, -proud, -half), spaceStructure)
		}
	}
}

// kite is the sunshield's outline: a long diamond, its tips fore and aft
// and its corners out at the sides where the booms hold it, each edge
// between two corners drawn in toward the middle as a tensioned
// membrane's is. Its length and width are the shield's published ones;
// that it is a kite with scalloped edges, and how deep the scallop, are
// by eye from NASA's picture of it.
func kite(across, along float64) []maths.Vec {
	corners := []maths.Vec{maths.V(spaceZero, spaceZero, -along),
		maths.V(across, spaceZero, spaceZero),
		maths.V(spaceZero, spaceZero, along),
		maths.V(-across, spaceZero, spaceZero)}
	steps := int(spacecraftValue("shield-edge-steps"))
	sag := spacecraftValue("shield-sag")
	var outline []maths.Vec
	for at, from := range corners {
		to := corners[(at+spaceOne)%len(corners)]
		for step := range steps {
			share := float64(step) / float64(steps)
			straight := from.Add(to.Sub(from).Scale(share))
			drawn := spaceOne - sag*math.Sin(math.Pi*share)
			outline = append(outline, straight.Scale(drawn))
		}
	}
	return outline
}

// shields retains the source layer count and authored visible separation.
// The layers are "closer together at center and further apart at the
// edges" (NASA, Webb's sunshield); by how much is authored.
func (row spacecraft) shields(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	layers := int(research.Value("webb-shield-layers"))
	for layer := range layers {
		contraction := spacecraftValue("shield-contraction")
		share := spaceOne - float64(layer)*contraction
		across := size.span * spaceHalf * share
		along := size.shieldLength * spaceHalf * share
		outline := kite(across, along)
		thickness := size.aperture * spacecraftValue("shield-thickness")
		gap := spacecraftValue("shield-gap")
		height := float64(layer) * size.aperture * gap
		// the layers fan: close together at the shield's middle and
		// further apart at its edges, so each one's middle sits lower
		// than its rim, and the sun-facing one is flat
		close := spacecraftValue("shield-gap-centre")
		plate := instrumentPlate{outline: outline, thickness: thickness,
			dish: float64(layer) * size.aperture * (gap - close)}
		at := maths.V(spaceZero, height, spaceZero)
		pose := maths.NewSubspace(at, maths.Identity, spaceOne)
		assembly.add(plate, pose, spaceFoil)
	}
}

// secondary joins three separate rim anchors to the primary support. The
// secondary stands 7169.0 mm in front of the primary's vertex and is 738
// mm across (McElwain et al. 2023, table 2), scaled with the aperture.
func (row spacecraft) secondary(assembly *instrumentAssembly,
	size spacecraftSize, primary maths.Subspace,
) {
	aperture := size.aperture
	scale := aperture / research.Value("webb-primary-diameter-metres") /
		millimetresPerMetre
	reach := scale * research.Value("webb-secondary-vertex-millimetres")
	across := scale * research.Value("webb-secondary-diameter-millimetres")
	focal := primary.Out(maths.Up.Scale(reach))
	radius := aperture * spacecraftValue("strut-radius")
	for corner := range triangleCorners {
		angle := fullTurn * float64(corner) / triangleCorners
		local := maths.V(math.Cos(angle)*aperture*spaceHalf,
			spaceZero,
			-math.Sin(angle)*aperture*spaceHalf)
		rim := local.Scale(across / aperture)
		anchor := focal.Add(primary.Orientation.Out(rim))
		start := primary.Out(local)
		assembly.rod(start, anchor, radius, spaceStructure)
	}
	reflector := mirrorDish{
		radius: across * spaceHalf,
		depth:  aperture * spacecraftValue("mirror-depth")}
	secondary := maths.NewSubspace(focal, primary.Orientation, spaceOne)
	assembly.add(reflector, secondary, spaceSecondary)
}

// hexRings solves n=3r(r+1) for complete rings around an empty centre.
func hexRings(panels int) int {
	discriminant := spaceOne + quadraticFour*float64(panels)/triangleCorners
	rings := (math.Sqrt(discriminant) - spaceOne) * spaceHalf
	return int(rings)
}
