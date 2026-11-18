package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// These family ranges and material colours are authored architectural
// scenery. Quick uses at most three broad glazing bands per elevation;
// it is a distant city representation, with no interiors or traffic agents.
type cityFamily struct{ name string }

type cityDesign struct {
	width, depth, height float64
	floors               int
	tower, gable         bool
	wall                 slot
}

const (
	cityRoof     slot = 4
	cityGlass    slot = 5
	cityRoad     slot = 6
	cityPavement slot = 7
	cityIsland   slot = 8
)

// cityLooks is a shared finite palette so independent buildings batch.
func cityLooks() looks {
	return looks{0: plain("city pale plaster", .62, .58, .49),
		1:        plain("city brick", .35, .16, .1),
		2:        plain("city grey masonry", .36, .39, .4),
		3:        plain("city warm stone", .49, .4, .29),
		cityRoof: plain("city roof and coping", .16, .18, .19),
		cityGlass: &thing.Material{Name: "city dark glazing", Colour: maths.RGB(.05,
			.11, .15), Gloss: .65},
		cityRoad:     plain("city asphalt", .055, .06, .065),
		cityPavement: plain("city lot paving", .27, .28, .26),
		cityIsland:   plain("city planted traffic islands", .09, .17, .055),
		cityPlant:    plain("city rooftop plant", .4, .41, .4)}
}

// cityBuildingDesign picks dimensions once, shared by planning and geometry.
func cityBuildingDesign(name string, from seed) cityDesign {
	design := cityDesign{width: cityRange("width", from.draw(1, 0, 0)),
		depth:  cityRange("depth", from.draw(1, 0, 1)),
		floors: cityCount("house-floors", from.draw(1, 0, 2)),
		gable:  name == "city-rowhouse", wall: slot(from.draw(1, 0, 3) * 4)}
	switch name {
	case "city-apartment":
		design.floors = cityCount("apartment-floors", from.draw(1, 0, 2))
	case "city-tower":
		design.floors = cityCount("tower-floors", from.draw(1, 0, 2))
		design.tower = true
	}
	design.height = float64(design.floors) * cityRange("storey", from.draw(1, 0, 4))
	return design
}

// bounds encloses the roof cap as well as the building's walls.
func (design cityDesign) bounds() maths.Box {
	top := design.height + cityValue("roof-thickness") - cityValue("roof-seat")
	if design.gable {
		top = design.height + cityValue("gable-rise") - cityValue("roof-seat")
	}
	return maths.Box{Min: maths.V(-design.width/2, 0, -design.depth/2),
		Max: maths.V(design.width/2, top, design.depth/2)}
}

// parts shares primitive boxes and material roles across all three families.
// The tower has a broad podium, inset shaft and a narrower crown. Low-rise
// roofs have a separate coping cap. All solids are outward-wound and closed.
func (design cityDesign) parts(detail string) []part {
	assembly := instrumentAssembly{}
	w, d, h := design.width, design.depth, design.height
	box := func(w, h, d, x, y, z float64, of slot) {
		assembly.box(w, h, d, maths.V(x-w/2, y, z-d/2), of)
	}
	if design.tower {
		box(w, h*cityValue("podium-height-share"), d, 0, 0, 0, design.wall)
		box(w*cityValue("shaft-width-share"), h*cityValue("shaft-height-share"),
			d*cityValue("shaft-width-share"), 0,
			h*cityValue("podium-height-share")-cityValue("joint-overlap"), 0, design.wall)
		box(w*cityValue("crown-width-share"),
			h*cityValue("crown-height-share")+cityValue("joint-overlap"),
			d*cityValue("crown-width-share"), 0,
			h*cityValue("crown-base-share")-cityValue("joint-overlap"), 0, design.wall)
		box(w*cityValue("crown-roof-share"), cityValue("roof-thickness"),
			d*cityValue("crown-roof-share"), 0, h-cityValue("roof-seat"), 0, cityRoof)
	} else {
		box(w, h, d, 0, 0, 0, design.wall)
		if design.gable {
			assembly.add(cityGable{w, d, cityValue("gable-rise")},
				maths.NewSubspace(maths.V(0, h-cityValue("roof-seat"), 0), maths.Identity, 1), cityRoof)
		} else {
			box(w, cityValue("roof-thickness"), d, 0, h-cityValue("roof-seat"), 0, cityRoof)
		}
	}
	if detail == "hull" {
		return assembly.object.parts
	}
	bands := min(design.floors, int(cityValue("glazing-bands")))
	shaft := 1.0
	if design.tower {
		shaft = cityValue("shaft-width-share")
	}
	for band := range bands {
		y := (float64(band) + .6) / float64(bands) * h * .78
		if design.tower {
			y = h*.22 + (float64(band)+.5)/float64(bands)*h*.62
		}
		// Glass penetrates the wall slightly, avoiding coincident surfaces.
		box(w*shaft*cityValue("glazing-width-share"), cityValue("glazing-height"),
			cityValue("glazing-thickness"), 0, y, d*shaft/2+cityValue("glazing-offset"), cityGlass)
		box(cityValue("glazing-thickness"), cityValue("glazing-height"),
			d*shaft*cityValue("glazing-depth-share"), w*shaft/2+cityValue("glazing-offset"),
			y, 0, cityGlass)
	}
	return assembly.object.parts
}

