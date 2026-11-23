package thing

import (
	"fmt"

	"github.com/janearc/slayer/materials"
)

// The metals spacecraft are made of, and the film their blankets wear.
// Measured metals carry their complex index at the wavelengths each
// source tabulates nearest the tracer's red, green and blue (610, 550
// and 465 nm).
//
// The n and k are not held here: materials holds them, in
// the rows the refractiveindex.info database release v2026-05-24 lists
// from the papers named below, committed under materials/testdata.
//
// The book below is built from materials once, before any look
// is made, and each constructor reads its page's readings at red,
// green and blue into N and K.

// metals is the book of measured metals and the explicit bronze estimate, by name,
// each page materials.ByName made. A page that cannot be made, or one
// without a reading for every band, is a mistake in slayer's own code,
// which its tests find, so making the book stops the program and says
// which.
var metals = readMetals("gold", "silver", "aluminium", "beryllium",
	"titanium", "nickel-aluminium-bronze")

// readMetals builds the book of the metals named from materials, before
// any look is made.
func readMetals(names ...string) map[string]materials.Material {
	book := make(map[string]materials.Material, len(names))
	for _, name := range names {
		made, err := materials.ByName(name)
		if err != nil {
			panic(fmt.Sprintf("thing: the book of metals is broken: %v", err))
		}
		book[name] = made
	}
	return book
}

// look is a metal made of one page of the book: the page's readings at
// red, green and blue, in the tracer's order, as N and K. A page
// without a reading for a band is a mistake in slayer's own code, and
// stops the program rather than drawing a metal with missing readings.
func look(name string) *Metal {
	sight := metals[name].Optics()
	made := &Metal{}
	for at, band := range [...]materials.Band{materials.Red, materials.Green,
		materials.Blue} {
		if !sight.Has(band) {
			panic(fmt.Sprintf("thing: %s has no %s reading",
				name, band))
		}
		read := sight.At(band)
		made.N[at], made.K[at] = read.N(), read.K()
	}
	return made
}

// Gold is Johnson and Christy's (Phys. Rev. B 6, 4370, 1972), at 617,
// 549 and 471 nm: face on it mirrors 0.93 of red, 0.79 of green and
// 0.40 of blue. Webb's mirrors are gold. materials holds its readings
// in the rows 0.6168, 0.5486 and 0.4714 of main/Au/nk/Johnson.yml, in
// the refractiveindex.info database release v2026-05-24.
func Gold() *Metal {
	return look("gold")
}

// Silver is Johnson and Christy's, at 617, 549 and 471 nm: face on it
// mirrors 0.99 of red and 0.98 of green and of blue. An optical solar
// reflector is silver under glass.
// materials holds its readings in the rows 0.6168, 0.5486 and 0.4714 of
// main/Ag/nk/Johnson.yml, in the refractiveindex.info database release
// v2026-05-24.
func Silver() *Metal {
	return look("silver")
}

// Aluminium is Rakic and others' Brendel-Bormann fit (Appl. Opt. 37,
// 5271, 1998), at 613.6, 541.5 and 458.3 nm: face on it mirrors 0.91
// of red and of green and 0.92 of blue. A blanket's and Webb's
// sunshield's coating is aluminium.
//
// materials holds its readings in the rows 6.1359e-01, 5.4147e-01 and
// 4.5833e-01 of main/Al/nk/Rakic-BB.yml, in the refractiveindex.info
// database release v2026-05-24; its blue n is the published row's
// 0.64439.
func Aluminium() *Metal {
	return look("aluminium")
}

// Titanium is Johnson and Christy's (Phys. Rev. B 9, 5056, 1974), at
// 617, 549 and 471 nm, as refractiveindex.info lists it: face on it
// mirrors 0.61 of red, 0.58 of green and 0.55 of blue, a warm grey.
// materials holds its readings in the rows 0.617, 0.549 and 0.471 of
// main/Ti/nk/Johnson.yml, in the refractiveindex.info database release
// v2026-05-24.
func Titanium() *Metal {
	return look("titanium")
}

// NickelAluminiumBronze is an estimated conductor appearance for C95800.
// The three normal reflectances are digitized from deposited C63280 sample K
// in Mullaney and Tatam (2025), Figure 6. Its n=1 closure is not measured
// complex-index data. See doc/model/material/nickel-aluminium-bronze.md
// for limitations.
func NickelAluminiumBronze() *Metal {
	return look("nickel-aluminium-bronze")
}

// Rutile is titanium dioxide so many metres thick, as titanium's own
// oxide grows over it in air: clear, at Devore's ordinary index for
// rutile at 549 nm, 2.65 (J. Opt. Soc. Am. 41, 416, 1951: n squared is
// 5.913 + 0.2441 / (l squared - 0.0803), l in microns; 2.59 at 617 nm
// and 2.76 at 471, where a film has one index). The colours a thicker
// oxide shows are interference, which a Film does not draw.
func Rutile(thick float64) *Film {
	return &Film{Index: metalsRutileIndex, Thickness: thick}
}

// metalsRutileIndex is rutile's ordinary index at 549 nm, from picks.yaml.
var metalsRutileIndex = pick("rutile-index")

// metalsKaptonIndex is a polyimide film's refractive index, from picks.yaml.
var metalsKaptonIndex = pick("kapton-index")

// metalsKaptonAbsorbs is how much of red, green and blue a polyimide film
// absorbs, per metre, from picks.yaml.
var metalsKaptonAbsorbs = [metalBands]float64{
	metalRed:   pick("kapton-absorbs-red"),
	metalGreen: pick("kapton-absorbs-green"),
	metalBlue:  pick("kapton-absorbs-blue"),
}

// Kapton is a polyimide film so many metres thick, amber, as a
// blanket's outer layer is over its aluminium. Its index, 1.7, and how
// it absorbs are from secondary sources read only in summary: a film of
// about 40 microns passes about 80 percent above 600 nm and next to
// nothing below 500, its edge near 550.
//
// So its absorption here is that film's: -ln 0.8 over 40 microns in
// red, 5.6 thousand a metre; about
// 30 percent passing in green, 30 thousand; a thousandth in blue, 170
// thousand. Picked from those, not measured; the gold of a Kapton
// blanket follows from them.
func Kapton(thick float64) *Film {
	return &Film{Index: metalsKaptonIndex, Thickness: thick,
		Absorbs: metalsKaptonAbsorbs}
}
