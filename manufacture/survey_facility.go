package manufacture

import "github.com/janearc/slayer/maths"

// surveyCladding is a separate exterior surface, hidden with the enclosure.
const surveyCladding slot = surveyCamera + spaceOne

// facilityValue reads picked scenery dimensions, not a surveyed site plan.
func facilityValue(name string) float64 {
	return groundValue("survey-facility-" + name)
}

// facility gives Rubin its enclosure base and long, stepped service wing.
// Functional cues come from the pinned Rubin summit-facility description.
func (survey surveyOptics) facility(assembly *instrumentAssembly,
	row groundObservatory,
) {
	assembly.object.dressed[surveyCladding] = groundPaint(
		"observatory enclosure", "survey-facility-cladding-reflectance")
	radius := row.diameter * spaceHalf
	base := facilityValue("base-height")
	wall := facilityValue("wall")
	assembly.add(groundDrum(radius, radius-wall, base), maths.Local(), body)
	survey.upperEnclosure(assembly, radius, wall, base, row.height)
	survey.serviceWing(assembly)
}

// upperEnclosure keeps the observing slit through the faceted walls and roof.
func (survey surveyOptics) upperEnclosure(assembly *instrumentAssembly,
	radius, wall, base, top float64,
) {
	gap := facilityValue("joint-gap")
	roof := facilityValue("roof-height")
	opening := groundValue("opening-deg") * maths.Degree
	sides := int(facilityValue("facets"))
	shell := facetedShell{base: radius, top: radius, wall: wall,
		height: top - base - roof - gap, opening: opening, sides: sides}
	at := maths.NewSubspace(maths.YAxis.Scale(base+gap),
		maths.Identity, spaceOne)
	assembly.add(shell, at, surveyCladding)
	shell.top = radius * facilityValue("roof-top-share")
	shell.height = roof - wall - gap/spaceHalf
	at.Origin.Y = top - roof + gap
	assembly.add(shell, at, surveyCladding)
	cap := slitCylinder{outer: shell.top, inner: spaceZero,
		height: wall, opening: opening}
	at.Origin.Y = top - wall
	assembly.object.parts = append(assembly.object.parts,
		part{shape: cap, space: at, of: surveyCladding,
			fine: cut{around: sides}})
}

// serviceWing separates the tiered maintenance building from the enclosure.
func (survey surveyOptics) serviceWing(assembly *instrumentAssembly) {
	for _, tier := range []string{"lower", "middle", "upper"} {
		width := facilityValue(tier + "-width")
		depth := facilityValue(tier + "-depth")
		x, y := facilityValue(tier+"-east"), facilityValue(tier+"-floor")
		corner := maths.V(x, y, -depth*spaceHalf)
		height := facilityValue(tier + "-height")
		assembly.box(width, height, depth, corner, body)
		corner.Y += height + facilityValue("joint-gap")
		assembly.box(width, facilityValue("roof-sheet"), depth,
			corner, surveyCladding)
	}
	survey.loadingDoors(assembly)
}

// loadingDoors make the long wing read as a service facility.
func (survey surveyOptics) loadingDoors(assembly *instrumentAssembly) {
	for _, entry := range []string{"loading", "personnel"} {
		corner := maths.V(facilityValue(entry+"-east"),
			facilityValue(entry+"-floor"), facilityValue(entry+"-south"))
		assembly.box(facilityValue(entry+"-width"),
			facilityValue(entry+"-height"), facilityValue("door-depth"),
			corner, surveyCladding)
	}
}

// facilityBounds reserves room for the service wing as well as the tower.
func (survey surveyOptics) facilityBounds(row groundObservatory) maths.Box {
	radius := row.diameter * spaceHalf
	low := maths.V(-radius, spaceZero, -radius)
	high := maths.V(facilityValue("east-bound"), row.height, radius)
	return maths.SegmentBox(low, high, spaceZero)
}
