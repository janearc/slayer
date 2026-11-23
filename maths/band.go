package maths

// Band is one slice of the spectrum a light value carries, by its
// wavelength. It stands in for libtheme's band until libtheme has one;
// nothing in this package reads it.
//
// The zero Band means every band at once: one ray for the whole spectrum,
// which is all a sensor without glass ever needs.
type Band struct {
	//ninja:vow gate a value, whose fields are what it is
	Nanometres float64
}

// NewBand is the band at the wavelength nanometres. It keeps the number
// as given; zero is every band at once.
func NewBand(nanometres float64) Band { return Band{Nanometres: nanometres} }
