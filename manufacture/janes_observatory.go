package manufacture

// The personal telescopes: a 36-inch Dobsonian, a 5-inch apochromatic
// refractor and a 24-inch Schmidt-Cassegrain, each its own catalogue
// kind.

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// personalTelescope keeps the requested apertures in three visual families.
type personalTelescope struct{ design string }

// personalGlass follows the existing instrument material roles.
const (
	personalGlass    slot = civicObjective + spaceOne
	personalEyepiece      = personalGlass + spaceOne
)

// personalValue reads authored proportions, not an optical prescription.
func personalValue(name string) float64 {
	return groundValue("personal-" + name)
}

// value reads one design's dimension or envelope share.
func (row personalTelescope) value(name string) float64 {
	return personalValue(row.design + "-" + name)
}

// personalRows gives each instrument its own ordinary catalogue kind.
func personalRows() map[string]maker {
	return map[string]maker{
		"dobsonian-36":    personalTelescope{design: "dobsonian-36"},
		"apo-refractor-5": personalTelescope{design: "apo-refractor-5"},
		"sct-24":          personalTelescope{design: "sct-24"},
	}
}

// made retains quick as the ordinary specimen.
func (row personalTelescope) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt changes circular tessellation while keeping one physical instrument.
func (row personalTelescope) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(groundValue("sides-" + detail))},
		object: good{dressed: row.looks()}}
	aperture := row.value("aperture-inches") * metresPerInch
	seat := maths.YAxis.Scale(aperture * row.value("seat"))
	altitude := personalValue("altitude-deg") * maths.Degree
	pose := tiltedFrame(seat, altitude-math.Pi*spaceHalf)
	switch row.design {
	case "dobsonian-36":
		row.dobsonian(assembly, aperture, pose)
	case "apo-refractor-5":
		row.refractor(assembly, aperture, pose)
	case "sct-24":
		row.schmidt(assembly, aperture, pose)
	}
	personalFinder(assembly, pose, aperture,
		aperture*row.value("length"))
	low := maths.V(-row.value("width"), spaceZero, -row.value("front"))
	high := maths.V(row.value("width"), row.value("height"),
		row.value("back"))
	bounds := maths.SegmentBox(low.Scale(aperture),
		high.Scale(aperture), spaceZero)
	object, err := assembly.finish(bounds)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}

// looks separates mount, structural metal, mirrors and opaque lens proxies.
func (row personalTelescope) looks() looks {
	paint := plain(row.design+" body",
		row.value("red"), row.value("green"), row.value("blue"))
	steel := groundPaint("telescope support", "structure-reflectance")
	glass := maths.RGB(personalValue("glass-red"),
		personalValue("glass-green"), personalValue("glass-blue"))
	return looks{
		body: paint, spaceStructure: steel,
		spaceMirror:      groundMirror("primary mirror"),
		spaceSecondary:   groundMirror("secondary mirror"),
		personalGlass:    personalPolished("objective proxy", glass),
		personalEyepiece: personalPolished("eyepiece proxy", glass),
	}
}

// personalPolished is an opaque scenery coating.
func personalPolished(name string, shade maths.Light) *thing.Material {
	return thing.Polished(name, shade,
		groundValue("mirror-gloss"), personalValue("glass-index"))
}

// localBox places a closed box in the instrument's tilted frame.
func localBox(assembly *instrumentAssembly, pose maths.Subspace,
	width, height, depth float64, corner maths.Vec, material slot,
) {
	assembly.boxAt(width, height, depth, corner, pose, material)
}

// objective makes an opaque, polished aperture proxy at a tube's front.
func objective(assembly *instrumentAssembly, pose maths.Subspace,
	aperture, length float64,
) {
	at := maths.NewSubspace(maths.YAxis.Scale(length),
		maths.Identity, spaceOne).Within(pose)
	thick := aperture * personalValue("objective-thickness")
	assembly.add(groundDrum(aperture*spaceHalf, spaceZero, thick),
		at, personalGlass)
}

