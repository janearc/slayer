package manufacture

import "fmt"

// The names the core tank is found by: the stack it is a section of, the
// section, and the kind it is made as.
const (
	slsStackName    = "sls-block-1"
	slsCoreSection  = "core stage"
	slsCoreTankKind = "sls-core-tank"
)

// coreTankOf is the Space Launch System's core stage tank alone, as a
// stack of the one section: the very section the sls-block-1 row lists, so
// its width, its length, its material and its paint are that row's and
// cannot drift from it, with no booster beside it and nothing above it.
// It is refused if the rows list no such stack or the stack no such
// section.
func coreTankOf(rows []stackRow) (stackRow, error) {
	for _, row := range rows {
		if row.Name != slsStackName {
			continue
		}
		for _, section := range row.Sections {
			if section.Part == slsCoreSection {
				return stackRow{Name: slsCoreTankKind,
					Says: "The core stage of " + row.Name + ", alone: " +
						"its tank, as that stack makes it.",
					Sections: []stackSection{section}}, nil
			}
		}
	}
	return stackRow{}, fmt.Errorf("manufacture: stacks: no stack %q with a "+
		"section %q, so there is no core tank", slsStackName, slsCoreSection)
}
