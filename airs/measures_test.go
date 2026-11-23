package airs

// gasDepth is the column's Rayleigh optical depth, straight up.
func (column Column) gasDepth() float64 {
	depth := 0.0
	for _, layer := range column.Layers {
		depth += layer.GasScattering * (layer.Top - layer.Bottom)
	}
	return depth
}

// particleDepth is the column's particles' optical depth, straight up.
func (column Column) particleDepth() float64 {
	depth := 0.0
	for _, layer := range column.Layers {
		depth += layer.ParticleExtinction * (layer.Top - layer.Bottom)
	}
	return depth
}

// meanAlbedo is the particles' single-scattering albedo averaged over the
// column, the extinction a shell holds being the weight; nought where
// there are no particles.
func (column Column) meanAlbedo() float64 {
	extinction, scattering := 0.0, 0.0
	for _, layer := range column.Layers {
		held := layer.ParticleExtinction * (layer.Top - layer.Bottom)
		extinction += held
		scattering += held * layer.ParticleAlbedo
	}
	if extinction == 0 {
		return 0
	}
	return scattering / extinction
}

// meanAsymmetry is the particles' asymmetry averaged over the column, the
// scattering a shell holds being the weight; nought where there is none.
func (column Column) meanAsymmetry() float64 {
	scattering, product := 0.0, 0.0
	for _, layer := range column.Layers {
		held := layer.ParticleExtinction * layer.ParticleAlbedo *
			(layer.Top - layer.Bottom)
		scattering += held
		product += held * layer.ParticleAsymmetry
	}
	if scattering == 0 {
		return 0
	}
	return product / scattering
}

// absorptionDepth is the column's gas absorption optical depth, straight
// up: Earth's ozone, where Venus has none.
func (column Column) absorptionDepth() float64 {
	depth := 0.0
	for _, layer := range column.Layers {
		depth += layer.GasAbsorption * (layer.Top - layer.Bottom)
	}
	return depth
}
