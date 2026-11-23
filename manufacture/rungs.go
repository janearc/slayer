package manufacture

// rungMaker is a maker that says which complexities it makes, so a caller
// can pick among them without making one to find out.
type rungMaker interface {
	rungs() []string
}

// RungsOf is the complexities a kind makes, coarse to fine, said without
// making anything. A kind that does not say is taken to make quick alone,
// which every kind makes: asking for less than a kind says is safe, and
// asking for more is refused by MakeWithComplexity.
func RungsOf(row Kind) []string {
	if said, says := row.how.(rungMaker); says {
		return said.rungs()
	}
	return []string{"quick"}
}

// rungs of a plant are all four: its bark and foliage cells are made at
// each, and a hull is its crown as one box.
func (seasonalPlant) rungs() []string { return Complexities() }

// rungs of a whorled tree are all four, as madeAt gives them.
func (conifer) rungs() []string { return Complexities() }

// rungs of a spruce are all four: recursive foliage takes the hull as
// quick shoots under a hull's crown.
func (physicalSpruce) rungs() []string { return Complexities() }