// made constructs the same family used by the city at ordinary quick detail.
func (row cityFamily) made(from seed) (good, error) { return row.madeAt(from, "quick") }

// madeAt refuses unsupported detail and unsown seeds before construction.
func (row cityFamily) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	if !cityLevel(detail) {
		return good{}, fmt.Errorf("manufacture: %s has no %s representation", row.name, detail)
	}
	design := cityBuildingDesign(row.name, from)
	if detail == cityStreetLevel {
		return design.streetLevel(from)
	}
	// The glazing's outer surfaces are inside the envelope used by layout:
	// glazing projects six centimetres beyond the primary wall surfaces.
	made := good{parts: design.parts(detail), dressed: cityLooks()}
	extent := design.envelope()
	made.extent = &extent
	return made, nil
}

// streetLevel makes one building alone at street level, its modules
// shared within it as they are across a city, in the same envelope.
func (design cityDesign) streetLevel(from seed) (good, error) {
	batch := newCityBatch()
	batch.detailed(design, from, maths.Local(), false)
	made, err := batch.finish()
	if err != nil {
		return good{}, err
	}
	extent := design.envelope()
	made.extent = &extent
	return made, nil
}

// cityEnvelope returns actual geometry bounds without tessellating all parts.
func (design cityDesign) envelope() maths.Box {
	bounds := design.bounds()
	if !design.tower {
		bounds.Max.X += cityValue("glazing-offset") + cityValue("glazing-thickness")/2
		bounds.Max.Z += cityValue("glazing-offset") + cityValue("glazing-thickness")/2
	}
	return bounds
}

// cityGable is a closed triangular roof prism on the shared box footprint.
type cityGable struct{ width, depth, rise float64 }

// skin winds the convex roof's eight faces away from its centre.
func (roof cityGable) skin(cut) surface {
	w, d := roof.width/2, roof.depth/2
	points := []maths.Vec{maths.V(-w, 0, -d), maths.V(w, 0, -d),
		maths.V(w, 0, d), maths.V(-w, 0, d), maths.V(0, roof.rise, -d), maths.V(0, roof.rise, d)}
	triangles := [][3]int{{0, 1, 4}, {3, 5, 2}, {0, 3, 2}, {0, 2, 1},
		{0, 4, 5}, {0, 5, 3}, {1, 2, 5}, {1, 5, 4}}
	var skin surface
	centre := maths.V(0, roof.rise/3, 0)
	for _, indices := range triangles {
		a, b, c := points[indices[0]], points[indices[1]], points[indices[2]]
		if b.Sub(a).Cross(c.Sub(a)).Dot(a.Add(b).Add(c).Scale(1.0/3).Sub(centre)) < 0 {
			b, c = c, b
		}
		skin = append(skin, face{a, b, c})
	}
	return skin
}
