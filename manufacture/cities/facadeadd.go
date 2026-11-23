package cities

import "github.com/janearc/slayer/maths"

// measured is one of the culture's sizes for what is added to a face, by
// the name its data gives it.
func (facade facade) measured(name string) float64 {
	if facade.Measure == nil {
		return 0
	}
	return facade.Measure(name)
}

// added is what has been added to one opening of a storey: what is under
// a window, or an awning and a lamp at a door.
func (facade facade) added(opening facadeOpening) []placedElement {
	extra := opening.Extra
	switch {
	case opening.Door:
		parts := append(facade.awning(opening), facade.wallLamp(opening)...)
		return append(parts, facade.portal(opening)...)
	case extra.Under == guardUnder:
		return facade.guard(opening)
	case extra.Under == "flower-box":
		return facade.flowerBox(opening)
	case extra.Under == "balcony":
		return facade.balcony(opening)
	case extra.Under == "aircon":
		return facade.aircon(opening)
	}
	return nil
}

// foot is the height of the top of the sill, the level whatever stands
// under a window is hung from.
func (facade facade) foot(opening facadeOpening) float64 {
	return opening.Sill - facade.Finish.SillThickness
}

// flowerBox is a box under a window with leaves in it and blooms along it.
func (facade facade) flowerBox(opening facadeOpening) []placedElement {
	high, deep := facade.measured("box-height"), facade.measured("box-depth")
	bloom, pitch := facade.measured("bloom-size"),
		facade.measured("bloom-pitch")
	top := facade.foot(opening)
	extra := opening.Extra
	parts := []placedElement{
		put("planter", Box{opening.Width, high, deep}, extra.Planter,
			maths.V(opening.Left, top-high, 0)),
		put("foliage", Box{opening.Width - 2*bloom/half, bloom, deep - bloom},
			extra.Leaf, maths.V(opening.Left+bloom/half, top, bloom/half)),
	}
	for along := pitch / half; along+bloom < opening.Width; along += pitch {
		parts = append(parts, put("bloom", Box{bloom, bloom, bloom},
			extra.Bloom, maths.V(opening.Left+along, top+bloom/half,
				(deep-bloom)/half)))
	}
	return parts
}

// aircon is an air-conditioning unit hung on the wall under a window.
func (facade facade) aircon(opening facadeOpening) []placedElement {
	wide, high := facade.measured("unit-width"), facade.measured("unit-height")
	return []placedElement{put("unit", Box{wide, high, facade.measured("unit-depth")},
		opening.Extra.Unit, maths.V(opening.Left+(opening.Width-wide)/half,
			facade.foot(opening)-high-facade.Finish.SillThickness, 0))}
}

// balcony is a floor under a window, standing out of the wall, with a rail
// of wrought iron round its three open sides: a top bar and uprights.
func (facade facade) balcony(opening facadeOpening) []placedElement {
	side, deep := facade.measured("balcony-side"),
		facade.measured("balcony-depth")
	slab, rail := facade.measured("balcony-slab"), facade.measured("rail-top")
	left, wide := opening.Left-side, opening.Width+2*side
	floor := facade.foot(opening)
	parts := []placedElement{put("balcony", Box{wide, slab, deep}, facade.Wall,
		maths.V(left, floor-slab, 0))}
	parts = append(parts, facade.rails(left, floor, wide, deep, rail)...)
	return append(parts, facade.uprights(left, floor, wide, deep, rail)...)
}

// rails are the top bars of a balcony's rail: along the front and up each
// side.
func (facade facade) rails(left, floor, wide, deep, rail float64) []placedElement {
	bar := facade.measured("rail-bar")
	top := floor + rail - bar
	return []placedElement{
		put("handrail", Box{wide, bar, bar}, facade.Iron, maths.V(left, top,
			deep-bar)),
		put("handrail", Box{bar, bar, deep}, facade.Iron,
			maths.V(left, top, 0)),
		put("handrail", Box{bar, bar, deep}, facade.Iron,
			maths.V(left+wide-bar, top, 0)),
	}
}

