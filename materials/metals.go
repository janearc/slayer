package materials

// The metals a thing can say it is made of, by name, with their optics
// as the refractiveindex.info database release v2026-05-24 lists them
// (cite the database as Polyanskiy, M. N. Refractiveindex.info database
// of optical constants, Scientific Data 11, 94 (2024),
// doi 10.1038/s41597-023-02898-2; it is in the public domain, copyright
// and related rights waived via CC0 1.0).
//
// The database files the rows
// come from are committed under testdata, whole and unchanged, and the
// tests read the values back out of them.
//
// Each metal is its complex index, n + ik, at the rows nearest the
// tracer's red, green and blue. scene/thing/metals.go reads the values
// from here, through ByName, so the two cannot disagree.
//
// Aluminium's blue n was corrected to the published row on 2026-11-11,
// by decision: thing had 0.6444 where the published row, line 63
// of main/Al/nk/Rakic-BB.yml, has 6.4439e-01, and thing now carries the
// published value, as this package does.
//
// The yard's six substances, of yard.go, are not metals and carry no
// optics; they come through the same door, ByName, so one lookup finds
// every material a thing can say it is made of.

// release is the database release the metals' rows are read from
const release = "refractiveindex.info database release v2026-05-24"

// the papers the metals' rows come from
const (
	noble      = "P. B. Johnson and R. W. Christy, Optical constants of the noble metals, Phys. Rev. B 6, 4370-4379 (1972)"
	transition = "P. B. Johnson and R. W. Christy, Optical constants of transition metals: Ti, V, Cr, Mn, Fe, Co, Ni, and Pd, Phys. Rev. B 9, 5056-5070 (1974)"
	rakic      = "A. D. Rakić, A. B. Djurišic, J. M. Elazar, M. L. Majewski, Optical properties of metallic films for vertical-cavity optoelectronic devices, Appl. Opt. 37, 5271-5283 (1998)"
)

// row is one row of a metal's optics: the band it answers, the
// wavelength in nanometres, n and k, and where the row was read: the
// material and its file within the database, the paper, and the row as
// the file writes it
type row struct {
	band       Band
	wavelength float64
	n, k       float64
	material   string
	file       string
	paper      string
	text       string
}

// optics is readings from a table of rows, each read through the
// constructors, refusing a bad row
func optics(table []row) (Optics, error) {
	bands := make(map[Band]Reading, len(table))
	for _, each := range table {
		where, err := NewSource(release,
			each.material+", "+each.file, each.paper, each.text)
		if err != nil {
			return Optics{}, err
		}
		read, err := NewReading(each.wavelength, each.n, each.k, where)
		if err != nil {
			return Optics{}, err
		}
		bands[each.band] = read
	}
	return NewOptics(bands)
}

// page is what the database holds of one material: what it is made of;
// the rows of its optics, where any are cited; and its row of density
// with its rows of x-ray coefficients, where those are. A page is data
// and nothing more: made reads it through the constructors.
type page struct {
	parts map[string]float64
	sight []row
	dense *densityRow
	stops []coefficientRow
}

// made is the page's material under a name, every part of it read
// through the constructors, refusing a bad page by what is wrong in it
func (leaf page) made(name string) (Material, error) {
	formula, err := NewFormula(leaf.parts)
	if err != nil {
		return Material{}, err
	}
	kind, err := leaf.plain(name, formula)
	if err != nil {
		return Material{}, err
	}
	if leaf.dense == nil {
		return kind, nil
	}
	return xrayed(kind, *leaf.dense, leaf.stops)
}

// plain is the page's material before any x-ray row: with its optics
// where the page has rows of them, and a substance of its formula alone
// where it has none
func (leaf page) plain(name string, formula Formula) (Material, error) {
	if len(leaf.sight) == 0 {
		return NewSubstance(name, formula)
	}
	sight, err := optics(leaf.sight)
	if err != nil {
		return Material{}, err
	}
	return NewOpticalMaterial(name, formula, sight)
}

// ByName is the material a thing says it is made of, by its name: the
// metals of this file, with their optics, the yard's substances of
// yard.go, with their formulas alone, and lead and water of xray.go,
// with what they take from an x-ray beam. It refuses a name it does not
// know, by name.
func ByName(name string) (Material, error) {
	for _, book := range []map[string]page{metals, semiconductors, yard, xrays} {
		if leaf, known := book[name]; known {
			return leaf.made(name)
		}
	}
	return Material{}, refuse("no material %q", name)
}

