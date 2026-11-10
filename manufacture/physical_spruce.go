package manufacture

// physicalSpruce keeps the legacy whorled architecture as a reference while
// named factory specimens use fixed-size, recursively branched needle shoots.
type physicalSpruce struct{ legacy conifer }

// spruceShootStyle is the third physical shoot family, after redwood/sequoia.
const spruceShootStyle = 3

// made supplies the named factory's ordinary quick specimen.
func (row physicalSpruce) made(from seed) (good, error) { return row.madeAt(from, "quick") }

// madeAt retains seeded major limbs and changes only their foliage surfaces.
func (row physicalSpruce) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	made, err := row.legacy.scaffold(from)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	made = row.terminals(made)
	return recursiveFoliage(made, from, spruceShootStyle, detail)
}

// terminals identifies the leaf-bearing axes of a spruce without branchlets.
// Norway already marks its hanging branchlet foliage and leaves major sleeves
// unmarked; ordinary spruce carries foliage on the main branch segments.
func (row physicalSpruce) terminals(made good) good {
	if row.legacy.grows.shoots != nil {
		return made
	}
	for index := range made.parts {
		if made.parts[index].of == leaf {
			made.parts[index].terminal = true
		}
	}
	return made
}