// uprights are the thin bars under a balcony's top bars, a pitch apart
// along the front and the two sides.
func (facade facade) uprights(left, floor, wide, deep, rail float64) []placedElement {
	pitch, thin := facade.measured("baluster-pitch"),
		facade.measured("baluster-bar")
	high := rail - facade.measured("rail-bar")
	var parts []placedElement
	for along := pitch / half; along+thin <= wide; along += pitch {
		parts = append(parts, put("upright", Box{thin, high, thin}, facade.Iron,
			maths.V(left+along, floor, deep-thin)))
	}
	for along := pitch / half; along < deep-thin; along += pitch {
		parts = append(parts,
			put("upright", Box{thin, high, thin}, facade.Iron,
				maths.V(left, floor, along)),
			put("upright", Box{thin, high, thin}, facade.Iron,
				maths.V(left+wide-thin, floor, along)))
	}
	return parts
}

// awning is a slab of cloth over a door, hung from the wall above it and
// sloping down away from it.
func (facade facade) awning(opening facadeOpening) []placedElement {
	if opening.Extra.Awning == nil {
		return nil
	}
	side := facade.measured("awning-side")
	slope := maths.Rotation(maths.XAxis,
		facade.measured("awning-tilt")*maths.Degree)
	return []placedElement{{Place: maths.NewSubspace(maths.V(opening.Left-side,
		opening.Sill+opening.Height+facade.measured("awning-rise"), 0), slope,
		0), Element: solid{Name: "awning", Material: opening.Extra.Awning,
		Form: Box{opening.Width + 2*side, facade.measured("awning-thick"),
			facade.measured("awning-depth")}}}}
}

// wallLamp is a lamp on the wall beside a door, glowing, and a light.
func (facade facade) wallLamp(opening facadeOpening) []placedElement {
	if opening.Extra.Lamp == nil {
		return nil
	}
	size := facade.measured("lamp-fixture")
	at := maths.V(opening.Left-facade.measured("lamp-beside")-size,
		opening.Sill+opening.Height-facade.measured("lamp-beside"), 0)
	return []placedElement{{Place: standingAt(at), Element: solid{Name: "lamp",
		Form: Box{size, size * half, size}, Material: opening.Extra.Lamp,
		Lamp: &Lamp{Heat: wallLampHeat,
			Luminance: facade.measured("wall-lamp-strength")}}}}
}

// fixtures is what a bay has added to it as a whole, standing on the
// face: a dish, a downpipe, a cable, a laundry line and a shop sign.
func (facade facade) fixtures() []placedElement {
	fit := facade.Fixtures
	var parts []placedElement
	if fit.Dish {
		parts = append(parts, facade.dish()...)
	}
	if fit.Pipe {
		parts = append(parts, facade.pipe()...)
	}
	if fit.Cable {
		parts = append(parts, facade.cable()...)
	}
	if fit.Laundry {
		parts = append(parts, facade.laundry()...)
	}
	if fit.Sign {
		parts = append(parts, facade.sign()...)
	}
	return parts
}

// dish is a satellite dish on an arm from the wall near the top of the
// face, leaning up.
func (facade facade) dish() []placedElement {
	size, arm := facade.measured("dish-size"), facade.measured("dish-arm")
	across := facade.Width * dishPlace
	high := facade.Height - dishDown*facade.storeyHeight()
	lean := maths.Rotation(maths.XAxis, -facade.measured("dish-tilt")*
		maths.Degree)
	return []placedElement{
		put("arm", Box{armWidth, armWidth, arm}, facade.Iron,
			maths.V(across, high, 0)),
		{Place: maths.NewSubspace(maths.V(across-size/half, high, arm), lean,
			0), Element: solid{Name: "dish", Material: facade.Fixtures.Receiver,
			Form: Box{size, size, armWidth}}},
	}
}

// dishPlace is how far along a face, as a share of its width, a dish is
// hung, and dishDown how far down from the top, in storeys. armWidth is
// the width of a dish's arm in metres. Choices of the detail study, by eye.
const (
	dishPlace = 0.8
	dishDown  = 0.35
	armWidth  = 0.05
)

// pipe is a downpipe from the top of the face to the ground, near one
// edge.
func (facade facade) pipe() []placedElement {
	size := facade.measured("pipe-size")
	return []placedElement{put("pipe", Box{size, facade.Height, size}, facade.Iron,
		maths.V(facade.Width-facade.measured("pipe-edge")-size, 0, 0))}
}