// dobsonian carries an open Newtonian on a rocker and large altitude bearings.
func (row personalTelescope) dobsonian(assembly *instrumentAssembly,
	aperture float64, pose maths.Subspace,
) {
	base := aperture * row.value("base-height")
	board := groundDrum(aperture*row.value("base-radius"), spaceZero, base)
	assembly.add(board, maths.Local(), body)
	rocker := forkFrame{width: aperture * row.value("rocker-width"),
		height: aperture * row.value("rocker-height"),
		arm:    aperture * personalValue("mount-arm"),
		depth:  aperture * row.value("rocker-depth")}
	at := maths.NewSubspace(maths.YAxis.Scale(base),
		maths.Identity, spaceOne)
	assembly.add(rocker, at, body)
	row.altitudeBearings(assembly, aperture, pose.Origin)
	row.mirrorBox(assembly, aperture, pose)
	length := aperture * row.value("length")
	rim := aperture * personalValue("tube-radius")
	personalCage(assembly, aperture, length, pose)
	assembly.add(mirrorDish{radius: aperture * spaceHalf,
		depth: aperture * groundValue("mirror-sag")}, pose, spaceMirror)
	personalDiagonal(assembly, aperture, length, rim, pose)
	row.sideFocuser(assembly, aperture, length, pose)
}

// altitudeBearings put two circular bearing faces beside the mirror box.
func (row personalTelescope) altitudeBearings(assembly *instrumentAssembly,
	aperture float64, centre maths.Vec,
) {
	radius := aperture * row.value("bearing-radius")
	thick := aperture * personalValue("bearing-thickness")
	orientation := maths.Rotation(maths.ZAxis, -math.Pi*spaceHalf)
	for _, sign := range []float64{-spaceOne, spaceOne} {
		offset := sign * aperture * row.value("bearing-offset")
		at := centre.Add(maths.XAxis.Scale(offset))
		space := maths.NewSubspace(at, orientation, spaceOne)
		assembly.add(groundDrum(radius, spaceZero, thick), space, body)
	}
}

// mirrorBox uses separately closed panels with small authored joint clearances.
func (row personalTelescope) mirrorBox(assembly *instrumentAssembly,
	aperture float64, pose maths.Subspace,
) {
	width := aperture * row.value("box-width")
	height := aperture * row.value("box-height")
	wall := aperture * personalValue("box-wall")
	gap := aperture * personalValue("box-joint")
	half := width * spaceHalf
	corner := maths.V(-half, -wall-gap, -half)
	localBox(assembly, pose, width, wall, width, corner, body)
	for _, sign := range []float64{-spaceOne, spaceOne} {
		x := sign*half - (sign+spaceOne)*wall*spaceHalf
		localBox(assembly, pose, wall, height, width,
			maths.V(x, spaceZero, -half), body)
		z := sign*half - (sign+spaceOne)*wall*spaceHalf
		corner := maths.V(-half+wall+gap, spaceZero, z)
		span := width - (wall+gap)/spaceHalf
		localBox(assembly, pose, span, height, wall, corner, body)
	}
}

// sideFocuser suggests a Newtonian eyepiece beside the upper cage.
func (row personalTelescope) sideFocuser(assembly *instrumentAssembly,
	aperture, length float64, pose maths.Subspace,
) {
	rim := aperture * personalValue("tube-radius")
	from := pose.Out(maths.V(-rim, length, spaceZero))
	reach := aperture * personalValue("focuser-length")
	to := pose.Out(maths.V(-rim-reach,
		length, spaceZero))
	assembly.rod(from, to, aperture*personalValue("focuser-radius"),
		spaceStructure)
}

// refractor stands a long objective tube on a tripod with a counterweight.
func (row personalTelescope) refractor(assembly *instrumentAssembly,
	aperture float64, pose maths.Subspace,
) {
	row.tripod(assembly, aperture)
	length := aperture * row.value("length")
	radius := aperture * personalValue("tube-radius")
	wall := aperture * personalValue("tube-wall")
	assembly.add(groundDrum(radius, radius-wall, length), pose, body)
	objective(assembly, pose, aperture, length)
	personalTubeRings(assembly, pose, aperture)
	shield := groundDrum(aperture*personalValue("shield-radius"),
		aperture*personalValue("tube-radius"),
		aperture*personalValue("shield-length"))
	shieldStart := length + aperture*personalValue("shield-joint")
	front := maths.NewSubspace(maths.YAxis.Scale(shieldStart),
		maths.Identity, spaceOne).Within(pose)
	assembly.add(shield, front, body)
	personalFocuser(assembly, pose, aperture)
	row.counterweight(assembly, aperture)
}

