package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

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