// metals are the pages of the five metals, by name. No bulk density is
// cited here for any but aluminium; one comes with a source in a later
// step, and Density answers nought until then.
var metals = map[string]page{
	// Johnson and Christy's gold, the rows 0.6168, 0.5486 and 0.4714
	// of main/Au/nk/Johnson.yml, the ones nearest the tracer's red,
	// green and blue.
	"gold": {parts: map[string]float64{"Au": 1}, sight: []row{
		{Red, 616.8, 0.21, 3.272, "gold", "main/Au/nk/Johnson.yml", noble, "0.6168 0.21 3.272"},
		{Green, 548.6, 0.43, 2.455, "gold", "main/Au/nk/Johnson.yml", noble, "0.5486 0.43 2.455"},
		{Blue, 471.4, 1.31, 1.849, "gold", "main/Au/nk/Johnson.yml", noble, "0.4714 1.31 1.849"},
	}},
	// Johnson and Christy's silver, the rows 0.6168, 0.5486 and 0.4714
	// of main/Ag/nk/Johnson.yml, the ones nearest the tracer's red,
	// green and blue.
	"silver": {parts: map[string]float64{"Ag": 1}, sight: []row{
		{Red, 616.8, 0.06, 4.152, "silver", "main/Ag/nk/Johnson.yml", noble, "0.6168 0.06 4.152"},
		{Green, 548.6, 0.06, 3.586, "silver", "main/Ag/nk/Johnson.yml", noble, "0.5486 0.06 3.586"},
		{Blue, 471.4, 0.05, 2.869, "silver", "main/Ag/nk/Johnson.yml", noble, "0.4714 0.05 2.869"},
	}},
	// Rakić and others' Brendel-Bormann fit of aluminium, the rows
	// 6.1359e-01, 5.4147e-01 and 4.5833e-01 of main/Al/nk/Rakic-BB.yml,
	// the ones nearest the tracer's red, green and blue. Its density
	// and what it takes from an x-ray beam are NIST's, of xray.go.
	"aluminium": {parts: map[string]float64{"Al": 1}, sight: []row{
		{Red, 613.59, 1.1911, 7.0711, "aluminium", "main/Al/nk/Rakic-BB.yml", rakic, "6.1359e-01 1.1911e+00 7.0711e+00"},
		{Green, 541.47, 0.93525, 6.3099, "aluminium", "main/Al/nk/Rakic-BB.yml", rakic, "5.4147e-01 9.3525e-01 6.3099e+00"},
		{Blue, 458.33, 0.64439, 5.3994, "aluminium", "main/Al/nk/Rakic-BB.yml", rakic, "4.5833e-01 6.4439e-01 5.3994e+00"},
	}, dense: &aluminiumDense, stops: []coefficientRow{aluminiumTaken}},
	// Rakić and others' Lorentz-Drude fit of beryllium, the rows
	// 6.1045e-01, 5.4960e-01 and 4.6562e-01 of main/Be/nk/Rakic-LD.yml,
	// the ones nearest the tracer's red, green and blue.
	"beryllium": {parts: map[string]float64{"Be": 1}, sight: []row{
		{Red, 610.45, 3.4188, 3.1684, "beryllium", "main/Be/nk/Rakic-LD.yml", rakic, "6.1045e-01 3.4188e+00 3.1684e+00"},
		{Green, 549.6, 3.3455, 3.1517, "beryllium", "main/Be/nk/Rakic-LD.yml", rakic, "5.4960e-01 3.3455e+00 3.1517e+00"},
		{Blue, 465.62, 3.1353, 3.1466, "beryllium", "main/Be/nk/Rakic-LD.yml", rakic, "4.6562e-01 3.1353e+00 3.1466e+00"},
	}},
	// Johnson and Christy's titanium, the rows 0.617, 0.549 and 0.471
	// of main/Ti/nk/Johnson.yml, the ones nearest the tracer's red,
	// green and blue.
	"titanium": {parts: map[string]float64{"Ti": 1}, sight: []row{
		{Red, 617, 2.67, 3.72, "titanium", "main/Ti/nk/Johnson.yml", transition, "0.617 2.67 3.72"},
		{Green, 549, 2.54, 3.43, "titanium", "main/Ti/nk/Johnson.yml", transition, "0.549 2.54 3.43"},
		{Blue, 471, 2.32, 3.10, "titanium", "main/Ti/nk/Johnson.yml", transition, "0.471 2.32 3.10"},
	}},
}
