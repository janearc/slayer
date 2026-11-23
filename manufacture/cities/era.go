package cities

// A street is built over decades, and each lot is rebuilt when its house
// is: so a culture may say the eras its lots come from, and draw one for
// each bay before anything else of it. An era's own elements, named for the
// element and a hyphen and the era, replace the culture's for that bay.

// The names of the elements that draw an era.
const (
	// eraElement lists the eras, as options. The weights it gives are
	// not used: the share of each era is its own element, drawn for a
	// district, so one district is mostly of one era and another of
	// another.
	eraElement = "era"
	// eraRollElement is the roll, from nought to one, that picks a bay's
	// era from the shares.
	eraRollElement = "era-roll"
	// eraSharePrefix begins the name of the element of an era's share.
	eraSharePrefix = "era-share-"
)

// eraAt is the era of the bay at a place, or nothing for a culture that
// says no eras. The roll is drawn for the bay, so neighbours may differ.
func (drawing *drawer) eraAt(at place) string {
	if !drawing.has(eraElement) {
		return ""
	}
	eras := drawing.options(eraElement)
	shares := make([]float64, len(eras))
	var total float64
	for index, one := range eras {
		shares[index] = drawing.number(eraSharePrefix+one.Name, place{})
		total += shares[index]
	}
	left := drawing.number(eraRollElement, at) * total
	for index, one := range eras {
		if left -= shares[index]; left < 0 {
			return one.Name
		}
	}
	return eras[len(eras)-1].Name
}

// eraColour is the colour of a bay's wall: the row's own, or, where the
// bay's era has a palette of its own, one drawn from that.
func (plan rowPlan) eraColour(drawing *drawer, at place) option {
	if drawing.era != "" && drawing.has("palette-"+drawing.era) {
		return drawing.pick("palette", drawing.options("palette"), at)
	}
	return plan.colours[at.Bay]
}
