package manufacture

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// prima builds a dark layered canopy, a faceted equipment hull, side
// radiators, and a trussed open telescope. Only its 1.8 m primary is sourced;
// these exterior dimensions are picked from the NASA/JPL concept image.
func (row spacecraft) prima(assembly *instrumentAssembly, size spacecraftSize) {
	row.primaHull(assembly, size)
	row.primaCanopy(assembly, size)
	row.primaRadiators(assembly, size)
	row.primaOptics(assembly, size)
	row.primaAntenna(assembly, size)
	row.primaEquipment(assembly, size)
}

// primaFrameRod keeps long structural members smooth without spending the
// mirror and canopy tessellation budget on their round cross-sections.
func (row spacecraft) primaFrameRod(assembly *instrumentAssembly,
	from, to maths.Vec, radius float64, material slot,
) {
	fine := assembly.fine
	assembly.fine.around = int(primaValue("structure-sides"))
	assembly.frameRod(from, to, radius, material)
	assembly.fine = fine
}

type primaHullSection struct {
	z, halfWidth, top, bottom, bevel float64
}

// primaHull is a closed bevelled equipment body swept through five changed
// cross-sections. It gives the spacecraft its wedge, not a display box.
type primaHull []primaHullSection

// skin is the hull's surface: its stations joined ring to ring, and the
// two ends capped.
func (hull primaHull) skin(cut) surface {
	if len(hull) < 2 {
		return nil
	}
	var skin surface
	for station := 0; station < len(hull)-1; station++ {
		back, front := hull.ring(station), hull.ring(station+1)
		for corner := range len(back) {
			next := (corner + spaceOne) % len(back)
			skin = quad(skin, back[corner], back[next], front[next], front[corner])
		}
	}
	skin = hull.cap(skin, 0, false)
	return hull.cap(skin, len(hull)-1, true)
}

// ring is the eight corners of the section at a station: a rectangle with
// its corners bevelled.
func (hull primaHull) ring(station int) []maths.Vec {
	section := hull[station]
	w, b, top, bottom, z := section.halfWidth, section.bevel,
		section.top, section.bottom, section.z
	return []maths.Vec{
		maths.V(-w+b, bottom, z), maths.V(w-b, bottom, z),
		maths.V(w, bottom+b, z), maths.V(w, top-b, z),
		maths.V(w-b, top, z), maths.V(-w+b, top, z),
		maths.V(-w, top-b, z), maths.V(-w, bottom+b, z),
	}
}

// cap closes the hull at a station with a fan of faces from the middle of
// the section; forward turns them to face forward.
func (hull primaHull) cap(skin surface, station int, forward bool) surface {
	ring := hull.ring(station)
	section := hull[station]
	centre := maths.V(spaceZero, (section.top+section.bottom)*spaceHalf,
		section.z)
	for corner := range len(ring) {
		next := (corner + spaceOne) % len(ring)
		if forward {
			skin = append(skin, face{centre, ring[corner], ring[next]})
		} else {
			skin = append(skin, face{centre, ring[next], ring[corner]})
		}
	}
	return skin
}