// tripod has three splayed legs under a separate mount head.
func (row personalTelescope) tripod(assembly *instrumentAssembly,
	aperture float64,
) {
	top := maths.YAxis.Scale(aperture * row.value("tripod-height"))
	for index := range triangleCorners {
		angle := fullTurn * float64(index) / triangleCorners
		spread := aperture * row.value("tripod-spread")
		bottom := maths.V(spread*math.Cos(angle),
			aperture*personalValue("foot-height"),
			spread*math.Sin(angle))
		foot := groundDrum(aperture*personalValue("foot-radius"),
			spaceZero, aperture*personalValue("foot-height"))
		at := maths.V(bottom.X, spaceZero, bottom.Z)
		space := maths.NewSubspace(at, maths.Identity, spaceOne)
		assembly.add(foot, space, spaceStructure)
		offset := aperture * personalValue("leg-head-offset")
		local := maths.V(offset*math.Cos(angle),
			spaceZero, offset*math.Sin(angle))
		joint := top.Add(local)
		assembly.rod(bottom, joint,
			aperture*personalValue("leg-radius"), spaceStructure)
	}
	width := aperture * personalValue("head-width")
	height := aperture * personalValue("head-height")
	assembly.box(width, height, width,
		maths.V(-width*spaceHalf, top.Y, -width*spaceHalf), body)
}

// counterweight supplies a visible shaft and weight, not a balance calculation.
func (row personalTelescope) counterweight(assembly *instrumentAssembly,
	aperture float64,
) {
	from := maths.V(spaceZero, aperture*row.value("seat"), spaceZero)
	offset := aperture * personalValue("weight-offset")
	drop := aperture * personalValue("weight-drop")
	to := from.Add(maths.V(offset, -drop, spaceZero))
	assembly.rod(from, to, aperture*personalValue("weight-shaft"),
		spaceStructure)
	run := to.Sub(from).Unit()
	depth := aperture * personalValue("weight-depth")
	end := to.Add(run.Scale(depth))
	assembly.rod(to, end, aperture*personalValue("weight-radius"),
		spaceStructure)

}

// schmidt suggests a short folded instrument with an opaque corrector proxy.
func (row personalTelescope) schmidt(assembly *instrumentAssembly,
	aperture float64, pose maths.Subspace,
) {
	pier := aperture * row.value("pier-height")
	pedestal := groundDrum(aperture*row.value("pier-radius"),
		spaceZero, pier)
	assembly.add(pedestal, maths.Local(), body)
	fork := forkFrame{width: aperture * row.value("fork-width"),
		height: aperture * row.value("fork-height"),
		arm:    aperture * personalValue("mount-arm"),
		depth:  aperture * personalValue("fork-depth")}
	at := maths.NewSubspace(maths.YAxis.Scale(pier),
		maths.Identity, spaceOne)
	assembly.add(fork, at, spaceStructure)
	radius := aperture * personalValue("tube-radius")
	length := aperture * row.value("length")
	wall := aperture * personalValue("tube-wall")
	assembly.add(groundDrum(radius, radius-wall, length), pose, body)
	assembly.add(mirrorDish{radius: aperture * spaceHalf,
		depth: aperture * groundValue("mirror-sag")}, pose, spaceMirror)
	objective(assembly, pose, aperture, length)
	thick := aperture * personalValue("objective-thickness")
	front := maths.NewSubspace(maths.YAxis.Scale(
		length+thick),
		maths.Identity, spaceOne).Within(pose)
	secondary := groundDrum(aperture*personalValue("sct-obstruction"),
		spaceZero, aperture*personalValue("objective-thickness"))
	assembly.add(secondary, front, spaceStructure)
	inside := maths.YAxis.Scale(length - thick/spaceHalf)
	secondaryAt := tiltedFrame(inside, math.Pi).Within(pose)
	mirror := mirrorDish{
		radius: aperture * personalValue("sct-obstruction"),
		depth:  aperture * personalValue("sct-secondary-sag")}
	assembly.add(mirror, secondaryAt, spaceSecondary)
	rear := -aperture * personalValue("sct-focuser-length")
	back := pose.Out(maths.YAxis.Scale(rear))
	assembly.rod(back, pose.Origin,
		aperture*personalValue("focuser-radius"), spaceStructure)
}

// personalTubeRings close around the refractor without coincident wall faces.
func personalTubeRings(assembly *instrumentAssembly, pose maths.Subspace,
	aperture float64,
) {
	for _, name := range []string{"ring-lower", "ring-upper"} {
		height := aperture * personalValue(name)
		at := maths.NewSubspace(maths.YAxis.Scale(height),
			maths.Identity, spaceOne).Within(pose)
		ring := groundDrum(aperture*personalValue("ring-outer"),
			aperture*personalValue("ring-inner"),
			aperture*personalValue("ring-depth"))
		assembly.add(ring, at, spaceStructure)
	}
}

