package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// civicObservatory suggests Griffith: a public building, not a big reflector.
type civicObservatory struct{}

const (
	civicRoof      slot = largeFifth + spaceOne
	civicObjective      = civicRoof + spaceOne
)

// civicValue reads the authored building dimensions from the choice book.
func civicValue(name string) float64 { return groundValue("civic-" + name) }

// made retains quick as the ordinary specimen.
func (row civicObservatory) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt keeps building and objective sizes fixed across surface details.
func (row civicObservatory) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	walls := groundPaint("civic walls", "civic-wall-reflectance")
	roof := groundPaint("copper dome proxy", "civic-roof-reflectance")
	tube := groundPaint("refractor tube", "structure-reflectance")
	assembly := &instrumentAssembly{fine: cut{
		around: int(groundValue("sides-" + detail))},
		object: good{dressed: looks{
			body:           walls,
			civicRoof:      roof,
			spaceStructure: tube,
			civicObjective: groundMirror("Zeiss objective proxy"),
		}}}
	row.building(assembly)
	row.towers(assembly)
	row.refractor(assembly)
	width, depth := civicValue("terrace-width"), civicValue("terrace-depth")
	near := maths.V(-width*spaceHalf, spaceZero, -depth*spaceHalf)
	far := maths.V(width*spaceHalf, civicValue("top"), depth*spaceHalf)
	bounds := maths.SegmentBox(near, far, spaceZero)
	object, err := assembly.finish(bounds)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return *object, nil
}

// building joins the wings, low rear planetarium and domed rotunda.
// Dimensions here are scenery, not measurements of Griffith's building.
func (row civicObservatory) building(assembly *instrumentAssembly) {
	width, depth := civicValue("terrace-width"), civicValue("terrace-depth")
	floor := civicValue("terrace-height")
	assembly.box(width, floor, depth,
		maths.V(-width*spaceHalf, spaceZero, -depth*spaceHalf), body)
	for _, sign := range []float64{-spaceOne, spaceOne} {
		centre := sign * civicValue("tower-offset")
		wing := civicValue("wing-width")
		back := -civicValue("wing-depth") * spaceHalf
		corner := maths.V(centre-wing*spaceHalf, floor, back)
		assembly.box(wing, civicValue("wing-height"),
			civicValue("wing-depth"),
			corner, body)
	}
	radius := civicValue("planetarium-radius")
	height := civicValue("planetarium-wall")
	at := maths.V(spaceZero, floor, civicValue("planetarium-offset"))
	assembly.add(groundDrum(radius, spaceZero, height),
		civicFrame(at), body)
	at.Y += height
	assembly.add(civicDome{radius: radius,
		height: civicValue("planetarium-roof")},
		civicFrame(at), civicRoof)
	width = civicValue("hall-width")
	depth = civicValue("hall-depth")
	assembly.box(width, civicValue("wing-height"), depth,
		maths.V(-width*spaceHalf, floor, -depth*spaceHalf), body)
	at = maths.V(spaceZero, floor, civicValue("rotunda-offset"))
	radius = civicValue("rotunda-radius")
	height = civicValue("rotunda-wall")
	assembly.add(groundDrum(radius, spaceZero, height),
		civicFrame(at), body)
	at.Y += height
	assembly.add(civicDome{radius: radius,
		height: civicValue("rotunda-roof")},
		civicFrame(at), civicRoof)
	row.entrance(assembly, floor)
}

// entrance gives the public building a portico under its central dome.
func (row civicObservatory) entrance(assembly *instrumentAssembly,
	floor float64,
) {
	width := civicValue("entry-width")
	front := civicValue("entry-front")
	height := civicValue("entry-height")
	assembly.box(width, height, civicValue("entry-depth"),
		maths.V(-width*spaceHalf, floor, front), body)
	canopy := civicValue("canopy-depth")
	assembly.box(width, civicValue("canopy-height"), canopy,
		maths.V(-width*spaceHalf, floor+height, front-canopy*spaceHalf),
		body)
	count := int(civicValue("columns"))
	spacing := width / float64(count)
	for index := range count {
		x := -width*spaceHalf + spacing*(float64(index)+spaceHalf)
		at := maths.V(x, floor, front-civicValue("column-setback"))
		column := groundDrum(civicValue("column-radius"),
			spaceZero, height)
		assembly.add(column, civicFrame(at), body)
	}
}

// towers differ: the east dome has a refractor, the west a closed roof.
// The visitor map identifies the east Zeiss and west solar instrument.
func (row civicObservatory) towers(assembly *instrumentAssembly) {
	radius := civicValue("tower-radius")
	floor := civicValue("terrace-height") + civicValue("wing-height")
	height := civicValue("tower-wall")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		at := maths.V(sign*civicValue("tower-offset"), floor, spaceZero)
		assembly.add(groundDrum(radius, spaceZero, height),
			civicFrame(at), body)
		at.Y += height
		if sign > spaceZero {
			opening := groundValue("opening-deg") * maths.Degree
			assembly.add(domeShell{radius: radius,
				thick:   civicValue("roof-thickness"),
				opening: opening},
				civicFrame(at), civicRoof)
			continue
		}
		assembly.add(civicDome{radius: radius, height: radius},
			civicFrame(at), civicRoof)
	}
}

// refractor preserves the guide's nominal 12-inch objective. Tube, mount
// and opaque objective proxy are authored, with no lens prescription.
func (row civicObservatory) refractor(assembly *instrumentAssembly) {
	floor := civicValue("terrace-height") + civicValue("wing-height")
	at := maths.V(civicValue("tower-offset"), floor, spaceZero)
	pier := civicValue("pier-height")
	pedestal := groundDrum(civicValue("pier-radius"), spaceZero, pier)
	assembly.add(pedestal, civicFrame(at), spaceStructure)
	at.Y += pier
	pose := tiltedFrame(at, groundValue("altitude-deg")*maths.Degree-
		math.Pi*spaceHalf)
	radius := research.Value("griffith-objective-inches") *
		metresPerInch * spaceHalf
	length := civicValue("tube-length")
	tube := groundDrum(radius+civicValue("tube-wall"), radius, length)
	assembly.add(tube, pose, spaceStructure)
	end := pose.Out(maths.YAxis.Scale(length))
	objective := maths.NewSubspace(end, pose.Orientation, spaceOne)
	lens := groundDrum(radius, spaceZero, civicValue("objective-thickness"))
	assembly.add(lens, objective, civicObjective)
}

// civicFrame places a structural part with local Y pointing up.
func civicFrame(at maths.Vec) maths.Subspace {
	return maths.NewSubspace(at, maths.Identity, spaceOne)
}

// civicDome closes a revolved elliptical cap with an exact axial pole.
// It has no slit; that belongs to the east instrument's shell.
type civicDome struct{ radius, height float64 }

// skin closes the base and pole without a rounded duplicate seam.
func (shape civicDome) skin(fine cut) surface {
	perLevel := int(civicValue("roof-cuts-per-level"))
	levels := max(spaceOne, fine.around/perLevel)
	profile := [][coordinatePair]float64{
		{spaceZero, spaceZero}, {shape.radius, spaceZero}}
	for level := spaceOne; level < levels; level++ {
		angle := math.Pi * spaceHalf * float64(level) / float64(levels)
		profile = append(profile, [coordinatePair]float64{
			shape.radius * math.Cos(angle),
			shape.height * math.Sin(angle)})
	}
	profile = append(profile,
		[coordinatePair]float64{spaceZero, shape.height})
	return (turnedProfile{points: profile}).skin(fine)
}
