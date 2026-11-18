package manufacture

import "github.com/janearc/slayer/maths"

// cityItem is one plant module, its footprint and its material.
type cityItem struct {
	what       string
	wide, deep float64
	material   slot
}

// cityRow packs plant along a roof deck from left to right, skipping
// what does not fit what is left of the row.
type cityRow struct {
	building  *cityDetail
	from, end float64
	deck      float64
}

// roof finishes the highest stack: a rowhouse's gable and chimney, a
// tower's crown, or an apartment's plant.
func (building *cityDetail) roof(top cityStack) {
	design := building.design
	switch {
	case design.gable:
		seat := maths.Vec{Y: design.height - cityValue("roof-seat")}
		building.solid.add(cityGable{design.width, design.depth,
			cityValue("gable-rise")},
			maths.NewSubspace(seat, maths.Identity, spaceOne), cityRoof)
		building.chimney(top, seat.Y)
	case design.tower:
		building.crown(top)
	default:
		building.plant(top)
	}
}

// chimney stands on the gable's slope, halfway out, near the back, its
// top just below the ridge.
func (building *cityDetail) chimney(top cityStack, seat float64) {
	across, along := top.planes()
	building.share(building.design.wall, cityModule{what: cityChimneyModule},
		maths.Vec{X: across * spaceHalf, Y: seat,
			Z: cityValue("chimney-depth") - along})
}

// crown is a tower's plant room on its shaft: the quick level's crown
// and roof slab, louvre bands round it, and a mast on the shaft deck.
func (building *cityDetail) crown(shaft cityStack) {
	design := building.design
	base := float64(shaft.to) * design.storey()
	share, roof := cityValue("crown-width-share"),
		cityValue("crown-roof-share")
	building.box(design.width*share, design.height-base,
		design.depth*share, maths.Vec{Y: base}, design.wall)
	building.box(design.width*roof, cityValue("roof-thickness"),
		design.depth*roof,
		maths.Vec{Y: design.height - cityValue("roof-seat")}, cityRoof)
	building.louvres(base)
	across, along := shaft.planes()
	inside := cityWhole * cityValue("parapet-thickness")
	building.share(cityPlant, cityModule{what: cityMastModule},
		maths.Vec{X: across - inside,
			Y: base + cityValue("cornice-height"), Z: inside - along})
}

// louvres are evenly spaced bands round the crown's plant room.
func (building *cityDetail) louvres(base float64) {
	design := building.design
	count := int(cityValue("louvre-bands"))
	reach := cityWhole * cityValue("louvre-projection")
	share, height := cityValue("crown-width-share"),
		cityValue("louvre-height")
	space := (design.height - base) / float64(count+1)
	for band := 1; band <= count; band++ {
		building.box(design.width*share+reach, height,
			design.depth*share+reach, maths.Vec{
				Y: base + float64(band)*space - height*spaceHalf}, cityPlant)
	}
}

// plant fills a flat roof's deck inside its parapet: the overrun and
// tank in a back row and cooling units and a mast in front when the
// deck is deep enough for two rows, all in one row when it is not.
func (building *cityDetail) plant(top cityStack) {
	across, along := top.planes()
	inner, gap := cityValue("parapet-thickness"), cityValue("plant-gap")
	row := cityRow{building: building, from: inner - across + gap,
		end: across - inner - gap, deck: float64(top.to)*
			building.design.storey() + cityValue("cornice-height")}
	back, front := cityPlantItems()
	deep := cityWhole * (along - inner)
	cooler := cityValue("cooler-depth")
	behind := deep - cooler - cityPlantRowGaps*gap
	if behind < cityValue("overrun-depth") {
		row.place(append(back, front...), inner-along+gap, 1,
			deep-cityWhole*gap)
		return
	}
	row.place(back, inner-along+gap, 1, behind)
	row.place(front, along-inner-gap, -1, cooler)
}

// cityPlantItems are the back row's overrun and tank and the front
// row's cooling units and mast.
func cityPlantItems() ([]cityItem, []cityItem) {
	tank := cityWhole * cityValue("tank-radius")
	mast := cityWhole * cityValue("mast-radius")
	back := []cityItem{{cityOverrunModule, cityValue("overrun-width"),
		cityValue("overrun-depth"), cityRoof},
		{cityTankModule, tank, tank, cityPlant}}
	var front []cityItem
	for range int(cityValue("coolers")) {
		front = append(front, cityItem{cityCoolerModule,
			cityValue("cooler-width"), cityValue("cooler-depth"), cityPlant})
	}
	return back, append(front, cityItem{cityMastModule, mast, mast,
		cityPlant})
}

// place packs items from the row's start, each against an edge of the
// deck: the back edge when sign is one, the front when it is minus one.
// An item deeper than the row's depth, or wider than is left, is left out.
func (row cityRow) place(items []cityItem, edge, sign, depth float64) {
	cursor, gap := row.from, cityValue("plant-gap")
	for _, item := range items {
		if item.deep > depth || cursor+item.wide > row.end {
			continue
		}
		at := maths.Vec{X: cursor + item.wide*spaceHalf, Y: row.deck,
			Z: edge + sign*item.deep*spaceHalf}
		row.building.share(item.material, cityModule{what: item.what}, at)
		cursor += item.wide + gap
	}
}