// personalFinder adds a small parallel tube on a short bracket.
func personalFinder(assembly *instrumentAssembly, pose maths.Subspace,
	aperture, length float64,
) {
	along := length * personalValue("finder-seat")
	offset := aperture * personalValue("finder-offset")
	start := pose.Out(maths.V(offset, along, spaceZero))
	reach := aperture * personalValue("finder-length")
	end := start.Add(pose.Orientation.Y.Scale(reach))
	assembly.rod(start, end, aperture*personalValue("finder-radius"),
		spaceStructure)
	radius := aperture * personalValue("tube-radius")
	foot := pose.Out(maths.V(radius, along, spaceZero))
	assembly.rod(foot, start, aperture*personalValue("finder-bracket"),
		spaceStructure)
}

// personalCage uses paired diagonal members rather than parallel cage posts.
func personalCage(assembly *instrumentAssembly, aperture, length float64,
	pose maths.Subspace,
) {
	lower := aperture * personalValue("cage-bottom")
	radius := aperture * personalValue("cage-radius")
	upper := aperture * personalValue("tube-radius")
	thickness := aperture * personalValue("cage-thickness")
	for _, station := range []struct{ height, radius float64 }{
		{lower, radius}, {length, upper},
	} {
		at := maths.NewSubspace(maths.YAxis.Scale(station.height),
			maths.Identity, spaceOne).Within(pose)
		assembly.add(groundDrum(station.radius,
			station.radius-thickness, thickness),
			at, spaceStructure)
	}
	pairs := int(personalValue("truss-pairs"))
	spread := personalValue("truss-spread-deg") * maths.Degree
	for index := range pairs {
		angle := fullTurn * float64(index) / float64(pairs)
		start := maths.V(radius*math.Cos(angle),
			lower+thickness*spaceHalf, -radius*math.Sin(angle))
		from := pose.Out(start)
		for _, sign := range []float64{-spaceOne, spaceOne} {
			next := angle + sign*spread
			end := maths.V(upper*math.Cos(next),
				length+thickness*spaceHalf,
				-upper*math.Sin(next))
			to := pose.Out(end)
			girth := aperture * groundValue("truss-radius")
			assembly.rod(from, to, girth,
				spaceStructure)
		}
	}
}

// personalDiagonal turns incoming light toward the Newtonian's side focuser.
// Its flat face is a visual fold, not a calibrated ray-traced prescription.
func personalDiagonal(assembly *instrumentAssembly,
	aperture, length, rim float64, pose maths.Subspace,
) {
	orientation := maths.Rotation(maths.ZAxis,
		personalValue("diagonal-tilt-deg")*maths.Degree)
	local := maths.NewSubspace(maths.YAxis.Scale(length),
		orientation, spaceOne).Within(pose)
	radius := aperture * personalValue("diagonal-radius")
	thick := aperture * personalValue("objective-thickness")
	mirror := groundDrum(radius, spaceZero, thick)
	assembly.add(mirror, local, spaceSecondary)
	groundObservatory{aperture: aperture}.secondaryRods(assembly,
		pose, rim, length, radius)
}

// personalFocuser houses the diagonal and its joints.
func personalFocuser(assembly *instrumentAssembly,
	pose maths.Subspace, aperture float64,
) {
	rear := -aperture * personalValue("rear-focuser")
	start := pose.Out(maths.YAxis.Scale(rear))
	joint := aperture * personalValue("focuser-joint")
	radius := aperture * personalValue("focuser-radius")
	tube := start.Add(pose.Orientation.Y.Scale(joint))
	assembly.rod(tube, pose.Origin, radius, spaceStructure)
	width := aperture * personalValue("diagonal-housing")
	half := width * spaceHalf
	localBox(assembly, pose, width, width, width,
		maths.V(-half, rear-half, -half), spaceStructure)
	reach := aperture * personalValue("diagonal-length")
	end := start.Add(pose.Orientation.Z.Scale(reach))
	bend := start.Add(pose.Orientation.Z.Scale(joint))
	assembly.rod(bend, end, radius, spaceStructure)
	reach = aperture * personalValue("eyepiece-length")
	eyepiece := end.Add(pose.Orientation.Y.Scale(reach))
	assembly.rod(end, eyepiece, radius, personalEyepiece)
}
