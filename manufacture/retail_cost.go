package manufacture

// RetailCost is what a frontage costs to draw: the closed pieces it is
// made of, the triangles in them, and the lamps it carries.
type RetailCost struct {
	Solids, Triangles, Lamps int
}

// RetailUnits are the unit types of street-level retail, in the order
// their names sort.
func RetailUnits() ([]string, error) {
	if retailBookError != nil {
		return nil, retailBookError
	}
	return retailBookRead.unitNames(), nil
}

// RetailUnitWidth is how wide the unit of a type a seed draws is, in
// metres: its bays as the seed draws them, whole bays of the city's.
func RetailUnitWidth(unit string, number uint64) (float64, error) {
	builder, err := frontageRow{only: unit}.built(sow(number), "hull")
	if err != nil {
		return 0, err
	}
	return builder.left, nil
}

// RetailUnitCost is the cost of one unit of a type drawn from a seed at a
// detail level: its shell, its parts, its props and its stickers.
func RetailUnitCost(unit string, number uint64, detail string) (RetailCost,
	error) {
	builder, err := frontageRow{only: unit}.built(sow(number), detail)
	if err != nil {
		return RetailCost{}, err
	}
	cost := RetailCost{Solids: builder.solids, Lamps: len(builder.lamps)}
	for _, skin := range builder.skins {
		cost.Triangles += len(skin)
	}
	return cost, nil
}