// primaHull adds the equipment hull, its sections scaled by the craft's
// aperture.
func (row spacecraft) primaHull(assembly *instrumentAssembly, size spacecraftSize) {
	a := size.aperture
	sections := primaHull{
		{z: primaValue("hull-back-z") * a, halfWidth: primaValue("hull-back-width") * a,
			top: primaValue("hull-back-top") * a, bottom: primaValue("hull-back-bottom") * a,
			bevel: primaValue("hull-back-bevel") * a},
		{z: primaValue("hull-aft-z") * a, halfWidth: primaValue("hull-aft-width") * a,
			top: primaValue("hull-aft-top") * a, bottom: primaValue("hull-aft-bottom") * a,
			bevel: primaValue("hull-aft-bevel") * a},
		{z: primaValue("hull-mid-z") * a, halfWidth: primaValue("hull-mid-width") * a,
			top: primaValue("hull-mid-top") * a, bottom: primaValue("hull-mid-bottom") * a,
			bevel: primaValue("hull-mid-bevel") * a},
		{z: primaValue("hull-fore-z") * a, halfWidth: primaValue("hull-fore-width") * a,
			top: primaValue("hull-fore-top") * a, bottom: primaValue("hull-fore-bottom") * a,
			bevel: primaValue("hull-fore-bevel") * a},
		{z: primaValue("hull-nose-z") * a, halfWidth: primaValue("hull-nose-width") * a,
			top: primaValue("hull-nose-top") * a, bottom: primaValue("hull-nose-bottom") * a,
			bevel: primaValue("hull-nose-bevel") * a},
	}
	assembly.add(sections, maths.Local(), body)

	// A narrow raised spine and sloped armour cheek distinguish the top deck.
	spineWidth := primaValue("spine-width") * a
	spineHeight := primaValue("spine-height") * a
	spineLength := primaValue("spine-length") * a
	assembly.box(spineWidth, spineHeight, spineLength,
		maths.V(-spineWidth*spaceHalf, 0.46*a, -1.48*a), spaceFoil)
	for _, side := range []float64{-spaceOne, spaceOne} {
		row.primaFrameRod(assembly, maths.V(side*0.42*a, 0.45*a, -1.7*a),
			maths.V(side*0.96*a, 0.28*a, 1.1*a), 0.045*a, spaceStructure)
	}
}

// primaCanopy lays two shaped thermal panels on each of four stepped layers.
// Edge rails and cross ribs make the foil a framed, deployable assembly.
func (row spacecraft) primaCanopy(assembly *instrumentAssembly, size spacecraftSize) {
	a := size.aperture
	layers := int(primaValue("shield-layers"))
	frontZ, aftZ := primaValue("canopy-front-z")*a,
		primaValue("canopy-aft-z")*a
	frontOuter, aftOuter := primaValue("canopy-front-outer")*a,
		primaValue("canopy-aft-outer")*a
	frontInner, aftInner := primaValue("canopy-front-inner")*a,
		primaValue("canopy-aft-inner")*a
	thickness := primaValue("canopy-thickness") * a
	gap := primaValue("canopy-layer-gap") * a
	upper := primaValue("canopy-upper") * a
	for layer := range layers {
		shrink := spaceOne - float64(layer)*primaValue("canopy-layer-shrink")
		y := upper - float64(layer)*gap
		for _, side := range []float64{-spaceOne, spaceOne} {
			outline := primaCanopyOutline(side, frontZ, aftZ,
				frontOuter*shrink, aftOuter*shrink,
				frontInner*shrink, aftInner*shrink)
			plate := instrumentPlate{outline: outline, thickness: thickness,
				dish: primaValue("canopy-dish") * a}
			at := maths.NewSubspace(maths.V(spaceZero, y, spaceZero),
				maths.Identity, spaceOne)
			assembly.add(plate, at, spaceFoil)
			row.primaCanopyFrame(assembly, outline, y+thickness,
				primaValue("canopy-rib-radius")*a, layer == 0,
				frontZ, aftZ, a)
		}
	}
	// Four short stanchions fix the lowest canopy layer to hull hardpoints.
	for _, side := range []float64{-spaceOne, spaceOne} {
		for _, z := range []float64{-1.75 * a, 0.95 * a} {
			row.primaFrameRod(assembly, maths.V(side*0.95*a, 0.30*a, z),
				maths.V(side*1.18*a, upper-float64(layers-1)*gap,
					z-0.10*a), primaValue("canopy-stanchion-radius")*a,
				spaceStructure)
		}
	}
}

// primaCanopyOutline is the plan of one layer of the canopy on a side, as
// points from the front to the aft, between an outer and an inner width.
func primaCanopyOutline(side, frontZ, aftZ, frontOuter, aftOuter,
	frontInner, aftInner float64,
) []maths.Vec {
	outline := []maths.Vec{
		maths.V(side*frontOuter, spaceZero, frontZ),
		maths.V(side*frontInner, spaceZero, frontZ),
		maths.V(side*aftInner, spaceZero, aftZ),
		maths.V(side*aftOuter, spaceZero, aftZ),
	}
	if side < spaceZero {
		for left, right := 0, len(outline)-1; left < right; left, right = left+1, right-1 {
			outline[left], outline[right] = outline[right], outline[left]
		}
	}
	return outline
}

