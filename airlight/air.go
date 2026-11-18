package airlight

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/janearc/slayer/airs"
	"github.com/janearc/slayer/skylight"
)

// Air is a planet's air as the renderer asks it: its name, its column
// at a wavelength as the solver takes it, and the sun at its top.
type Air interface {
	// Name is the air's name, as a set file says it.
	Name() string
	// Column is the air at a wavelength in nanometres, its shells from
	// the planet's mean radius up, and its ground's albedo.
	Column(nanometres float64) (skylight.Column, error)
	// Sun is the sun's irradiance at the top of the air at a wavelength
	// in nanometres, in watts a square metre a nanometre: the mean over
	// the band the wavelength stands for.
	Sun(nanometres float64) (float64, error)
}

// Known is an air a set may name, how many photons each of its
// wavelengths is walked with, and its field's bins: of the view's
// cosine from up, an even count so the horizon falls between two, and
// of the azimuth from the sun.
type Known struct {
	Air             Air
	Photons         int
	Views, Azimuths int
}

// an entry is how an air of package airs is had: its column at a
// wavelength, the sun's irradiance over a band at its distance, the
// photons each wavelength is walked with, and its view and azimuth bins.
type entry struct {
	column          func(nanometres float64) (airs.Column, error)
	sunAt           func(sun airs.Sun, low, high float64) (float64, error)
	photons         int
	views, azimuths int
}

// named is every air a set may name. Earth's sun is at one astronomical
// unit, which is the spectrum's own, airs.Sun.Band; Venus's is nearer.
var named = map[string]entry{
	"earth": {airs.Earth, airs.Sun.Band, earthPhotons, earthViews,
		earthAzimuths},
	"venus": {airs.Venus, airs.Sun.AtVenus, venusPhotons, venusViews,
		venusAzimuths},
}

// Names is every air a set may name, in order.
func Names() []string { return slices.Sorted(maps.Keys(named)) }

// Named is the air of a name lit by the sun's spectrum, as airs.ReadSun
// read it from the shelf, or a refusal naming the airs there are.
func Named(name string, sun airs.Sun) (Known, error) {
	made, has := named[name]
	if !has {
		return Known{}, fmt.Errorf("airlight: no air is named %q; the "+
			"airs are %s", name, strings.Join(Names(), ", "))
	}
	return Known{Air: fromAirs{name: name, column: made.column, sun: sun,
		sunAt: made.sunAt}, Photons: made.photons, Views: made.views,
		Azimuths: made.azimuths}, nil
}

// fromAirs is an air of package airs, its layers as the solver's shells,
// lit by the sun's spectrum at its distance.
type fromAirs struct {
	name   string
	column func(nanometres float64) (airs.Column, error)
	sun    airs.Sun
	sunAt  func(sun airs.Sun, low, high float64) (float64, error)
}

// Name is the air's name.
func (air fromAirs) Name() string { return air.name }

// Column is the air of package airs at a wavelength, as the solver's.
func (air fromAirs) Column(nanometres float64) (skylight.Column, error) {
	column, err := air.column(nanometres)
	if err != nil {
		return skylight.Column{}, err
	}
	return mapped(column), nil
}

// Sun is the sun's irradiance at the air's top, the mean over the band
// of everyNanometres about the wavelength, ends included.
func (air fromAirs) Sun(nanometres float64) (float64, error) {
	reach := float64(everyNanometres) / two
	return air.sunAt(air.sun, nanometres-reach, nanometres+reach)
}

// mapped is a column of package airs as the solver's, field for field:
// the radius, each layer as a shell, and the ground's albedo.
//
// The two packages' layers hold the same seven numbers by the same names,
// and TestTheShellsHoldWhatTheLayersHold holds them to it: a number one
// has and the other has not would not map, and fails that test.
func mapped(column airs.Column) skylight.Column {
	shells := make([]skylight.Shell, len(column.Layers))
	for index, layer := range column.Layers {
		shells[index] = skylight.Shell{Bottom: layer.Bottom, Top: layer.Top,
			GasScattering:      layer.GasScattering,
			GasAbsorption:      layer.GasAbsorption,
			ParticleExtinction: layer.ParticleExtinction,
			ParticleAlbedo:     layer.ParticleAlbedo,
			ParticleAsymmetry:  layer.ParticleAsymmetry}
	}
	return skylight.Column{Radius: column.Radius, Shells: shells,
		GroundAlbedo: column.GroundAlbedo}
}

// grounded is the column seen from a ground so many metres above its
// own: the shells wholly below it gone, the one it stands in cut there,
// every height counted from it, and the radius out to it. A ground
// below the column's lowest shell stretches that shell down to it.
//
// So the solver's ground is the set's ground. A camera a metre or two
// above it is well inside the field's first shell, and the light there
// is the ground's within the field's noise.
func grounded(column skylight.Column, ground float64) (skylight.Column,
	error) {
	if len(column.Shells) == none {
		return column, skylight.ErrShells
	}
	top := column.Shells[len(column.Shells)-one].Top
	if !(ground < top) {
		return column, fmt.Errorf("airlight: the ground at %g m is not "+
			"below the air's top at %g m", ground, top)
	}
	var kept []skylight.Shell
	for _, shell := range column.Shells {
		if shell.Top <= ground {
			continue
		}
		if len(kept) == none {
			shell.Bottom = ground
		}
		shell.Bottom, shell.Top = shell.Bottom-ground, shell.Top-ground
		kept = append(kept, shell)
	}
	column.Shells, column.Radius = kept, column.Radius+ground
	return column, nil
}
