package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// enclosureDetails fits authored support members inside the existing shell.
// The service wing has its own construction; no generic dome is added to it.
func (row groundObservatory) enclosureDetails(assembly *instrumentAssembly) {
	if row.keck {
		return
	}
	defer assembly.detailSince(len(assembly.object.parts))
	if row.survey != nil {
		row.survey.facilityDetails(assembly)
		return
	}
	radius := row.diameter * spaceHalf
	if row.cylindrical {
		radius *= groundValue("subaru-upper-radius")
	}
	wall := radius * groundValue("wall-share")
	girth := row.aperture * groundValue("enclosure-rib-radius")
	seat := radius - wall - girth*groundValue("detail-clearance")
	opening := groundValue("opening-deg") * maths.Degree
	count := int(groundValue("enclosure-rib-count"))
	for index := range count {
		angle := fullTurn * float64(index) / float64(count)
		if nearSlit(angle, opening+2*row.slitMargin(seat, girth)) {
			continue
		}
		row.shellMember(assembly, seat, angle, girth)
	}
	track := row.aperture * groundValue("slit-track-radius")
	seat = radius - wall - track*groundValue("detail-clearance")
	angle := opening*spaceHalf + row.slitMargin(seat, track)*groundValue("detail-clearance")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		row.shellMember(assembly, seat, sign*angle, track)
	}
	row.gallery(assembly, radius, wall)
}

// slitMargin reserves the member's width even near the dome's narrow top.
func (row groundObservatory) slitMargin(radius, girth float64) float64 {
	if !row.cylindrical {
		radius *= math.Cos(groundValue("dome-rib-top-deg") * maths.Degree)
	}
	return math.Asin(girth / radius)
}

// shellMember follows a dome meridian, or the vertical wall of a drum.
// A chord stays inside the curved shell instead of crossing its outer skin.
func (row groundObservatory) shellMember(assembly *instrumentAssembly,
	radius, azimuth, girth float64,
) {
	if row.cylindrical {
		base := row.height * groundValue("cylinder-base")
		wall := row.diameter * spaceHalf * groundValue("wall-share")
		x, z := radius*math.Sin(azimuth), -radius*math.Cos(azimuth)
		assembly.rod(maths.V(x, base+girth, z),
			maths.V(x, row.height-wall-girth, z), girth, spaceStructure)
		return
	}
	base := row.height - row.diameter*spaceHalf
	steps := int(groundValue("dome-rib-steps"))
	stop := groundValue("dome-rib-top-deg") * maths.Degree
	point := func(step int) maths.Vec {
		latitude := stop * float64(step) / float64(steps)
		run := radius * math.Cos(latitude)
		return maths.V(run*math.Sin(azimuth), base+radius*math.Sin(latitude),
			-run*math.Cos(azimuth))
	}
	for step := range steps {
		from, to := point(step), point(step+1)
		// Keep the separately capped members from sharing an ambiguous edge.
		gap := to.Sub(from).Unit().Scale(groundValue("rib-joint-metres"))
		assembly.rod(from.Add(gap), to.Sub(gap), girth, spaceStructure)
	}
}

// nearSlit tests the angular distance from the forward-facing opening.
func nearSlit(azimuth, opening float64) bool {
	distance := math.Mod(math.Abs(azimuth)+math.Pi, fullTurn) - math.Pi
	return math.Abs(distance) < opening*spaceHalf
}