// primaCanopyFrame adds rods along a canopy outline at a height, and ribs
// across it if addRibs is set.
func (row spacecraft) primaCanopyFrame(assembly *instrumentAssembly,
	outline []maths.Vec, y, radius float64, addRibs bool,
	frontZ, aftZ, aperture float64,
) {
	for i, here := range outline {
		next := outline[(i+spaceOne)%len(outline)]
		row.primaFrameRod(assembly, maths.V(here.X, y, here.Z),
			maths.V(next.X, y, next.Z), radius, spaceStructure)
	}
	if !addRibs {
		return
	}
	for _, share := range []float64{
		(primaValue("canopy-rib-one-z")*aperture - aftZ) / (frontZ - aftZ),
		(primaValue("canopy-rib-two-z")*aperture - aftZ) / (frontZ - aftZ),
		(primaValue("canopy-rib-three-z")*aperture - aftZ) / (frontZ - aftZ),
	} {
		z := aftZ + share*(frontZ-aftZ)
		outer := outline[3].X + (outline[0].X-outline[3].X)*share
		inner := outline[2].X + (outline[1].X-outline[2].X)*share
		from := maths.V(inner, y, z)
		to := maths.V(outer, y, z)
		row.primaFrameRod(assembly, from, to, radius*spaceHalf, spacePanel)
	}
}

// primaRadiators adds the radiators on the two sides of the hull, their
// thickness, height and length scaled by the aperture.
func (row spacecraft) primaRadiators(assembly *instrumentAssembly, size spacecraftSize) {
	a := size.aperture
	thick := primaValue("radiator-thickness") * a
	height := primaValue("radiator-height") * a
	length := primaValue("radiator-length") * a
	centreX := primaValue("radiator-centre-x") * a
	centreY := primaValue("radiator-centre-y") * a
	centreZ := primaValue("radiator-centre-z") * a
	for _, side := range []float64{-spaceOne, spaceOne} {
		x := side*centreX - thick*spaceHalf
		assembly.box(thick, height, length,
			maths.V(x, centreY-height*spaceHalf, centreZ-length*spaceHalf),
			spacePanel)
		// Ribs divide each silver radiator into ten readable cells.
		for row := range 5 {
			y := centreY - height*spaceHalf +
				float64(row+1)*height/6
			assembly.box(thick*1.20, 0.032*a, length,
				maths.V(x-thick*0.10, y, centreZ-length*spaceHalf),
				spaceStructure)
		}
		for column := range 4 {
			z := centreZ - length*spaceHalf +
				float64(column+1)*length/5
			assembly.box(thick*1.20, height, 0.034*a,
				maths.V(x-thick*0.10, centreY-height*spaceHalf, z),
				spaceFoil)
		}
		// A shallow equipment saddle caps the inboard lower edge.
		assembly.box(thick*1.6, 0.24*a, 0.68*a,
			maths.V(x-thick*0.25, centreY-height*spaceHalf,
				centreZ-0.15*a), body)
	}
}