// cable is a cable run along the face under the windows of the first
// storey above the ground, with two drops to the ground floor.
func (facade facade) cable() []placedElement {
	if len(facade.Storeys) < 2 {
		return nil
	}
	size := facade.measured("cable-size")
	run := facade.floorAt(1) + cableUnder
	parts := []placedElement{put("cable", Box{facade.Width, size, size}, facade.Iron,
		maths.V(0, run, 0))}
	for _, share := range []float64{cableFirst, cableSecond} {
		parts = append(parts, put("cable", Box{size, run, size}, facade.Iron,
			maths.V(facade.Width*share, 0, 0)))
	}
	return parts
}

// cableUnder is how far over the first floor's level the cable runs, in
// metres, and cableFirst and cableSecond are where along the face its two
// drops are, as shares of its width. Choices of the detail study, by eye.
const (
	cableUnder  = 0.25
	cableFirst  = 0.3
	cableSecond = 0.55
)

// laundry is a line between the first two windows of the first storey
// above the ground, held out on a pole at each end, with washing hung on
// it.
func (facade facade) laundry() []placedElement {
	if len(facade.Storeys) < 2 || len(facade.Storeys[1]) < 2 {
		return nil
	}
	left, right := facade.Storeys[1][0], facade.Storeys[1][1]
	from := left.Left + left.Width + left.ShutterWidth
	to := right.Left - right.ShutterWidth
	reach, size := facade.measured("line-reach"), facade.measured("line-size")
	high := facade.floorAt(1) + left.Sill + left.Height*laundryUp
	parts := []placedElement{
		put("pole", Box{armWidth, armWidth, reach}, facade.Iron,
			maths.V(from, high, 0)),
		put("pole", Box{armWidth, armWidth, reach}, facade.Iron,
			maths.V(to-armWidth, high, 0)),
		put("line", Box{to - from, size, size}, facade.Iron,
			maths.V(from, high, reach)),
	}
	return append(parts, facade.washing(from, to, high, reach)...)
}

// laundryUp is how far up its window the line is hung, as a share of the
// window's height. A choice of the detail study, by eye.
const laundryUp = 0.9

// washing is the pieces hung on a laundry line, side by side, each a
// different cloth from the last.
func (facade facade) washing(from, to, high, reach float64) []placedElement {
	wide, drop := facade.measured("cloth-width"), facade.measured("cloth-drop")
	var parts []placedElement
	for index, along := 0, from+washingGap; along+wide < to-washingGap; index++ {
		parts = append(parts, put("washing", Box{wide, drop, washingThick},
			facade.Fixtures.Cloth[index%len(facade.Fixtures.Cloth)],
			maths.V(along, high-drop, reach)))
		along += wide + washingGap
	}
	return parts
}

// washingGap is the space between pieces of washing, and washingThick how
// thick a piece is, both in metres. Choices of the detail study, by eye.
const (
	washingGap   = 0.08
	washingThick = 0.015
)

// plinth is a stone band along the foot of the face, standing a little out
// of the wall, broken where a door is.
func (facade facade) plinth() []placedElement {
	high, proud := facade.measured("plinth-height"),
		facade.measured("plinth-proud")
	var parts []placedElement
	from := 0.0
	for _, opening := range facade.Storeys[0] {
		if !opening.Door {
			continue
		}
		parts = append(parts, facade.plinthTo(from, opening.Left, high,
			proud)...)
		from = opening.Left + opening.Width
	}
	return append(parts, facade.plinthTo(from, facade.Width, high, proud)...)
}

// plinthTo is the plinth between two places along the face, or nothing if
// they are the same.
func (facade facade) plinthTo(from, to, high, proud float64) []placedElement {
	if !(to > from) {
		return nil
	}
	return []placedElement{put("plinth", Box{to - from, high, proud},
		facade.Fixtures.Stone, maths.V(from, 0, 0))}
}

// sign is a board over the door of the ground floor, above its awning.
func (facade facade) sign() []placedElement {
	for _, opening := range facade.Storeys[0] {
		if !opening.Door {
			continue
		}
		wide := facade.measured("sign-width")
		return []placedElement{put("sign", Box{wide, facade.measured("sign-height"),
			facade.measured("sign-thick")}, facade.Fixtures.Board,
			maths.V(opening.Left+(opening.Width-wide)/half, opening.Sill+
				opening.Height+facade.measured("awning-rise")+
				facade.measured("sign-rise"), 0))}
	}
	return nil
}
