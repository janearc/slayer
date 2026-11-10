package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// owlObservatory is a scenery interpretation of the 2005 OWL concept.
// Its open structure and segmented secondary distinguish it from the ELT.
type owlObservatory struct{}

// owlValue reads authored structure; source optics stay in the research book.
func owlValue(name string) float64 { return groundValue("owl-" + name) }

// owlRows keeps the open telescope and movable shelter independently placeable.
func owlRows() map[string]maker {
	return map[string]maker{
		"owl":         owlObservatory{},
		"owl-shelter": owlShelter{},
	}
}

// made supplies the ordinary quick representation.
func (row owlObservatory) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt retains one physical instrument across circular tessellation.
func (row owlObservatory) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(groundValue("sides-" + detail))},
		object: good{dressed: groundLooks()}}
	assembly.object.dressed[body] = groundPaint("observatory platform",
		"enclosure-reflectance")
	pose := row.frame()
	row.base(assembly, pose)
	row.structure(assembly, pose)
	row.optics(assembly, pose)
	if err := assembly.shareComponents(); err != nil {
		return good{}, err
	}
	low := maths.V(-owlValue("extent-width"), spaceZero,
		-owlValue("extent-front"))
	high := maths.V(owlValue("extent-width"), owlValue("extent-height"),
		owlValue("extent-back"))
	made, err := assembly.finish(maths.SegmentBox(low, high, spaceZero))
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *made, nil
}

// frame is a picked parked pose, not the concept's full mount kinematics.
func (owlObservatory) frame() maths.Subspace {
	angle := owlValue("altitude-deg")*maths.Degree - math.Pi*spaceHalf
	seat := maths.YAxis.Scale(owlValue("primary-seat"))
	return tiltedFrame(seat, angle)
}

// optics keeps both source panel inventories and the four corrector roles.
func (row owlObservatory) optics(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	primary := owlSegmented{
		count:     int(research.Value("owl-primary-panels")),
		hole:      int(owlValue("primary-hole-rings")),
		diameter:  research.Value("owl-primary-metres"),
		curvature: research.Value("owl-primary-curvature-metres"),
		fill:      owlValue("panel-fill"), thickness: owlValue("panel-thickness")}
	assembly.add(primary, pose, spaceMirror)
	length := owlValue("structure-length")
	front := tiltedFrame(maths.YAxis.Scale(length), math.Pi).Within(pose)
	secondary := owlSegmented{
		count:    int(research.Value("owl-secondary-panels")),
		hole:     int(owlValue("secondary-hole-rings")),
		diameter: research.Value("owl-secondary-metres"),
		fill:     owlValue("panel-fill"), thickness: owlValue("panel-thickness")}
	assembly.add(secondary, front, spaceSecondary)
	row.corrector(assembly, pose)
}
