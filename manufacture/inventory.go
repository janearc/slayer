package manufacture

// PartCount reports constructed components before material batching. Compound
// mirror forms count each backed panel; arrays count every placed component.
// Tessellation, material slots and prototype reuse do not add parts. This is
// an inventory of the model, not certification of its engineering accuracy.
func (made good) PartCount() int {
	count := len(made.native)
	for _, number := range made.partCounts() {
		count += number
	}
	return count
}

// partCounts retains component inventories separately from rendering batches.
func (made good) partCounts() map[slot]int {
	counts := map[slot]int{}
	for _, piece := range made.parts {
		if made.groups[piece.of] != nil {
			continue
		}
		count := 1
		if compound, ok := piece.shape.(interface{ componentCount() int }); ok {
			count = compound.componentCount()
		}
		counts[piece.of] += count
	}
	for material, group := range made.groups {
		if count, known := made.groupParts[material]; known {
			counts[material] = count
		} else {
			counts[material] = len(group.Members())
		}
	}
	return counts
}

// componentCount is the number of closed panels emitted by the annular form.
func (primary annularPrimary) componentCount() int {
	return len(annularPanels(primary.count, primary.hole))
}

// componentCount counts both flat and spherical OWL panels independently.
func (mirror owlSegmented) componentCount() int {
	return len(annularPanels(mirror.count, mirror.hole))
}

// componentCount sums hexagonal rings of six, twelve, eighteen panels, etc.
func (primary hexagonalPrimary) componentCount() int {
	return 3 * primary.rings * (primary.rings + 1)
}

// componentCount distinguishes the single dish from its separated tile form.
func (mirror mirrorDish) componentCount() int {
	if !mirror.segmented {
		return 1
	}
	return len(mirror.panelCentres())
}