// primaOptics adds the open telescope: the primary mirror with its frames
// and the truss that carries them, scaled by the aperture.
func (row spacecraft) primaOptics(assembly *instrumentAssembly, size spacecraftSize) {
	a := size.aperture
	axis := maths.About(maths.XAxis, math.Pi*spaceHalf)
	centreY := primaValue("optical-height") * a
	startZ := primaValue("optical-start-z") * a
	length := size.length
	outer, wall := size.radius, primaValue("optical-wall")*a
	profile := []ComponentProfilePoint{
		{Radius: measure.Metres(outer), Height: measure.Metres(spaceZero)},
		{Radius: measure.Metres(outer), Height: measure.Metres(length)},
		{Radius: measure.Metres(outer - wall), Height: measure.Metres(length)},
		{Radius: measure.Metres(outer - wall), Height: measure.Metres(spaceZero)},
	}
	tubePose := maths.NewSubspace(maths.V(spaceZero, centreY, startZ), axis, spaceOne)
	ComponentSpec{Form: ComponentDrum, Frame: tubePose,
		Radius: measure.Metres(outer), Height: measure.Metres(length),
		Smooth: true, Profile: profile}.appendTo(assembly, body)

	// The instrument tube has a deep black throat, stepped inner baffles and a
	// bright reinforced rim. These stations are modelled as annular sleeves.
	for _, share := range []float64{0.16, 0.37, 0.58, 0.79, 0.97} {
		z := startZ + length*share
		row.primaAnnulus(assembly, maths.V(spaceZero, centreY, z),
			outer-wall*1.65, outer-wall*spaceHalf, 0.035*a,
			spaceFoil)
	}
	for _, share := range []float64{0.03, 0.45, 0.94} {
		z := startZ + length*share
		row.primaAnnulus(assembly, maths.V(spaceZero, centreY, z),
			outer, outer+0.055*a, 0.08*a, spaceStructure)
	}

	// Primary and secondary are separate shallow reflectors in the optical
	// train. Their scale follows the primary aperture; no optical prescription
	// is claimed.
	primaryZ := startZ + primaValue("primary-station-z")*a
	assembly.add(mirrorDish{radius: a * spaceHalf,
		depth: primaValue("primary-sag") * a},
		maths.NewSubspace(maths.V(spaceZero, centreY, primaryZ), axis, spaceOne),
		spaceMirror)
	secondaryZ := startZ + length*primaValue("secondary-station-share")
	secondaryRadius := primaValue("secondary-radius") * a
	secondaryAxis := maths.About(maths.XAxis, -math.Pi*spaceHalf)
	assembly.add(mirrorDish{radius: secondaryRadius,
		depth: primaValue("secondary-sag") * a},
		maths.NewSubspace(maths.V(spaceZero, centreY, secondaryZ),
			secondaryAxis, spaceOne), spaceSecondary)

	// Heavy four-sided frame holds the baffle and mirror spider. The first and
	// last rectangular rings are braced into a truss, not free antenna rods.
	backZ, frontZ := startZ+0.14*a, startZ+length*0.96
	frameX := primaValue("truss-frame-x") * a
	frameY := primaValue("truss-frame-y") * a
	girth := primaValue("truss-frame-radius") * a
	for _, z := range []float64{backZ, frontZ} {
		row.primaRectFrame(assembly, frameX, centreY, frameY, z, girth)
	}
	for _, side := range []float64{-spaceOne, spaceOne} {
		for _, upper := range []float64{-spaceOne, spaceOne} {
			back := maths.V(side*frameX, centreY+upper*frameY, backZ)
			front := maths.V(side*frameX, centreY+upper*frameY, frontZ)
			row.primaFrameRod(assembly, back, front, girth, spaceStructure)
			other := maths.V(-side*frameX, centreY+upper*frameY, frontZ)
			row.primaFrameRod(assembly, back, other, girth*0.78, spaceStructure)
		}
	}

	// Three spider vanes hold the small secondary behind the aperture rim.
	for vane := range triangleCorners {
		angle := fullTurn * float64(vane) / triangleCorners
		from := maths.V(frameX*0.88*math.Cos(angle),
			centreY+frameY*0.88*math.Sin(angle), frontZ)
		to := maths.V(secondaryRadius*0.72*math.Cos(angle),
			centreY+secondaryRadius*0.72*math.Sin(angle), secondaryZ)
		row.primaFrameRod(assembly, from, to,
			primaValue("spider-radius")*a, spaceStructure)
	}
}

