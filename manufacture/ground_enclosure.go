package manufacture

import "github.com/janearc/slayer/maths"

// enclosure keeps one physical envelope across all surface details.
func (row groundObservatory) enclosure(assembly *instrumentAssembly) {
	if row.keck {
		row.keckEnclosure(assembly)
		return
	}
	if row.survey != nil {
		row.survey.facility(assembly, row)
		return
	}
	if row.cylindrical {
		row.subaruEnclosure(assembly)
		return
	}
	radius := row.diameter * spaceHalf
	wall := radius * groundValue("wall-share")
	gap := radius * groundValue("enclosure-joint-gap")
	opening := groundValue("opening-deg") * maths.Degree
	base := row.height - radius
	// Separate closed pieces have clearance instead of coincident caps.
	drum := groundDrum(radius, radius-wall, base-gap)
	assembly.add(drum, maths.Local(), body)
	defer assembly.turnSince(len(assembly.object.parts))
	at := maths.NewSubspace(maths.V(spaceZero, base, spaceZero),
		maths.Identity, spaceOne)
	dome := domeShell{radius: radius, thick: wall, opening: opening}
	assembly.add(dome, at, body)
}

// groundDrum is a closed wall with open space through its centre.
func groundDrum(outer, inner, height float64) form {
	return turnedProfile{points: [][coordinatePair]float64{
		{outer, spaceZero}, {outer, height},
		{inner, height}, {inner, spaceZero}}}
}
