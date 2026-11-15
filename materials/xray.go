package materials

// What the x-ray smoke's three materials take from a beam: lead,
// aluminium and liquid water, each with its mass attenuation
// coefficient at 60 keV and its density, from the NIST X-Ray Mass
// Attenuation Coefficients, Standard Reference Database 126, the tables
// of Hubbell and Seltzer.
//
// The data is NIST's and is not open; in NIST's own words: "© 1989,
// 1990, 1996 copyright by the U.S. Secretary of Commerce on behalf of
// the United States of America. All rights reserved." Only the rows
// used are quoted here, each as its page writes it. The pages
// themselves are held by celestial-data's nistxray adapter, each to the
// sha256 of its content, as NIST served them on 2026-11-11.
//
// A coefficient comes from the material's own page of the database; a
// density comes from its tables of material constants, Table 1 for the
// elements and Table 2 for water. NIST says of those tables that some
// density values are only nominal.

// nist is the database the x-ray rows are read from, and when
const nist = "NIST Standard Reference Database 126, X-Ray Mass Attenuation Coefficients, pages as served on 2026-11-11"

// hubbell is the work NIST asks to be cited for the database
const hubbell = "J. H. Hubbell and S. M. Seltzer, Tables of X-Ray Mass Attenuation Coefficients and Mass Energy-Absorption Coefficients (version 1.4), NIST, 2004, doi 10.18434/T4D01F"

// FilmEnergy is the photon energy the x-ray rows are read at, in MeV:
// 60 keV, a picked medical-film energy, about 0.021 nm
const FilmEnergy = 0.06

// coefficientRow is one row of a material's page: the page, the photon
// energy in MeV, mu over rho in square centimetres a gram, and the row
// as the page writes it: energy, mu over rho, and the energy-absorption
// coefficient beside it
type coefficientRow struct {
	page      string
	energy    float64
	muOverRho float64
	text      string
}

// densityRow is one row of a table of material constants: the page, the
// density in grams a cubic centimetre, and the row's cells in order
type densityRow struct {
	page    string
	density float64
	text    string
}

// xrayed is a material with its density and its coefficients, each read
// through the constructors from its row, refusing a bad row, no rows of
// coefficients, and two rows at one energy
func xrayed(kind Material, dense densityRow,
	rows []coefficientRow) (Material, error) {
	from, err := NewSource(nist, kind.name+", "+dense.page, hubbell,
		dense.text)
	if err != nil {
		return Material{}, err
	}
	kind, err = kind.AtDensity(dense.density, from)
	if err != nil {
		return Material{}, err
	}
	taken := make([]Attenuation, 0, len(rows))
	for _, each := range rows {
		read, err := NewSource(nist, kind.name+", "+each.page, hubbell,
			each.text)
		if err != nil {
			return Material{}, err
		}
		coefficient, err := NewAttenuation(each.energy, each.muOverRho, read)
		if err != nil {
			return Material{}, err
		}
		taken = append(taken, coefficient)
	}
	stops, err := NewAttenuations(taken...)
	if err != nil {
		return Material{}, err
	}
	return kind.Attenuating(stops)
}

// aluminiumDense and aluminiumTaken are aluminium's row of Table 1 and
// the 60 keV row of its page
var (
	aluminiumDense = densityRow{"tab1.html", 2.699,
		"13 Al Aluminum 0.48181 166.0 2.699E+00"}
	aluminiumTaken = coefficientRow{"ElemTab/z13.html", FilmEnergy, 0.2778,
		"6.00000E-02  2.778E-01  1.099E-01"}
)

// xrays are the pages of the x-ray smoke's two materials that are not
// metals, by name, with no optics cited for either
var xrays = map[string]page{
	// the element: its density from row 82 of Table 1 and its
	// coefficient from the 60 keV row of its page
	"lead": {parts: map[string]float64{"Pb": 1},
		dense: &densityRow{"tab1.html", 11.35,
			"82 Pb Lead 0.39575 823.0 1.135E+01"},
		stops: []coefficientRow{{"ElemTab/z82.html", FilmEnergy, 5.021,
			"6.00000E-02  5.021E+00  4.149E+00"}}},
	// water as a liquid, the soft thing of the x-ray smoke: its density
	// from the row Water, Liquid of Table 2 and its coefficient from
	// the 60 keV row of its page
	"water": {parts: map[string]float64{"H": 2, "O": 1},
		dense: &densityRow{"tab2.html", 1.000,
			"Water, Liquid 0.55508 75.0 1.000E+00 1: 0.111898 8: 0.888102"},
		stops: []coefficientRow{{"ComTab/water.html", FilmEnergy, 0.2059,
			"6.00000E-02  2.059E-01  3.190E-02"}}},
}