// primaAnnulus adds a flat ring at a point, between an inner and an outer
// radius and a width thick, in a material.
func (row spacecraft) primaAnnulus(assembly *instrumentAssembly, at maths.Vec,
	inner, outer, width float64, material slot,
) {
	axis := maths.About(maths.XAxis, math.Pi*spaceHalf)
	pose := maths.NewSubspace(at, axis, spaceOne)
	profile := []ComponentProfilePoint{
		{Radius: measure.Metres(outer), Height: measure.Metres(spaceZero)},
		{Radius: measure.Metres(outer), Height: measure.Metres(width)},
		{Radius: measure.Metres(inner), Height: measure.Metres(width)},
		{Radius: measure.Metres(inner), Height: measure.Metres(spaceZero)},
	}
	ComponentSpec{Form: ComponentDrum, Frame: pose,
		Radius: measure.Metres(outer), Height: measure.Metres(width),
		Smooth: true, Profile: profile}.appendTo(assembly, material)
}

// primaRectFrame adds a rectangle of rods at depth z: a half width and a
// half height about the height centreY.
func (row spacecraft) primaRectFrame(assembly *instrumentAssembly,
	halfWidth, centreY, halfHeight, z, radius float64,
) {
	corners := []maths.Vec{
		maths.V(-halfWidth, centreY-halfHeight, z),
		maths.V(halfWidth, centreY-halfHeight, z),
		maths.V(halfWidth, centreY+halfHeight, z),
		maths.V(-halfWidth, centreY+halfHeight, z),
	}
	for index, here := range corners {
		row.primaFrameRod(assembly, here, corners[(index+spaceOne)%len(corners)],
			radius, spaceStructure)
	}
}

// primaAntenna adds the antenna at its place, scaled by the aperture.
func (row spacecraft) primaAntenna(assembly *instrumentAssembly, size spacecraftSize) {
	a := size.aperture
	base := maths.V(primaValue("antenna-x")*a, primaValue("antenna-y")*a,
		primaValue("antenna-z")*a)
	radius := primaValue("antenna-radius") * a
	assembly.add(mirrorDish{radius: radius, depth: primaValue("antenna-depth") * a},
		maths.NewSubspace(base, maths.Identity, spaceOne), spacePanel)
	feed := base.Add(maths.V(spaceZero,
		primaValue("antenna-feed-height")*a, spaceZero))
	for arm := range triangleCorners {
		angle := fullTurn * float64(arm) / triangleCorners
		rim := base.Add(maths.V(radius*0.72*math.Cos(angle),
			0.025*a, radius*0.72*math.Sin(angle)))
		row.primaFrameRod(assembly, rim, feed,
			primaValue("antenna-feed-radius")*a, spaceStructure)
	}
	feedWidth := primaValue("antenna-feed-width") * a
	assembly.box(feedWidth, 0.09*a, feedWidth,
		maths.V(feed.X-feedWidth*spaceHalf, feed.Y, feed.Z-feedWidth*spaceHalf),
		spaceGold)
}

// primaEquipment adds the ports along each side of the hull and the other
// fittings on it, scaled by the aperture.
func (row spacecraft) primaEquipment(assembly *instrumentAssembly, size spacecraftSize) {
	a := size.aperture
	for _, side := range []float64{-spaceOne, spaceOne} {
		for port := range int(primaValue("port-count")) {
			z := (primaValue("port-start-z") +
				float64(port)*primaValue("port-z-step")) * a
			x := side * (1.49*a + 0.045*a)
			if side < spaceZero {
				x -= primaValue("port-width") * a
			}
			y := (primaValue("port-height-y") +
				float64(port%2)*primaValue("port-row-gap")) * a
			assembly.box(primaValue("port-width")*a,
				primaValue("port-height")*a, primaValue("port-depth")*a,
				maths.V(x, y, z), spaceGold)
		}
		// Dark covers and two polished access doors sit between the ports and
		// the radiator frames as distinct equipment modules.
		moduleX := side * 1.56 * a
		if side < spaceZero {
			moduleX -= 0.11 * a
		}
		assembly.box(0.11*a, 0.33*a, 0.62*a,
			maths.V(moduleX, -0.42*a, 0.70*a), body)
		assembly.box(0.04*a, 0.24*a, 0.42*a,
			maths.V(moduleX+side*0.11*a, -0.38*a, 0.78*a), spacePanel)
	}
}
