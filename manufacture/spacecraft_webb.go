package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// triangleCorners and hexagonCorners are defined polygon vertex counts.
// quadraticFour is defined by the discriminant in the quadratic formula.
const triangleCorners, hexagonCorners, quadraticFour = 3, 6, 4

// shieldTop includes the last membrane as well as the inter-layer gaps.
func (row spacecraft) shieldTop(size spacecraftSize) float64 {
	layers := research.Value("webb-shield-layers")
	return (layers-spaceOne)*size.aperture*spacecraftValue("shield-gap") +
		size.aperture*spacecraftValue("shield-thickness")
}

// exposed keeps the primary above a layered shield, with its face toward -z.
// Layer spacing, supporting dimensions and mirror figure remain authored.
func (row spacecraft) exposed(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	row.shields(assembly, size)
	aperture := size.aperture
	busY := row.shieldTop(size) + aperture*spacecraftValue("webb-bus-lift")
	busHeight := aperture * spacecraftValue("webb-bus-height")
	diameter := size.radius / spaceHalf
	bus := maths.V(-size.radius*spaceOne, busY, -size.radius*spaceOne)
	assembly.box(diameter, busHeight, diameter, bus, spaceStructure)
	centre := maths.V(spaceZero, busY+busHeight+aperture*spaceHalf,
		-size.radius)
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

// shields retains the source layer count and authored visible separation.
func (row spacecraft) shields(assembly *instrumentAssembly,
	size spacecraftSize,
) {
	layers := int(research.Value("webb-shield-layers"))
	for layer := range layers {
		contraction := spacecraftValue("shield-contraction")
		share := spaceOne - float64(layer)*contraction
		across := size.span * spaceHalf * share
		along := size.shieldLength * spaceHalf * share
		shoulder := size.shieldLength *
			spacecraftValue("shield-shoulder") * share
		outline := []maths.Vec{maths.V(spaceZero, spaceZero, -along),
			maths.V(across, spaceZero, -shoulder),
			maths.V(across, spaceZero, shoulder),
			maths.V(spaceZero, spaceZero, along),
			maths.V(-across, spaceZero, shoulder),
			maths.V(-across, spaceZero, -shoulder)}
		thickness := size.aperture * spacecraftValue("shield-thickness")
		plate := instrumentPlate{outline: outline, thickness: thickness}
		gap := spacecraftValue("shield-gap")
		height := float64(layer) * size.aperture * gap
		at := maths.V(spaceZero, height, spaceZero)
		pose := maths.NewSubspace(at, maths.Identity, spaceOne)
		assembly.add(plate, pose, spaceFoil)
	}
}

// secondary joins three separate rim anchors to the primary support.
func (row spacecraft) secondary(assembly *instrumentAssembly,
	size spacecraftSize, primary maths.Subspace,
) {
	aperture := size.aperture
	reach := aperture * spacecraftValue("secondary-reach")
	focal := primary.Out(maths.Up.Scale(reach))
	radius := aperture * spacecraftValue("strut-radius")
	for corner := range triangleCorners {
		angle := fullTurn * float64(corner) / triangleCorners
		local := maths.V(math.Cos(angle)*aperture*spaceHalf,
			spaceZero,
			-math.Sin(angle)*aperture*spaceHalf)
		rimShare := spacecraftValue("secondary-radius") / spaceHalf
		rim := local.Scale(rimShare)
		anchor := focal.Add(primary.Orientation.Out(rim))
		start := primary.Out(local)
		assembly.rod(start, anchor, radius, spaceStructure)
	}
	reflector := mirrorDish{
		radius: aperture * spacecraftValue("secondary-radius"),
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
