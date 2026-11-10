package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// owlShelter is the open, parked roll-away hangar as an independent site prop.
// The four folded arch sections are picked scenery, not hinge kinematics.
type owlShelter struct{}

// made supplies the ordinary quick representation.
func (row owlShelter) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt retains one parked shelter and chassis at all details.
func (row owlShelter) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(groundValue("sides-" + detail))},
		object: good{dressed: groundLooks()}}
	arches := int(owlValue("shelter-arches"))
	for index := range arches {
		row.arch(assembly, index)
	}
	row.rails(assembly)
	width := owlValue("shelter-extent-width")
	low := maths.V(-width, spaceZero, -owlValue("shelter-front"))
	high := maths.V(width, owlValue("shelter-extent-height"),
		owlValue("shelter-back"))
	made, err := assembly.finish(maths.SegmentBox(low, high, spaceZero))
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *made, nil
}

// arch is a closed roof module with a chassis and wheels at both feet.
func (row owlShelter) arch(assembly *instrumentAssembly, index int) {
	z := float64(index) * owlValue("shelter-arch-step")
	at := maths.V(spaceZero, owlValue("shelter-roof-seat"), z)
	pose := maths.NewSubspace(at, maths.Identity, spaceOne)
	shell := barrelShell{radius: owlValue("shelter-radius"),
		height: owlValue("shelter-height"), wall: owlValue("shelter-wall"),
		depth: owlValue("shelter-arch-depth")}
	assembly.add(shell, pose, body)
	for _, sign := range []float64{-spaceOne, spaceOne} {
		row.bogie(assembly, sign, z)
	}
}

// bogie separates wheel caps from the arch feet and the picked rail section.
func (owlShelter) bogie(assembly *instrumentAssembly, sign, z float64) {
	width := owlValue("shelter-bogie-width")
	depth := owlValue("shelter-bogie-depth")
	x := sign * owlValue("shelter-radius")
	corner := maths.V(x-width*spaceHalf,
		owlValue("shelter-chassis-seat"), z-depth*spaceHalf)
	assembly.box(width, owlValue("shelter-chassis-height"), depth,
		corner, spaceStructure)
	orientation := maths.Rotation(maths.ZAxis, -math.Pi*spaceHalf)
	thick := owlValue("shelter-wheel-thickness")
	radius := owlValue("shelter-wheel-radius")
	for _, offset := range []float64{-spaceOne, spaceOne} {
		at := maths.V(x-thick*spaceHalf, owlValue("shelter-wheel-seat"),
			z+offset*owlValue("shelter-wheel-offset"))
		pose := maths.NewSubspace(at, orientation, spaceOne)
		assembly.add(groundDrum(radius, spaceZero, thick),
			pose, spaceStructure)
	}
}

// rails supply a short parking section; no travel animation is claimed.
func (owlShelter) rails(assembly *instrumentAssembly) {
	width := owlValue("shelter-rail-width")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		x := sign * owlValue("shelter-radius")
		corner := maths.V(x-width*spaceHalf, spaceZero,
			-owlValue("shelter-rail-front"))
		assembly.box(width, owlValue("shelter-rail-height"),
			owlValue("shelter-rail-length"), corner, spaceStructure)
	}
}
