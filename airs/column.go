package airs

import (
	"math"
	"slices"
)

// Layer is a shell of air between two heights, with what the solver
// takes of it, every one a figure a metre, at one wavelength.
type Layer struct {
	// Bottom and Top are the shell's heights above the ground, in
	// metres.
	Bottom, Top float64
	// GasScattering is the gas's Rayleigh scattering a metre.
	GasScattering float64
	// GasAbsorption is the gas's absorption a metre.
	GasAbsorption float64
	// ParticleExtinction is the particles' extinction a metre.
	ParticleExtinction float64
	// ParticleAlbedo is the particles' single-scattering albedo.
	ParticleAlbedo float64
	// ParticleAsymmetry is the particles' asymmetry parameter, g.
	ParticleAsymmetry float64
}

// Column is a planet's air at one wavelength: the planet's radius in
// metres, the shells from the ground up with no gap between them, and the
// ground's albedo.
type Column struct {
	Radius       float64
	Layers       []Layer
	GroundAlbedo float64
}

// GasDepth is the column's Rayleigh optical depth, straight up.
func (column Column) GasDepth() float64 {
	depth := 0.0
	for _, layer := range column.Layers {
		depth += layer.GasScattering * (layer.Top - layer.Bottom)
	}
	return depth
}

// ParticleDepth is the column's particles' optical depth, straight up.
func (column Column) ParticleDepth() float64 {
	depth := 0.0
	for _, layer := range column.Layers {
		depth += layer.ParticleExtinction * (layer.Top - layer.Bottom)
	}
	return depth
}

// MeanAlbedo is the particles' single-scattering albedo averaged over the
// column, the extinction a shell holds being the weight; nought where
// there are no particles.
func (column Column) MeanAlbedo() float64 {
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

// MeanAsymmetry is the particles' asymmetry averaged over the column, the
// scattering a shell holds being the weight; nought where there is none.
func (column Column) MeanAsymmetry() float64 {
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

// Venus is Venus's air at a wavelength in nanometres: the shells from
// the ground to the top of the upper haze, and the ground's albedo. It
// refuses a wavelength outside what carbon dioxide's refractive index
// is known over, and a file of data that could not be read.
func Venus(nanometres float64) (Column, error) {
	if venusError != nil {
		return Column{}, venusError
	}
	return venusRead.column(nanometres)
}

// edges are the heights the air is cut at, from the ground to the top:
// every height a layer or an absorber begins or ends at, and between
// them as many equal cuts as keep each shell no thicker than the data's.
func (read data) edges() []float64 {
	marks := []float64{0, read.Gas.Top.Value}
	for _, layer := range read.Layers {
		marks = append(marks, layer.Bottom, layer.Top)
	}
	for _, absorber := range read.Absorbers {
		marks = append(marks, absorber.Bottom, absorber.Top)
	}
	return cuts(marks, read.Gas.Shell.Value)
}

// cuts are the heights an air is cut at, given the heights that must be
// among them, in any order and repeated or not: the marks, and between
// each two as many equal cuts as keep every shell no thicker than the
// shell given. The first is the lowest mark and the last the highest.
func cuts(marks []float64, shell float64) []float64 {
	marks = slices.Clone(marks)
	slices.Sort(marks)
	marks = slices.Compact(marks)
	edges := []float64{marks[0]}
	for index := 1; index < len(marks); index++ {
		pieces := int(math.Ceil((marks[index] - marks[index-1]) / shell))
		for piece := 1; piece <= pieces; piece++ {
			fraction := float64(piece) / float64(pieces)
			edges = append(edges, marks[index-1]*(1-fraction)+marks[index]*fraction)
		}
	}
	return edges
}

// column is the air at a wavelength in nanometres.
func (read data) column(nanometres float64) (Column, error) {
	cross, err := read.Gas.meanCross(nanometres)
	if err != nil {
		return Column{}, err
	}
	clouds := make([]optics, len(read.Layers))
	for index, layer := range read.Layers {
		if clouds[index], err = read.particles(layer, nanometres); err != nil {
			return Column{}, err
		}
	}
	column := Column{Radius: read.Planet.Radius.Value,
		GroundAlbedo: read.Ground.albedo(nanometres)}
	edges := read.edges()
	for index := 1; index < len(edges); index++ {
		column.Layers = append(column.Layers, read.shell(edges[index-1],
			edges[index], cross, clouds, nanometres))
	}
	return column, nil
}

// shell is the air between two heights: the gas by its cross section,
// the particles of the layer the shell's middle is in, and what any
// absorber there takes at this wavelength.
func (read data) shell(bottom, top, cross float64, clouds []optics,
	nanometres float64) Layer {
	middle := (bottom + top) / 2
	var particles optics
	for index, layer := range read.Layers {
		if layer.Bottom < middle && middle < layer.Top {
			particles = clouds[index]
		}
	}
	for _, absorber := range read.Absorbers {
		if absorber.Bottom < middle && middle < absorber.Top &&
			nanometres < absorber.Below {
			particles = particles.absorbed(absorber.Absorption.Value)
		}
	}
	return Layer{Bottom: bottom, Top: top,
		GasScattering:      read.Gas.scattering(bottom, top, cross),
		GasAbsorption:      read.Gas.Absorption.Value,
		ParticleExtinction: particles.extinction,
		ParticleAlbedo:     particles.albedo, ParticleAsymmetry: particles.asymmetry}
}

// albedo is the ground's albedo at a wavelength in nanometres: linear
// between the table's points and held flat past its ends.
func (ground groundData) albedo(nanometres float64) float64 {
	return ground.at(nanometres)
}

// at is a table's figure at a wavelength in nanometres: linear between
// the table's points and held flat past its ends.
func (table tableData) at(nanometres float64) float64 {
	index, fraction := table.locate(nanometres)
	points := table.Points
	return points[index][1] + fraction*(points[index+1][1]-points[index][1])
}

// locate is the segment of the table a wavelength is in, as the index of
// the point it begins at, and how far along it the wavelength is, from
// nought to one. Past the table's ends it is the first or last segment's
// own end, so that a figure beside it is held flat.
func (table tableData) locate(nanometres float64) (int, float64) {
	points := table.Points
	last := len(points) - 1
	if nanometres <= points[0][0] {
		return 0, 0
	}
	for index := 1; index <= last; index++ {
		if nanometres <= points[index][0] {
			return index - 1, (nanometres - points[index-1][0]) /
				(points[index][0] - points[index-1][0])
		}
	}
	return last - 1, 1
}

// AbsorptionDepth is the column's gas absorption optical depth, straight
// up: Earth's ozone, where Venus has none.
func (column Column) AbsorptionDepth() float64 {
	depth := 0.0
	for _, layer := range column.Layers {
		depth += layer.GasAbsorption * (layer.Top - layer.Bottom)
	}
	return depth
}
