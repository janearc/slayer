package manufacture

import (
	"fmt"
	"slices"
	"sort"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Street-level retail: the shopfronts at the foot of a city street, a
// ramen counter, a dive bar, a shuttered unit, a pair of vending machines
// on the pavement, a convenience store, an izakaya, an arcade, a pharmacy,
// each lit from inside so that its light spills onto the pavement.
//
// A unit is one of the types in retail-units.yaml; a frontage is a row
// of them
// drawn from a seed, with no two of a kind side by side, each a whole
// number of the city's bays wide, as many as its type says the seed may
// draw, and as tall as the city's ground storey (city_detail.go), on its
// wall plane.
//
// A set stands a frontage at the foot of a row of buildings,
// facing the street; retail_ground.go stands a podium of them on a
// building. Detail by level: hull is a lit opening, quick has the
// counter, door, awning and sign, demo the stools, bowls and the hardware.

// frontageRow is the kind frontage, a row of units drawn from the seed,
// or, if it names a type, a row of one unit of that type: frontage-ramen,
// or, if it is one, one unit of a type the seed draws: frontage-one.
type frontageRow struct {
	only string
	one  bool
}

// retailRows are the kinds of street-level retail by name: the frontage,
// one unit of a type the seed draws, a single ramen counter, a single
// dive bar, and a building on a podium of shops.
func retailRows() map[string]maker {
	return map[string]maker{"frontage": frontageRow{},
		"frontage-one":   frontageRow{one: true},
		"frontage-ramen": frontageRow{only: "ramen"},
		"frontage-dive":  frontageRow{only: "dive"},
		"retail-block":   retailBlockRow{}}
}

// made is the frontage at the quick detail level.
func (row frontageRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the frontage at the named detail level. It returns errGood
// when the seed is not made, and refuses a level there is none of.
func (row frontageRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	builder, err := row.built(from, detail)
	if err != nil {
		return good{}, err
	}
	return builder.good()
}

// built is the frontage's builder at a detail level, with every unit in
// it.
func (row frontageRow) built(from seed, detail string) (*retailBuilder,
	error) {
	level := retailLevel(detail)
	if level < 0 {
		return nil, fmt.Errorf("manufacture: frontage has no %s "+
			"representation; choose %v", detail, retailLevels)
	}
	if retailBookError != nil {
		return nil, retailBookError
	}
	names, err := row.units(from)
	if err != nil {
		return nil, err
	}
	builder := &retailBuilder{book: retailBookRead, level: level,
		skins: map[string]surface{}}
	for index, name := range names {
		builder.unit(index, name, from)
	}
	return builder, nil
}

// units are the unit types of a frontage, left to right: the one type it
// is of, the one type the seed draws, or a run drawn from the seed.
func (row frontageRow) units(from seed) ([]string, error) {
	if retailBookError != nil {
		return nil, retailBookError
	}
	if row.one {
		return []string{retailBookRead.pick(retailBookRead.unitNames(), "",
			from.draw(retailPickDraw, 0, 0))}, nil
	}
	if row.only == "" {
		return retailBookRead.run(from), nil
	}
	if _, known := retailBookRead.Units[row.only]; !known {
		return nil, fmt.Errorf("manufacture: no unit type %q; there are %v",
			row.only, retailBookRead.unitNames())
	}
	return []string{row.only}, nil
}

// run is the unit types of a run drawn from the seed: each from the
// types by their weights, but never the type that stands beside it.
func (book retailBook) run(from seed) []string {
	names := book.unitNames()
	run := make([]string, 0, book.Run.Units)
	for index := range book.Run.Units {
		beside := ""
		if index > 0 {
			beside = run[index-1]
		}
		share := from.draw(retailPickDraw, index, 0)
		run = append(run, book.pick(names, beside, share))
	}
	return run
}

// pick is the type a share of the way through the weights of the types
// but one, which it never picks.
func (book retailBook) pick(names []string, beside string,
	share float64) string {
	total := 0.0
	for _, name := range names {
		if name != beside {
			total += book.Units[name].Weight
		}
	}
	reach, chosen := share*total, ""
	for _, name := range names {
		if name == beside {
			continue
		}
		chosen = name
		reach -= book.Units[name].Weight
		if reach < 0 {
			break
		}
	}
	return chosen
}

// FrontageUnits is the unit types of the frontage a seed draws, left to
// right: ramen, dive, shuttered, vending, konbini, izakaya, arcade and
// pharmacy.
func FrontageUnits(number uint64) ([]string, error) {
	return frontageRow{}.units(sow(number))
}

// FrontageOne is the unit type of the single unit frontage-one makes of a
// seed.
func FrontageOne(number uint64) (string, error) {
	names, err := frontageRow{one: true}.units(sow(number))
	if err != nil {
		return "", err
	}
	return names[0], nil
}

// FrontageSlots is the places the frontage a seed draws keeps for a sign
// over each door, left to right, in the frontage's frame.
func FrontageSlots(number uint64) ([]FrontageSlot, error) {
	builder, err := frontageRow{}.built(sow(number), "hull")
	if err != nil {
		return nil, err
	}
	return builder.slots, nil
}

// unit draws one unit: what is its own, its shell, its kits, and its
// sign slot, and moves along to where the next one starts.
func (builder *retailBuilder) unit(index int, name string, from seed) {
	unit := builder.book.Units[name]
	builder.plan(index, unit, from)
	builder.shell(unit)
	for number, kit := range builder.kits(index, unit, from) {
		builder.dress(index*retailMostKits+number, kit, from)
	}
	builder.slot(name, unit)
	builder.left += builder.width
}

// lamp puts a unit's light where it stands in the frontage.
func (builder *retailBuilder) lamp(one retailLight) {
	along := one.At[0] + retailSlide(one.Along, builder.extra)
	across, _ := builder.across(along, along)
	builder.lamps = append(builder.lamps, retailLamp{heat: one.Heat,
		at: maths.V(across, one.At[1], one.At[2]), luminance: one.Luminance})
}

// slot notes the place a unit keeps for a sign, if it keeps one. The
// place is as wide as the unit has grown.
func (builder *retailBuilder) slot(name string, unit retailUnit) {
	if unit.Sign == nil {
		return
	}
	left, right := builder.across(unit.Sign[0], unit.Sign[2]+builder.extra)
	builder.slots = append(builder.slots, FrontageSlot{Unit: name,
		Name: "sign over the door", Width: right - left,
		Height: unit.Sign[3] - unit.Sign[1], Place: maths.NewSubspace(
			maths.V(left, unit.Sign[1], 0), maths.Identity, spaceOne)})
}

// opening is a unit's opening as it stands in the unit the seed drew: the
// data's, stretched with the unit if it stretches.
func (builder *retailBuilder) opening(unit retailUnit) []float64 {
	open := slices.Clone(unit.Opening)
	if unit.Stretch && len(open) == retailRectNumbers {
		open[2] += builder.extra
	}
	return open
}

// shell draws a unit's front wall, and behind an opening in it the floor,
// the ceiling and the walls of a room. A unit without an opening is a
// plain front. Each box stands past its edges into its neighbours. In a
// podium deeper than the room, plain wall fills the rest.
func (builder *retailBuilder) shell(unit retailUnit) {
	shell, wide := builder.book.Shell, builder.width+retailBleed
	_, tall := unit.size()
	front, open := -shell.Wall, builder.opening(unit)
	if len(open) == 0 {
		builder.solid(unit.Wall, maths.V(-retailBleed, 0, front),
			maths.V(wide, tall, 0))
		builder.filling(front*spaceHalf, tall)
		return
	}
	builder.front(unit, open, front, tall)
	back := -unit.Depth
	builder.solid(unit.Floor, maths.V(-retailBleed, 0, back),
		maths.V(wide, shell.Floor, -retailRecess))
	builder.solid(unit.Ceiling, maths.V(-retailBleed, tall-shell.Slab, back),
		maths.V(wide, tall, front+retailBleed))
	room := maths.V(-retailBleed, shell.Floor, back)
	builder.solid(unit.Inner, room, maths.V(wide, tall-shell.Slab,
		back+shell.Partition))
	for _, side := range []float64{-retailBleed, builder.width -
		shell.Partition} {
		builder.solid(unit.Sides, maths.V(side, shell.Floor,
			back+shell.Partition), maths.V(side+shell.Partition+retailBleed,
			tall-shell.Slab, front+retailBleed))
	}
	builder.filling(back+shell.Partition*spaceHalf, tall)
}

// filling is plain wall behind a unit, from the podium's back forward to
// a place inside the unit's own back wall, so that a podium of units of
// different depths has one back. It stands in the unit's wall, which is
// another material, so that no two faces lie in one plane. It is nothing
// for a frontage that stands alone.
func (builder *retailBuilder) filling(from, tall float64) {
	if builder.podium <= 0 {
		return
	}
	builder.solid(builder.book.Grounds.Filler, maths.V(-retailBleed, 0,
		-builder.podium), maths.V(builder.width+retailBleed,
		tall-retailBleed, from))
}

// front draws the front wall round an opening: a pier at each end, and a
// sill under it unless it opens down to the floor, and a lintel over it,
// the sill and the lintel standing into the piers.
func (builder *retailBuilder) front(unit retailUnit, open []float64, front,
	tall float64) {
	lap := builder.book.Shell.Partition
	wide := builder.width + retailBleed
	builder.solid(unit.Wall, maths.V(-retailBleed, 0, front),
		maths.V(open[0], tall, 0))
	builder.solid(unit.Wall, maths.V(open[2], 0, front), maths.V(wide, tall, 0))
	if open[1] > builder.book.Shell.Floor {
		builder.solid(unit.Wall, maths.V(open[0]-lap, 0, front),
			maths.V(open[2]+lap, open[1], 0))
	}
	builder.solid(unit.Wall, maths.V(open[0]-lap, open[3], front),
		maths.V(open[2]+lap, tall, 0))
}

// stickers draws each group of stickers, which the seed puts in the group's
// rectangle of the wall plane, wider as the unit is, each in a look it
// draws. The stream is the kit's place among the unit's kits in the run.
func (builder *retailBuilder) stickers(stream int, groups []retailScatter,
	from seed) {
	for which, group := range groups {
		if !builder.reaches(group.From) {
			continue
		}
		place := stream*retailMostStreams + which
		for count := range group.Count {
			draw := func(number int) float64 {
				return from.draw(retailStickerDraw, place,
					count*retailStickerDraws+number)
			}
			wide, tall := group.Size[0], group.Size[1]
			reach := group.Region[2] + builder.extra
			x := group.Region[0] + (reach-group.Region[0]-wide)*draw(0)
			y := group.Region[1] + (group.Region[3]-group.Region[1]-tall)*draw(1)
			look := group.Materials[int(draw(2)*float64(len(group.Materials)))]
			builder.solid(look, maths.V(x, y, group.Z),
				maths.V(x+wide, y+tall, group.Z+retailLeaf))
		}
	}
}

// props draws the props a kit may have on the pavement before the unit,
// each as likely as it says and where the seed puts it in its ranges,
// which are wider as the unit is.
func (builder *retailBuilder) props(stream int, uses []retailPropUse,
	from seed) {
	for which, use := range uses {
		place := stream*retailMostStreams + which
		if !builder.reaches(use.From) ||
			from.draw(retailPropDraw, place, 0) >= use.Chance {
			continue
		}
		reach := use.X[1] + builder.extra
		at := maths.V(use.X[0]+(reach-use.X[0])*
			from.draw(retailPropDraw, place, 1), 0,
			use.Z[0]+(use.Z[1]-use.Z[0])*from.draw(retailPropDraw, place, 2))
		for _, piece := range builder.book.Props[use.Prop].Parts {
			builder.part(piece, at, 0)
		}
	}
}

// good is the frontage as a made thing: a part for each material, each
// dressed in its look, and a light for each lamp.
func (builder *retailBuilder) good() (good, error) {
	made := good{dressed: looks{}}
	slots := builder.book.slots()
	names := make([]string, 0, len(builder.skins))
	for name := range builder.skins {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		made.parts = append(made.parts, part{shape: retailSkin{
			builder.skins[name]}, space: maths.Local(), of: slots[name]})
		made.dressed[slots[name]] = builder.book.look(name)
	}
	for _, one := range builder.lamps {
		shining, err := newRelativeLight(measure.Kelvin(one.heat),
			one.luminance)
		if err != nil {
			return good{}, err
		}
		made = made.lit(shining, one.at, maths.Up, measure.Metres(lampBeyond))
	}
	return made, nil
}
