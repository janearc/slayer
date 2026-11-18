package manufacture

import "github.com/janearc/slayer/maths"

// backplane gives Webb's rear a braced frame around the retained instrument
// module. NASA's backplane description establishes its location and role;
// this frame's members and proportions are authored visual construction.
func (row spacecraft) backplane(a *instrumentAssembly, size spacecraftSize, stand float64) {
	aperture := size.aperture
	half := aperture * spacecraftValue("backplane-half-width")
	height := aperture * spacecraftValue("backplane-height")
	front := -size.radius + aperture*spacecraftValue("backplane-gap")
	back := front + aperture*spacecraftValue("backplane-depth")
	thick := aperture * spacecraftValue("backplane-member-radius")
	fine := a.fine
	a.fine.around = octagonSides
	defer func() { a.fine = fine }()
	levels := int(spacecraftValue("backplane-levels"))
	for level := range levels {
		low := stand + height*float64(level)/float64(levels)
		high := stand + height*float64(level+spaceOne)/float64(levels)
		for _, depth := range []float64{front, back} {
			a.frameRod(maths.V(-half, low, depth), maths.V(half, low, depth), thick, spaceStructure)
			for _, side := range []float64{-spaceOne, spaceOne} {
				a.frameRod(maths.V(side*half, low, depth), maths.V(side*half, high, depth), thick, spaceStructure)
			}
		}
		for _, side := range []float64{-spaceOne, spaceOne} {
			a.frameRod(maths.V(side*half, low, front), maths.V(side*half, high, back), thick, body)
			a.frameRod(maths.V(side*half, low, back), maths.V(side*half, high, front), thick, body)
		}
	}
	for _, depth := range []float64{front, back} {
		a.frameRod(maths.V(-half, stand+height, depth), maths.V(half, stand+height, depth), thick, spaceStructure)
	}
	// A broad radiator behind the module gives the back a legible surface
	// against the slender frame. Its finish and dimensions are picked.
	wide := aperture * spacecraftValue("backplane-radiator-width")
	tall := aperture * spacecraftValue("backplane-radiator-height")
	a.box(wide, tall, thick, maths.V(-wide*spaceHalf,
		stand+(height-tall)*spaceHalf, back), body)
}

// frameRod leaves a tiny seam at a braced joint rather than duplicating
// coincident end-cap edges in the merged material mesh.
func (a *instrumentAssembly) frameRod(from, to maths.Vec, radius float64, material slot) {
	inset := to.Sub(from).Unit().Scale(radius * spacecraftValue("frame-joint-inset"))
	a.rod(from.Add(inset), to.Sub(inset), radius, material)
}
