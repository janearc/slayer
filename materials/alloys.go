package materials

// reflectancePage is an appearance proxy with an explicit experimental basis.
// Its normal reflectances become effective, nonunique conductor indices.
type reflectancePage struct {
	release, material, paper string
	rows                     []reflectanceRow
}

type reflectanceRow struct {
	band                    Band
	wavelength, reflectance float64
	text                    string
}

// optics turns the page's rows into one reading for each band, each with
// its own source. It refuses a band outside red to blue and a band given
// twice.
func (leaf reflectancePage) optics() (Optics, error) {
	bands := make(map[Band]Reading, len(leaf.rows))
	for _, row := range leaf.rows {
		if row.band < Red || row.band > Blue {
			return Optics{}, refuse("unsupported reflectance band %s", row.band)
		}
		if _, duplicate := bands[row.band]; duplicate {
			return Optics{}, refuse("two reflectance estimates for %s", row.band)
		}
		source, err := newSource(leaf.release, leaf.material, leaf.paper, row.text)
		if err != nil {
			return Optics{}, err
		}
		read, err := conductorFromReflectance(row.wavelength, row.reflectance, source)
		if err != nil {
			return Optics{}, err
		}
		bands[row.band] = read
	}
	return newOptics(bands)
}

// bulkRow is a cited density independent of any radiation coefficient table.
// A density does not imply that attenuation coefficients are available.
type bulkRow struct {
	density                        float64
	release, material, paper, text string
}

// attach gives a material the row's density, with the row's citation as
// the density's source.
func (row bulkRow) attach(kind Material) (Material, error) {
	source, err := newSource(row.release, row.material, row.paper, row.text)
	if err != nil {
		return Material{}, err
	}
	return kind.atDensity(row.density, source)
}

// alloys use selected nominal compositions. They are not certified samples.
// Appearance proxies are separate from grade/composition and marked estimated.
var alloys = map[string]page{
	"nickel-aluminium-bronze": {
		// C95800 chosen mass shares, converted to atom-count proportions.
		parts: map[string]float64{
			"Cu": .815 / weights["Cu"], "Al": .09 / weights["Al"],
			"Ni": .045 / weights["Ni"], "Fe": .04 / weights["Fe"],
			"Mn": .01 / weights["Mn"],
		},
		bulk: &bulkRow{7.64, "Wieland Concast C95800 data, read 2026-11-17",
			"UNS C95800 at 20 degrees C", "https://www.concast.com/c95800.php",
			"Density 0.276 lb/in3 at 68 F; converted and rounded to 7.64 g/cm3"},
		proxy: &reflectancePage{
			release:  "Mullaney and Tatam 2025, Figure 6; two-significant-figure digitization",
			material: "as-deposited UNS C63280 sample K; C95800 appearance proxy",
			paper: "Emissivity Measurements of Metals Used in Wire-Arc-Directed " +
				"Energy Deposition Processes, doi:10.3390/met15101078",
			rows: []reflectanceRow{
				{Red, 610, .46, "610 nm: R approximately 0.46; digitization allowance +/-0.01"},
				{Green, 550, .40, "550 nm: R approximately 0.40; digitization allowance +/-0.01"},
				{Blue, 465, .27, "465 nm: R approximately 0.27; digitization allowance +/-0.01"},
			},
		},
	},
}
