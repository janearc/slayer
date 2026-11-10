package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/scene/thing"
)

type groundObservatory struct {
	diameter, height, aperture float64
	segments                   int
	equatorial, cylindrical    bool
	survey                     *surveyOptics
	large                      *largeOptics
	pointing                   *maths.Subspace
}

// metresPerInch is defined by the international inch, exactly 0.0254 m.
const metresPerInch = 0.0254

// groundRows keeps measured defaults distinct from enclosure choices.
func groundRows() map[string]maker {
	keck := groundObservatory{
		diameter: groundValue("keck-enclosure-diameter"),
		height:   groundValue("keck-enclosure-height"),
		aperture: research.Value("keck-primary-metres"),
		segments: int(research.Value("keck-primary-segments"))}
	haleAperture := research.Value("palomar-primary-inches") * metresPerInch
	hale := groundObservatory{
		diameter: research.Value("palomar-enclosure-diameter-metres"),
		height:   research.Value("palomar-enclosure-height-metres"),
		aperture: haleAperture, equatorial: true}
	subaru := groundObservatory{
		diameter:    research.Value("subaru-enclosure-base-metres"),
		height:      research.Value("subaru-enclosure-height-metres"),
		aperture:    research.Value("subaru-primary-effective-metres"),
		cylindrical: true}
	return map[string]maker{
		"palomar": hale,
		"keck-i":  keck, "keck-ii": keck,
		"subaru": subaru,
		"rubin":  surveyGround(),
		"elt":    largeGround(),
	}
}

// made uses the ordinary quick representation.
func (row groundObservatory) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt retains one physical envelope and optical pose across details.
func (row groundObservatory) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(groundValue("sides-" + detail))},
		object: good{dressed: groundLooks()}}
	row.enclosure(assembly)
	pose := row.opticalFrame()
	if row.equatorial {
		row.polarMount(assembly)
	} else {
		row.forkMount(assembly)
	}
	row.optics(assembly, pose)
	radius := row.diameter * spaceHalf
	bounds := maths.SegmentBox(maths.V(-radius, spaceZero, -radius),
		maths.V(radius, row.height, radius), spaceZero)
	if row.survey != nil {
		bounds = row.survey.facilityBounds(row)
	}
	made, err := assembly.finish(bounds)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *made, nil
}

// opticalFrame points along the open observing slit, above the primary.
func (row groundObservatory) opticalFrame() maths.Subspace {
	if row.pointing != nil {
		return *row.pointing
	}
	angle := groundValue("altitude-deg")*maths.Degree - math.Pi*spaceHalf
	height := row.height * groundValue("primary-height")
	return tiltedFrame(maths.YAxis.Scale(height), angle)
}

// tiltedFrame rotates an instrument around its east-west axis.
func tiltedFrame(at maths.Vec, angle float64) maths.Subspace {
	orientation := maths.Rotation(maths.XAxis, angle)
	return maths.NewSubspace(at, orientation, spaceOne)
}

// groundLooks separates enclosure, tube, supports and the two mirrors.
func groundLooks() looks {
	primary := groundMirror("primary mirror")
	secondary := groundMirror("secondary mirror")
	enclosure := groundPaint("observatory enclosure",
		"enclosure-reflectance")
	structure := groundPaint("telescope mount and truss",
		"structure-reflectance")
	return looks{
		body:           enclosure,
		leaf:           groundPaint("optical tube", "tube-reflectance"),
		spaceStructure: structure,
		spaceMirror:    primary, spaceSecondary: secondary,
	}
}

// groundMirror keeps the same authored coating on the reflective surfaces.
func groundMirror(name string) *thing.Material {
	shade := maths.Grey(groundValue("mirror-reflectance"))
	return thing.Polished(name, shade,
		groundValue("mirror-gloss"), groundValue("mirror-index"))
}

// groundPaint is an authored neutral surface, not measured coating data.
func groundPaint(name, choice string) *thing.Material {
	shade := groundValue(choice)
	return plain(name, shade, shade, shade)
}
