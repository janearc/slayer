package manufacture

import (
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// cityModule names one mesh the street-level city models once and places
// many times: what it is, the family whose windows it carries, and how
// many bays a facade strip spans. Plant ignores family and bays.
type cityModule struct {
	what   string
	family string
	bays   int
}

// The shared modules, by what they are.
const (
	// cityStripModule is one storey of one face, its windows recessed.
	cityStripModule = "strip"
	// cityEntranceModule is a ground-floor strip with a door and canopy.
	cityEntranceModule = "entrance"
	// cityOverrunModule is a lift overrun on a flat roof.
	cityOverrunModule = "overrun"
	// cityCoolerModule is a rooftop cooling unit.
	cityCoolerModule = "cooler"
	// cityTankModule is a rooftop water tank.
	cityTankModule = "tank"
	// cityMastModule is a rooftop mast.
	cityMastModule = "mast"
	// cityChimneyModule is a rowhouse chimney stack.
	cityChimneyModule = "chimney"
)

// cityWhole is a whole width as two half-widths.
const cityWhole = 2

// cityOpening is a window or door in a strip's own frame: its sides,
// its foot and head, and whether it is a door.
type cityOpening struct {
	left, right, foot, head float64
	door                    bool
}

// module is the shared mesh for a module, made the first time a batch
// asks for it. A mesh the constructor refuses stops the batch.
func (batch *cityBatch) module(key cityModule) *mesh.Mesh {
	if made, known := batch.modules[key]; known {
		return made
	}
	made, err := key.skin().meshed()
	if err != nil && batch.trouble == nil {
		batch.trouble = err
	}
	batch.modules[key] = made
	return made
}

// skin is the module's surface in its own frame.
func (key cityModule) skin() surface {
	switch key.what {
	case cityOverrunModule:
		return cityBlock(cityValue("overrun-width"),
			cityValue("overrun-height"), cityValue("overrun-depth"))
	case cityCoolerModule:
		return cityBlock(cityValue("cooler-width"),
			cityValue("cooler-height"), cityValue("cooler-depth"))
	case cityTankModule:
		return cityPrism(cityValue("tank-radius"), cityValue("tank-height"))
	case cityMastModule:
		return cityPrism(cityValue("mast-radius"), cityValue("mast-height"))
	case cityChimneyModule:
		return cityBlock(cityValue("chimney-width"), cityValue("gable-rise")-
			cityValue("chimney-clearance"), cityValue("chimney-depth"))
	}
	return key.strip()
}

// cityWindow is a family's window width and height.
func cityWindow(family string) (float64, float64) {
	name := strings.TrimPrefix(family, "city-")
	return cityValue(name + "-window-width"),
		cityValue(name + "-window-height")
}

// openings are a strip's windows, one a bay, centred in each bay. An
// entrance strip's middle bay is a door instead, from the floor up.
func (key cityModule) openings() []cityOpening {
	pitch := cityValue("bay-pitch")
	wide, high := cityWindow(key.family)
	foot := (cityValue("module-height") - high) * cityValue("sill-share")
	door := cityValue("door-width") * spaceHalf
	found := make([]cityOpening, 0, key.bays)
	for bay := range key.bays {
		centre := (float64(bay) + spaceHalf) * pitch
		opening := cityOpening{left: centre - wide*spaceHalf,
			right: centre + wide*spaceHalf, foot: foot, head: foot + high}
		if key.what == cityEntranceModule && bay == key.bays/cityWhole {
			opening = cityOpening{left: centre - door, right: centre + door,
				head: cityValue("door-height"), door: true}
		}
		found = append(found, opening)
	}
	return found
}

// strip is the wall around each opening, the opening's reveals back to
// the core, and its sill, mullion and lintel or its canopy. The wall
// plane is z nought and the core lies the reveal depth behind it.
func (key cityModule) strip() surface {
	pitch, tall := cityValue("bay-pitch"), cityValue("module-height")
	var skin surface
	for bay, opening := range key.openings() {
		low, high := float64(bay)*pitch, float64(bay+1)*pitch
		skin = append(skin, cityWall(low, opening.left, 0, tall)...)
		skin = append(skin, cityWall(opening.right, high, 0, tall)...)
		skin = append(skin, cityWall(opening.left, opening.right, 0,
			opening.foot)...)
		skin = append(skin, cityWall(opening.left, opening.right,
			opening.head, tall)...)
		skin = append(skin, opening.reveals()...)
		skin = append(skin, key.dressing(opening)...)
	}
	return skin
}

// cityWall is a rectangle of the wall plane facing out, or nothing when
// it has no height, as below a door.
func cityWall(left, right, foot, head float64) surface {
	if head <= foot {
		return nil
	}
	return cityQuad(maths.Vec{X: left, Y: foot}, maths.Vec{X: right, Y: foot},
		maths.Vec{X: right, Y: head}, maths.Vec{X: left, Y: head}, maths.ZAxis)
}

// reveals are an opening's two sides and head, from the wall plane back
// to the core, each facing into the opening.
func (opening cityOpening) reveals() surface {
	back := -cityValue("reveal-depth")
	side := func(x float64, inward maths.Vec) surface {
		return cityQuad(maths.Vec{X: x, Y: opening.foot},
			maths.Vec{X: x, Y: opening.foot, Z: back},
			maths.Vec{X: x, Y: opening.head, Z: back},
			maths.Vec{X: x, Y: opening.head}, inward)
	}
	skin := side(opening.left, maths.XAxis)
	skin = append(skin, side(opening.right, maths.XAxis.Neg())...)
	return append(skin, cityQuad(maths.Vec{X: opening.left, Y: opening.head},
		maths.Vec{X: opening.left, Y: opening.head, Z: back},
		maths.Vec{X: opening.right, Y: opening.head, Z: back},
		maths.Vec{X: opening.right, Y: opening.head}, maths.YAxis.Neg())...)
}

// dressing is a window's sill and mullion, and a rowhouse's lintel, or
// a door's canopy.
func (key cityModule) dressing(opening cityOpening) surface {
	back := -cityValue("reveal-depth")
	over := cityValue("sill-overhang")
	if opening.door {
		return opening.canopy()
	}
	middle := (opening.left + opening.right) * spaceHalf
	mullion := cityValue("mullion-width") * spaceHalf
	skin := cityBox(opening.left-over, opening.right+over,
		opening.foot-cityValue("sill-thickness"), opening.foot, back,
		cityValue("sill-projection"))
	skin = append(skin, cityBox(middle-mullion, middle+mullion, opening.foot,
		opening.head, back, back+cityValue("mullion-depth"))...)
	if key.family != "city-rowhouse" {
		return skin
	}
	return append(skin, cityBox(opening.left-over, opening.right+over,
		opening.head, opening.head+cityValue("lintel-height"),
		back*spaceHalf, cityValue("lintel-projection"))...)
}

// canopy shelters a door, reaching out to the slab edges' face.
func (opening cityOpening) canopy() surface {
	over := cityValue("canopy-overhang")
	foot := opening.head + cityValue("canopy-gap")
	return cityBox(opening.left-over, opening.right+over, foot,
		foot+cityValue("canopy-thickness"), -cityValue("reveal-depth"),
		cityValue("facade-projection"))
}

// cityQuad is two faces over four corners in turn, wound to face out.
func cityQuad(a, b, c, d, out maths.Vec) surface {
	if b.Sub(a).Cross(c.Sub(a)).Dot(out) < 0 {
		b, d = d, b
	}
	return surface{{a, b, c}, {a, c, d}}
}

// cityBox is a closed box between two corners.
func cityBox(left, right, foot, head, back, front float64) surface {
	middle := maths.Vec{X: (left + right) * spaceHalf, Y: foot,
		Z: (back + front) * spaceHalf}
	return cityBlock(right-left, head-foot, front-back).placed(
		maths.NewSubspace(middle, maths.Identity, spaceOne))
}

// cityBlock is a closed box standing centred on its frame's start.
func cityBlock(wide, tall, deep float64) surface {
	return boxShape(wide, tall, deep).skin(cut{})
}

// cityPrism is a closed upright prism of plant-sides sides about its
// frame's start, standing on it.
func cityPrism(radius, tall float64) surface {
	sides := int(cityValue("plant-sides"))
	ring := func(index int, y float64) maths.Vec {
		turn := fullTurn * float64(index) / float64(sides)
		return maths.Vec{X: radius * math.Cos(turn), Y: y,
			Z: -radius * math.Sin(turn)}
	}
	top, bottom := maths.Vec{Y: tall}, maths.Vec{}
	var skin surface
	for index := range sides {
		following := (index + 1) % sides
		low, next := ring(index, 0), ring(following, 0)
		high, after := ring(index, tall), ring(following, tall)
		skin = append(skin, cityQuad(low, next, after, high, low.Add(next))...)
		skin = append(skin, cityTriangle(top, high, after, maths.YAxis),
			cityTriangle(bottom, next, low, maths.YAxis.Neg()))
	}
	return skin
}

// cityTriangle is one face over three corners, wound to face out.
func cityTriangle(a, b, c, out maths.Vec) face {
	if b.Sub(a).Cross(c.Sub(a)).Dot(out) < 0 {
		b, c = c, b
	}
	return face{a, b, c}
}
