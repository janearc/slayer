package manufacture

import "github.com/janearc/slayer/maths"

// enclosure keeps one physical envelope across all surface details.
func (row groundObservatory) enclosure(assembly *instrumentAssembly) {
	if row.survey != nil {
		row.survey.facility(assembly, row)
		return
	}
	radius := row.diameter * spaceHalf
	wall := radius * groundValue("wall-share")
	gap := radius * groundValue("enclosure-joint-gap")
	opening := groundValue("opening-deg") * maths.Degree
	base := row.height - radius
	if row.cylindrical {
		base = row.height * groundValue("cylinder-base")
	}
	// Separate closed pieces have clearance instead of coincident caps.
	drum := groundDrum(radius, radius-wall, base-gap)
	assembly.add(drum, maths.Local(), body)
	at := maths.NewSubspace(maths.V(spaceZero, base, spaceZero),
		maths.Identity, spaceOne)
	if !row.cylindrical {
		dome := domeShell{radius: radius, thick: wall, opening: opening}
		assembly.add(dome, at, body)
		return
	}
	upper := slitCylinder{outer: radius, inner: radius - wall,
		height: row.height - base - wall - gap, opening: opening}
	assembly.add(upper, at, body)
	roofAt := maths.V(spaceZero, row.height-wall, spaceZero)
	roof := maths.NewSubspace(roofAt, maths.Identity, spaceOne)
	assembly.add(slitCylinder{outer: radius, inner: spaceZero,
		height: wall, opening: opening}, roof, body)
}

// groundDrum is a closed wall with open space through its centre.
func groundDrum(outer, inner, height float64) form {
	return turnedProfile{points: [][coordinatePair]float64{
		{outer, spaceZero}, {outer, height},
		{inner, height}, {inner, spaceZero}}}
}
