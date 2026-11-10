package set

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/plume"
	yaml "gopkg.in/yaml.v3"
)

// Fire returns a copied volumetric recipe, or false for another recipe kind.
func (recipe Recipe) Fire() (manufacture.BurnSpec, bool) {
	if recipe.burn == nil {
		return manufacture.BurnSpec{}, false
	}
	spec := *recipe.burn
	if spec.Sparks != nil {
		copied := *spec.Sparks
		spec.Sparks = &copied
	}
	if spec.Smoke != nil {
		copied := *spec.Smoke
		spec.Smoke = &copied
	}
	return spec, true
}

// fireRecipe checks the model after reading its unit-bearing inputs.
func (reading reader) fireRecipe(name, block *yaml.Node) error {
	spec := manufacture.BurnSpec{}
	cells := 3.0
	err := reading.keys(block, map[string]step{
		"kind":        reading.only("fire"),
		"flame":       reading.fireFlame(&spec.Flame),
		"layout":      reading.fireLayout(&spec.Layout),
		"smoke":       reading.fireSmoke(&spec),
		"sparks":      reading.fireSparks(&spec),
		"light-cells": reading.whole(&cells, 3, 8),
	}, nil)
	if err != nil {
		return err
	}
	if err = reading.all(block, "fire recipe", "kind",
		"flame"); err != nil {
		return err
	}
	spec.LightCells = int(cells)
	if spec.Sparks != nil {
		spec.Sparks.Disk = spec.FuelRadius()
	}
	if spec.Smoke != nil {
		spec.Smoke.Base = 2 * spec.FuelRadius().InMetres()
	}
	if _, err = manufacture.Burn(spec); err != nil {
		return reading.refuse(block, "%v", err)
	}
	reading.set.recipes[name.Value] = Recipe{kind: "fire", burn: &spec}
	return nil
}

// fireFlame requires the geometry, optical depth and temperature range.
func (reading reader) fireFlame(fire *flame.Fire) step {
	return func(_, node *yaml.Node) error {
		var hot, edge float64
		var profile string
		var composition *flame.CompositionSpec
		err := reading.keys(node, map[string]step{
			"composition":         reading.fireComposition(&composition),
			"intermittency-start": reading.number(&fire.Intermittency, 0.001, 0.999),
			"temperature-profile": reading.word(&profile, "mean", "hot-tongues"),
			"radius": reading.quantity(&fire.Radius,
				"m", 0.001, 1e4),
			"height": reading.quantity(&fire.Height,
				"m", 0.001, 1e4),
			"absorption": reading.quantity(
				&fire.Absorption, "m^-1", 0, 1e4),
			"temperature": reading.quantity(&hot, "K", 1,
				1e4),
			"edge-temperature": reading.quantity(&edge, "K",
				1, 1e4),
		}, nil)
		if err != nil {
			return err
		}
		fire.HotTongues = profile == "hot-tongues"
		fire.Hottest, fire.Edge = measure.Kelvin(hot),
			measure.Kelvin(edge)
		if composition != nil {
			composition.Reference = fire.Hottest
			fire.Composition, err = flame.NewComposition(*composition)
			if err != nil {
				return reading.refuse(node, "invalid emission composition: %v", err)
			}
		}
		return reading.all(node, "flame", "radius", "height",
			"absorption",
			"temperature", "edge-temperature")
	}
}

// fireSmoke reads explicit plume inputs without inventing fuel defaults.
func (reading reader) fireSmoke(spec *manufacture.BurnSpec) step {
	return func(_, node *yaml.Node) error {
		smoke := plume.Fire{}
		err := reading.keys(node, map[string]step{
			"heat-release": reading.quantity(
				&smoke.Heat, "kW", 0.001, 1e12),
			"convective-fraction": reading.number(
				&smoke.Convective, 1e-6, 1),
			"soot-yield": reading.number(
				&smoke.Yield, 1e-12, 1),
			"combustion": reading.quantity(&smoke.Combustion,
				"MJ/kg", 0.001,
				1e6),
			"mass-extinction": reading.quantity(
				&smoke.Extinction, "m^2/g", 0, 1e6),
			"air-temperature": reading.quantity(&smoke.Air,
				"K", 1, 1e4),
			"air-density": reading.quantity(
				&smoke.Density, "kg/m^3", 1e-9, 1e4),
			"height": reading.quantity(&smoke.Top,
				"m", 0.001, 1e6),
			"albedo": reading.number(&spec.Albedo, 0, 1),
		}, nil)
		if err != nil {
			return err
		}
		if err = reading.all(node, "smoke", "heat-release",
			"convective-fraction",
			"soot-yield",
			"combustion", "mass-extinction",
			"air-temperature", "air-density",
			"height", "albedo"); err != nil {
			return err
		}
		spec.Smoke = &smoke
		return nil
	}
}

// fireLayout reads the footprint independently of flame dimensions.
func (reading reader) fireLayout(layout *manufacture.FireLayout) step {
	return func(_, node *yaml.Node) error {
		count := 0.0
		radius, length := 0.0, 0.0
		err := reading.keys(node, map[string]step{
			"pattern": reading.word(&layout.Pattern,
				"cluster", "line"),
			"count": reading.whole(&count, 2, 32),
			"radius": reading.quantity(&radius, "m",
				0.001, 1e4),
			"length": reading.quantity(&length, "m",
				0.001, 1e4),
			"height-variation": reading.number(
				&layout.Variation, 0, 0.5),
		}, nil)
		if err != nil {
			return err
		}
		if err = reading.all(node, "fire layout", "pattern",
			"count"); err != nil {
			return err
		}
		layout.Count = int(count)
		layout.Radius, layout.Length = measure.Metres(radius),
			measure.Metres(length)
		dimension := "radius"
		if layout.Pattern == "line" {
			dimension = "length"
		}
		return reading.all(node, "fire layout", dimension)
	}
}
