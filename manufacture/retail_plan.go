package manufacture

import (
	"slices"
	"strings"

	"github.com/janearc/slayer/maths"
)

// The ways a part, a light or a module moves when its unit is wider than
// the unit's least: carried along with the right edge, carried to keep to
// the middle, or filling the extra width.
const (
	retailRight  = "right"
	retailCentre = "centre"
	retailFill   = "fill"
)

// retailPalettePrefix opens the name of a palette where a material is
// named: the unit's seed picks the material from the palette's list.
const retailPalettePrefix = "@"

// retailMostOptions is the most options a unit may have. A unit and each
// of its options is dressed from a kit, and each kit draws from streams
// of its own.
const retailMostOptions = 8

// retailMostKits is the most kits a unit is dressed from: its own, and one
// for each option.
const retailMostKits = retailMostOptions + 1

// retailSlide is how far something that moves along a way is carried when
// its unit has grown by grow metres.
func retailSlide(along string, grow float64) float64 {
	switch along {
	case retailRight:
		return grow
	case retailCentre:
		return grow * spaceHalf
	}
	return 0
}

// slid is the part as it stands in a unit that has grown by grow metres:
// carried along, stretched, or given more places, as its Along says.
func (part retailPart) slid(grow float64) retailPart {
	if part.Along == retailFill {
		return part.filled(grow)
	}
	shift := retailSlide(part.Along, grow)
	if shift == 0 {
		return part
	}
	return part.moved(shift)
}

// moved is the part carried along x by shift, with the lists it keeps
// copied, so that the data is not changed.
func (part retailPart) moved(shift float64) retailPart {
	switch {
	case part.Box != nil:
		part.Box = slices.Clone(part.Box)
		part.Box[0] += shift
		part.Box[retailBoxRight] += shift
	case part.Slab != nil:
		part.Slab = slices.Clone(part.Slab)
		part.Slab[0] += shift
		part.Slab[1] += shift
	case part.Prism != nil:
		part.Prism = slices.Clone(part.Prism)
		part.Prism[0] += shift
	}
	return part
}

// filled is the part filling grow metres more of width: given as many
// more places as fit in the width if it repeats along x, and otherwise
// stretched to its right.
func (part retailPart) filled(grow float64) retailPart {
	step := retailPoint(part.Step).X
	if part.Count > 1 && step > 0 {
		part.Count += int(grow / step)
		return part
	}
	switch {
	case part.Box != nil:
		part.Box = slices.Clone(part.Box)
		part.Box[retailBoxRight] += grow
	case part.Slab != nil:
		part.Slab = slices.Clone(part.Slab)
		part.Slab[1] += grow
	}
	return part
}

// choose is the variant a share of the way through the weights of an
// option's variants.
func (option retailOption) choose(share float64) retailVariant {
	total := 0.0
	for _, variant := range option.Variants {
		total += variant.Weight
	}
	reach, last := share*total, len(option.Variants)-1
	for _, variant := range option.Variants[:last] {
		reach -= variant.Weight
		if reach < 0 {
			return variant
		}
	}
	return option.Variants[last]
}

// resolve is the material a name stands for: itself, or, for the name of
// a palette, the one the unit's seed picked from it.
func (builder *retailBuilder) resolve(name string) string {
	palette, named := strings.CutPrefix(name, retailPalettePrefix)
	if !named {
		return name
	}
	return builder.picks[palette]
}

// plan draws what is the unit's own: how many bays wide it is, which way
// it faces, and the material it has from each palette.
func (builder *retailBuilder) plan(index int, unit retailUnit, from seed) {
	bays := unit.Bays
	if unit.BaysMost > unit.Bays {
		more := unit.BaysMost - unit.Bays + 1
		bays += int(from.draw(retailWidthDraw, index, 0) * float64(more))
	}
	pitch := cityValue("bay-pitch")
	builder.width = float64(bays) * pitch
	builder.extra = float64(bays-unit.Bays) * pitch
	builder.mirror = from.draw(retailMirrorDraw, index, 0) < retailMirrorShare
	builder.picks = map[string]string{}
	for place, name := range builder.book.paletteNames() {
		options := builder.book.Palettes[name]
		share := from.draw(retailPaletteDraw, index*retailMostStreams+place, 0)
		builder.picks[name] = options[int(share*float64(len(options)))]
	}
}

// kits are what a unit is dressed from: its own kit, and the variant of
// each option it takes, drawn from the seed. An option that wants more
// bays than the unit has is not offered.
func (builder *retailBuilder) kits(index int, unit retailUnit,
	from seed) []retailKit {
	kits := []retailKit{unit.Kit}
	pitch := cityValue("bay-pitch")
	for which, option := range unit.Options {
		stream := index*retailMostStreams + which
		if builder.width < float64(option.Wide)*pitch-retailFit ||
			from.draw(retailOptionDraw, stream, 0) >= option.Chance {
			continue
		}
		kits = append(kits, option.choose(from.draw(retailOptionDraw, stream,
			1)).Kit)
	}
	return kits
}

// dress draws a kit: its parts, its modules, its lights, its stickers
// and its props. The stream is the kit's place among all the unit's kits
// in the run, which its stickers and props draw from.
func (builder *retailBuilder) dress(stream int, kit retailKit, from seed) {
	for _, piece := range kit.Parts {
		builder.part(piece, maths.Vec{}, builder.extra)
	}
	for _, one := range kit.Uses {
		builder.use(one)
	}
	for _, one := range kit.Lights {
		if builder.reaches(one.From) {
			builder.lamp(one)
		}
	}
	builder.stickers(stream, kit.Scatter, from)
	builder.props(stream, kit.Props, from)
}

// use draws a module where a unit puts it: its parts moved to the place,
// the ones that fill grown by how much more the module spans than it is
// wide, and by the unit's extra width if the module fills it.
func (builder *retailBuilder) use(one retailUse) {
	if !builder.reaches(one.From) {
		return
	}
	module := builder.book.Modules[one.Module]
	grow := 0.0
	if one.Span > 0 {
		grow = one.Span - module.Width
	}
	if one.Along == retailFill {
		grow += builder.extra
	}
	at := retailPoint(one.At)
	at.X += retailSlide(one.Along, builder.extra)
	for _, piece := range module.Parts {
		builder.part(piece, at, grow)
	}
}