// gallery keeps the access deck at the wall and rail height in metres.
// These human-scale dimensions do not grow with the primary mirror.
func (row groundObservatory) gallery(assembly *instrumentAssembly,
	radius, wall float64,
) {
	base := row.height - radius
	if row.cylindrical {
		base = row.height * groundValue("cylinder-base")
	}
	y := base * groundValue("gallery-height")
	outer := radius - wall*groundValue("gallery-wall-clearance")
	inner := outer - radius*groundValue("gallery-width")
	thickness := groundValue("gallery-deck-metres")
	deck := groundDrum(outer, inner, thickness)
	assembly.add(deck, maths.NewSubspace(maths.YAxis.Scale(y),
		maths.Identity, spaceOne), spaceStructure)
	count := int(groundValue("gallery-post-count"))
	girth := groundValue("gallery-post-metres")
	for index := range count {
		angle := fullTurn * float64(index) / float64(count)
		for _, radius := range []float64{inner + girth, outer - girth} {
			from := maths.V(radius*math.Sin(angle), y+thickness,
				-radius*math.Cos(angle))
			assembly.rod(from, from.Add(maths.YAxis.Scale(
				groundValue("gallery-rail-metres"))), girth, spaceStructure)
		}
	}
}

// facilityDetails puts roof members and corner posts on Rubin's service wing.
func (survey surveyOptics) facilityDetails(assembly *instrumentAssembly) {
	first := len(assembly.object.parts)
	defer func() {
		// These belong to the building, so the catalogue cutaway hides them.
		for index := first; index < len(assembly.object.parts); index++ {
			assembly.object.parts[index].of = surveyCladding
		}
	}()
	girth := groundValue("survey-facility-beam-radius")
	count := int(groundValue("survey-roof-beams"))
	for _, tier := range []string{"lower", "middle", "upper"} {
		width, depth := facilityValue(tier+"-width")-facilityValue("wing-inset"), facilityValue(tier+"-depth")-2*facilityValue("wing-inset")
		start, base := facilityValue(tier+"-east"), facilityValue(tier+"-floor")
		top := base + facilityValue(tier+"-height")
		for index := range count {
			x := start + girth + (width-2*girth)*float64(index)/float64(count-1)
			assembly.rod(maths.V(x, top, -depth*spaceHalf+girth),
				maths.V(x, top, depth*spaceHalf-girth), girth, spaceStructure)
		}
		for _, x := range []float64{start + girth, start + width - girth} {
			postTop := top - girth - groundValue("rib-joint-metres")
			assembly.rod(maths.V(x, base+girth, -depth*spaceHalf+girth),
				maths.V(x, postTop, -depth*spaceHalf+girth), girth, spaceStructure)
		}
	}
}

// opticsDetails adds support behind the primary without crossing its pupil.
// It leaves the existing truss, secondary and relay geometry alone.
func (row groundObservatory) opticsDetails(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	defer assembly.detailSince(len(assembly.object.parts))
	radius := row.aperture * groundValue("truss-rim")
	outer := radius * groundValue("cell-outer-share")
	inner := outer * groundValue("cell-inner-share")
	width := row.aperture * groundValue("cell-beam-radius")
	height := -row.aperture*groundValue("mirror-thickness") - 2*width
	for _, radius := range []float64{inner, outer} {
		ring := groundDrum(radius, radius-width, width)
		at := maths.NewSubspace(maths.YAxis.Scale(height),
			maths.Identity, spaceOne).Within(pose)
		assembly.add(ring, at, spaceStructure)
	}
	count := int(groundValue("mirror-cell-spokes"))
	for index := range count {
		angle := fullTurn * float64(index) / float64(count)
		from := maths.V(inner*math.Sin(angle), height, -inner*math.Cos(angle))
		to := maths.V(outer*math.Sin(angle), height, -outer*math.Cos(angle))
		if row.survey != nil {
			// Separate the spoke caps from the closed ring edges after placement.
			gap := to.Sub(from).Unit().Scale(groundValue("rib-joint-metres"))
			from, to = from.Add(gap), to.Sub(gap)
		}
		assembly.rod(pose.Out(from), pose.Out(to), width, spaceStructure)
	}
}

// detailSince caps small member sections independently of the main mirrors.
// Refining a large primary must not spend its full circular cut on every post.
func (assembly *instrumentAssembly) detailSince(first int) {
	limit := int(groundValue("detail-member-sides"))
	for index := first; index < len(assembly.object.parts); index++ {
		if assembly.object.parts[index].fine.around > limit {
			assembly.object.parts[index].fine.around = limit
		}
	}
}
