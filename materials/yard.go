package materials

// The yard's six substances, by name: what each is made of, as a
// formula, and nothing more. No bulk density and no optics are cited
// for them here, so they are made as substances and answer nought and
// never-made: the yard's looks are painted, and paint covers whatever
// it meets, so nothing asks these for readings. Each composition is an
// authored choice, said so below, not a citation.

// yard are the pages of the yard's six substances, by name. Neither a
// density nor optics is cited yet for any of them.
var yard = map[string]page{
	// a plain carbon steel, iron with a little carbon in it, about a
	// fifth of a percent by share, an authored choice: no grade is
	// named and no row is cited
	"steel": {parts: map[string]float64{"Fe": 0.998, "C": 0.002}},
	// ferric oxide, Fe2O3, an authored choice for the yard's pigment
	// red: no source is cited for the formula and none for what it
	// would measure
	"ochre": {parts: map[string]float64{"Fe": 2, "O": 3}},
	// titanium dioxide, TiO2, an authored choice for the yard's white
	// coat
	"white": {parts: map[string]float64{"Ti": 1, "O": 2}},
	// cellulose, C6H10O5, an authored choice for the yard's boards: no
	// species is named and no analysis cited
	"timber": {parts: map[string]float64{"C": 6, "H": 10, "O": 5}},
	// polyisoprene, C5H8, an authored choice for the yard's rubber: the
	// isoprene unit, with no citation of the real compound or its
	// carbon black
	"tyre": {parts: map[string]float64{"C": 5, "H": 8}},
	// silica, SiO2, an authored choice for the yard's opaque glass,
	// which is not the silica an optical table would carry
	"window": {parts: map[string]float64{"Si": 1, "O": 2}},
}
