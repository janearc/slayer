package manufacture

import (
	"math"

	"github.com/janearc/celestial/measure"
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
	architecture, err := Building(row.buildingSpec())
	if err != nil {
		return good{}, err
	}
	architecture.how.(buildingMaker).appendTo(assembly, map[BuildingSurface]slot{
		BuildingWalls: body, BuildingRoof: civicRoof})
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

// buildingSpec joins the wings, observatories, entrance and towers from the
// shared closed-form building vocabulary. Dimensions are scenery, not
// measurements of Griffith's building. Part order follows the original maker.
func (row civicObservatory) buildingSpec() BuildingSpec {
	width, depth := civicValue("terrace-width"), civicValue("terrace-depth")
	floor := civicValue("terrace-height")
	parts := make([]BuildingPart, 0, 24)
	box := func(width, height, depth float64, at maths.Vec, surface BuildingSurface) {
		parts = append(parts, BuildingPart{Form: BuildingBox, Position: at,
			Width: measure.Metres(width), Height: measure.Metres(height),
			Depth: measure.Metres(depth), Surface: surface})
	}
	drum := func(radius, inner, height float64, at maths.Vec, surface BuildingSurface) {
		parts = append(parts, BuildingPart{Form: BuildingDrum, Position: at,
			Radius: measure.Metres(radius), InnerRadius: measure.Metres(inner),
			Height: measure.Metres(height), Surface: surface})
	}
	dome := func(radius, height float64, at maths.Vec, surface BuildingSurface) {
		parts = append(parts, BuildingPart{Form: BuildingDome, Position: at,
			Radius: measure.Metres(radius), Height: measure.Metres(height),
			DomeCutsPerLevel: int(civicValue("roof-cuts-per-level")), Surface: surface})
	}
	box(width, floor, depth, maths.V(-width*spaceHalf, spaceZero, -depth*spaceHalf), BuildingWalls)
	for _, sign := range []float64{-spaceOne, spaceOne} {
		centre := sign * civicValue("tower-offset")
		wing := civicValue("wing-width")
		back := -civicValue("wing-depth") * spaceHalf
		corner := maths.V(centre-wing*spaceHalf, floor, back)
		box(wing, civicValue("wing-height"), civicValue("wing-depth"), corner, BuildingWalls)
	}
	radius := civicValue("planetarium-radius")
	height := civicValue("planetarium-wall")
	at := maths.V(spaceZero, floor, civicValue("planetarium-offset"))
	drum(radius, spaceZero, height, at, BuildingWalls)
	at.Y += height
	dome(radius, civicValue("planetarium-roof"), at, BuildingRoof)
	width = civicValue("hall-width")
	depth = civicValue("hall-depth")
	box(width, civicValue("wing-height"), depth,
		maths.V(-width*spaceHalf, floor, -depth*spaceHalf), BuildingWalls)
	at = maths.V(spaceZero, floor, civicValue("rotunda-offset"))
	radius = civicValue("rotunda-radius")
	height = civicValue("rotunda-wall")
	drum(radius, spaceZero, height, at, BuildingWalls)
	at.Y += height
	dome(radius, civicValue("rotunda-roof"), at, BuildingRoof)
	// The entry and portico precede the paired instrument towers as before.
	width = civicValue("entry-width")
	front := civicValue("entry-front")
	height = civicValue("entry-height")
	box(width, height, civicValue("entry-depth"), maths.V(-width*spaceHalf, floor, front), BuildingWalls)
	canopy := civicValue("canopy-depth")
	box(width, civicValue("canopy-height"), canopy,
		maths.V(-width*spaceHalf, floor+height, front-canopy*spaceHalf), BuildingWalls)
	count := int(civicValue("columns"))
	spacing := width / float64(count)
	for index := range count {
		x := -width*spaceHalf + spacing*(float64(index)+spaceHalf)
		at := maths.V(x, floor, front-civicValue("column-setback"))
		drum(civicValue("column-radius"), spaceZero, height, at, BuildingWalls)
	}
	// The paired towers retain west-then-east emission and their distinct roofs.
	radius = civicValue("tower-radius")
	towerFloor := floor + civicValue("wing-height")
	height = civicValue("tower-wall")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		at := maths.V(sign*civicValue("tower-offset"), towerFloor, spaceZero)
		drum(radius, spaceZero, height, at, BuildingWalls)
		at.Y += height
		if sign > spaceZero {
			parts = append(parts, BuildingPart{Form: BuildingSlitDome, Position: at,
				Radius: measure.Metres(radius), Thickness: measure.Metres(civicValue("roof-thickness")),
				Opening: measure.Degrees(groundValue("opening-deg")), Surface: BuildingRoof})
			continue
		}
		dome(radius, radius, at, BuildingRoof)
	}
	return BuildingSpec{Name: "griffith", Width: measure.Metres(civicValue("terrace-width")),
		Depth: measure.Metres(civicValue("terrace-depth")), Height: measure.Metres(civicValue("top")),
		Materials: []BuildingMaterial{
			{Surface: BuildingWalls, Name: "civic walls", Colour: maths.Grey(groundValue("civic-wall-reflectance"))},
			{Surface: BuildingRoof, Name: "copper dome proxy", Colour: maths.Grey(groundValue("civic-roof-reflectance"))},
		}, Parts: parts}
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
