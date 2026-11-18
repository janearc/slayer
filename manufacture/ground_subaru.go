package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// subaruEnclosure separates the white base, equipment belt and narrower
// shutter drum. Proportions and cladding counts are authored from photographs;
// only the existing overall dimensions and aperture are sourced measurements.
func (row groundObservatory) subaruEnclosure(assembly *instrumentAssembly) {
	radius := row.diameter * spaceHalf
	base := row.height * groundValue("cylinder-base")
	shoulder := row.height * groundValue("subaru-shoulder-height")
	wall := radius * groundValue("wall-share")
	gap := groundValue("rib-joint-metres")
	drum := radius * groundValue("subaru-upper-radius")
	foot := radius * groundValue("subaru-base-radius")
	belt := radius * groundValue("subaru-belt-radius")
	opening := groundValue("opening-deg") * maths.Degree
	assembly.object.dressed[spaceFoil] = groundPaint("observatory enclosure",
		"subaru-cladding-reflectance")
	assembly.add(groundDrum(foot, foot-wall, base-gap), maths.Local(), body)
	at := func(y float64) maths.Subspace {
		return maths.NewSubspace(maths.YAxis.Scale(y), maths.Identity, spaceOne)
	}
	assembly.add(groundDrum(belt, drum-wall, shoulder-base-gap), at(base), spaceFoil)
	firstTurn := len(assembly.object.parts)
	assembly.add(slitCylinder{outer: drum, inner: drum - wall,
		height: row.height - shoulder - wall - gap, opening: opening}, at(shoulder), spaceFoil)
	assembly.add(slitCylinder{outer: drum, inner: 0,
		height: wall, opening: opening}, at(row.height-wall), spaceFoil)
	assembly.turnSince(firstTurn)
	row.subaruCladding(assembly, drum, shoulder, opening)
	row.subaruWalkway(assembly, radius, belt, base)
	// The side tower is part of the fixed service base, not the rotating drum.
	width := radius * groundValue("subaru-tower-width")
	depth := radius * groundValue("subaru-tower-depth")
	height := row.height * groundValue("subaru-tower-height")
	assembly.box(width, height, depth,
		maths.V(radius-width, 0, -depth*spaceHalf), body)
	door := groundValue("subaru-door-width")
	assembly.box(door, groundValue("subaru-door-height"), gap,
		maths.V(-door*spaceHalf, 0, -foot-gap), spaceStructure)
}

// subaruCladding adds tall raised seams and deep shutter-edge housings.
// It leaves the open sector clear rather than laying bands across the slit.
func (row groundObservatory) subaruCladding(assembly *instrumentAssembly,
	radius, base, opening float64,
) {
	firstTurn := len(assembly.object.parts)
	count := int(groundValue("subaru-panel-count"))
	width := groundValue("subaru-seam-width")
	depth := groundValue("subaru-seam-depth")
	gap := groundValue("rib-joint-metres")
	height := row.height - base - gap
	for index := range count {
		angle := fullTurn * float64(index) / float64(count)
		margin := math.Atan(width / radius)
		if nearSlit(angle, opening+2*margin) {
			continue
		}
		subaruPanel(assembly, radius, angle, base, width, height, depth, spaceFoil)
	}
	edge := groundValue("subaru-shutter-edge-width")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		angle := sign * (opening*spaceHalf + math.Atan(edge/radius))
		subaruPanel(assembly, radius, angle, base, edge, height,
			groundValue("subaru-shutter-edge-depth"), spaceFoil)
	}
	belt := row.diameter * spaceHalf * groundValue("subaru-belt-radius")
	assembly.turnSince(firstTurn)
	bottom := row.height * groundValue("cylinder-base")
	for index := range int(groundValue("subaru-vent-count")) {
		angle := fullTurn * float64(index) / groundValue("subaru-vent-count")
		subaruPanel(assembly, belt, angle, bottom+groundValue("subaru-vent-floor"),
			groundValue("subaru-vent-width"), groundValue("subaru-vent-height"),
			gap, spaceStructure)
	}
}

// subaruPanel puts a closed box tangential to the enclosure, facing outward.
func subaruPanel(assembly *instrumentAssembly, radius, angle, floor,
	width, height, depth float64, material slot,
) {
	centre := maths.V(radius*math.Sin(angle), floor, -radius*math.Cos(angle))
	pose := maths.NewSubspace(centre, maths.Rotation(maths.YAxis, -angle), spaceOne)
	assembly.add(boxShape(width, height, depth), pose, material)
}

// subaruWalkway makes the projecting deck and a continuous guard rail.
// Small gaps between capped posts and rails avoid coincident mesh edges.
func (row groundObservatory) subaruWalkway(assembly *instrumentAssembly,
	radius, belt, floor float64,
) {
	thickness := groundValue("gallery-deck-metres")
	girth := groundValue("gallery-post-metres")
	gap := groundValue("rib-joint-metres")
	floor -= thickness + gap
	rim := radius - girth
	assembly.add(groundDrum(radius, belt, thickness),
		maths.NewSubspace(maths.YAxis.Scale(floor), maths.Identity, spaceOne), spaceFoil)
	rail := floor + thickness + groundValue("gallery-rail-metres")
	assembly.add(groundDrum(rim, rim-2*girth, 2*girth),
		maths.NewSubspace(maths.YAxis.Scale(rail), maths.Identity, spaceOne), spaceFoil)
	defer assembly.detailSince(len(assembly.object.parts))
	count := int(groundValue("subaru-walkway-posts"))
	for index := range count {
		angle := fullTurn * float64(index) / float64(count)
		x, z := (rim-girth)*math.Sin(angle), -(rim-girth)*math.Cos(angle)
		assembly.rod(maths.V(x, floor+thickness+gap, z),
			maths.V(x, rail-gap, z), girth, spaceFoil)
	}
}
