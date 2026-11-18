package skylight

// air is a column worked out for the walk: the planet's radius, the
// radius of its top sphere, its ground's albedo, its shells as layers
// split at the kept heights, and for each boundary between layers, from
// the ground up, the kept height it is, or unkept.
type air struct {
	radius, top, ground float64
	layers              []layer
	kept                []int
}

// a layer is a shell ready for the walk: the heights of its ends, their
// radii and the squares of those, its extinction per metre, the share
// of that it scatters, the share of the scattering that is gas, and its
// particles' asymmetry.
type layer struct {
	bottom, top                float64
	inner, outer               float64
	innerSquared, outerSquared float64
	extinction, albedo         float64
	gasShare, asymmetry        float64
}

// layered is the column worked out for the walk, each shell cut at the
// kept heights strictly inside it into layers of the same air. The
// heights have been checked to rise, so the cuts come in order.
func layered(column Column, heights []float64) air {
	last := column.Shells[len(column.Shells)-one]
	made := air{radius: column.Radius, top: column.Radius + last.Top,
		ground: column.GroundAlbedo}
	for _, shell := range column.Shells {
		bottom := shell.Bottom
		for _, height := range heights {
			if height > bottom && height < shell.Top {
				made.layers = append(made.layers,
					layerOf(column.Radius, shell, bottom, height))
				bottom = height
			}
		}
		made.layers = append(made.layers,
			layerOf(column.Radius, shell, bottom, shell.Top))
	}
	made.kept = keptOf(made.layers, heights)
	return made
}

// layerOf is the part of a shell from bottom to top, above a planet of
// radius, ready for the walk.
func layerOf(radius float64, shell Shell, bottom, top float64) layer {
	particles := shell.ParticleExtinction * shell.ParticleAlbedo
	scattering := shell.GasScattering + particles
	extinction := shell.GasScattering + shell.GasAbsorption +
		shell.ParticleExtinction
	inner, outer := radius+bottom, radius+top
	return layer{bottom: bottom, top: top, inner: inner, outer: outer,
		innerSquared: inner * inner, outerSquared: outer * outer,
		extinction: extinction, albedo: ratio(scattering, extinction),
		gasShare:  ratio(shell.GasScattering, scattering),
		asymmetry: shell.ParticleAsymmetry}
}

// ratio is part over whole, or nought when there is no whole: a layer
// with no extinction has no events, and one with no scattering ends
// each photon at its first.
func ratio(part, whole float64) float64 {
	if whole == none {
		return none
	}
	return part / whole
}

// keptOf is, for each boundary of the layers from the ground up, the
// index of the kept height it is, or unkept. Kept heights are boundaries
// exactly: each is a shell's end or a cut made at it.
func keptOf(layers []layer, heights []float64) []int {
	kept := make([]int, len(layers)+one)
	for boundary := range kept {
		kept[boundary] = unkept
		at := boundaryHeight(layers, boundary)
		for index, height := range heights {
			if height == at {
				kept[boundary] = index
			}
		}
	}
	return kept
}

// boundaryHeight is the height of a boundary between layers, counted
// from the ground, which is the first layer's bottom, to the top.
func boundaryHeight(layers []layer, boundary int) float64 {
	if boundary == len(layers) {
		return layers[boundary-one].top
	}
	return layers[boundary].bottom
}
