package manufacture

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
	offset := maths.V(width*spaceHalf, spaceZero, depth*spaceHalf)
	centre := corner.Add(offset)
	local := maths.NewSubspace(centre, maths.Identity, spaceOne)
	box := boxShape(width, height, depth)
	assembly.add(box, local.Within(pose), material)
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
