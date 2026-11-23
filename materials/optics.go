package materials

import (
	"fmt"
	"math"
)

// Band is one of the tracer's three colours, red, green and blue, the
// wavelengths it reads light at
type Band int

// the tracer's bands, in its order
const (
	Red Band = iota
	Green
	Blue
)

// String is the band's name
func (band Band) String() string {
	switch band {
	case Red:
		return "red"
	case Green:
		return "green"
	case Blue:
		return "blue"
	}
	return fmt.Sprintf("band %d", int(band))
}

// Source records a reading's provenance: dataset or study, material/sample,
// paper and source record. Estimated marks a derived appearance reading.
type Source struct {
	release   string
	material  string
	paper     string
	row       string
	estimated bool
}

// newSource is where a reading was read. It refuses a release, a
// material, a paper or a row that is not named.
func newSource(release, material, paper, row string) (Source, error) {
	if release == "" {
		return Source{}, refuse("a source with no release")
	}
	if material == "" {
		return Source{}, refuse("a source with no material")
	}
	if paper == "" {
		return Source{}, refuse("a source with no paper")
	}
	if row == "" {
		return Source{}, refuse("a source with no row")
	}
	return Source{release: release, material: material,
		paper: paper, row: row}, nil
}

// Release is the database release the row is in
func (where Source) Release() string { return where.release }

// Material is the material, and its file, within the release
func (where Source) Material() string { return where.material }

// Paper is the paper the row comes from
func (where Source) Paper() string { return where.paper }

// Row is the source record: a measured table row, or the reflectance record
// underlying an estimate. Estimated distinguishes derived optical readings.
func (where Source) Row() string { return where.row }

// Reading is a complex refractive index, n + ik, at one wavelength, with
// where it was read. The wavelength is in nanometres. Nothing is stored
// of absorption: Absorption works it out from k and the wavelength.
type Reading struct {
	wavelength float64
	n, k       float64
	where      Source
}

// newReading is n + ik at a wavelength in nanometres, read somewhere. It
// refuses a wavelength that is not a positive number, an n or a k that
// is negative or not a number, and a source never made.
func newReading(wavelength, n, k float64, where Source) (Reading, error) {
	if !positive(wavelength) {
		return Reading{}, refuse("a reading at wavelength %v nm",
			wavelength)
	}
	if negative(n) {
		return Reading{}, refuse("a reading of n %v", n)
	}
	if negative(k) {
		return Reading{}, refuse("a reading of k %v", k)
	}
	if where.release == "" {
		return Reading{}, refuse("a reading read nowhere")
	}
	return Reading{wavelength: wavelength, n: n, k: k, where: where}, nil
}

// Wavelength is the reading's wavelength in nanometres
func (read Reading) Wavelength() float64 { return read.wavelength }

// N is the reading's refractive index
func (read Reading) N() float64 { return read.n }

// K is the reading's extinction coefficient, the imaginary part of the
// index
func (read Reading) K() float64 { return read.k }

// Where is where the reading was read
func (read Reading) Where() Source { return read.where }

// Absorption is how much of the band's light a metre of the material
// takes: four pi k over the wavelength. It is worked out from the stored
// k and wavelength every time it is asked; nothing of it is stored.
func (read Reading) Absorption() float64 {
	return 4 * math.Pi * read.k / (read.wavelength * 1e-9)
}

// Optics holds complex-index readings by band at their cited wavelengths.
// Sources identify measured rows or derived estimates. Only NewOptics
// constructs one; missing bands do not become supplied readings.
type Optics struct {
	bands map[Band]Reading
}

// newOptics is readings by band. It refuses no bands and a reading never
// made.
func newOptics(bands map[Band]Reading) (Optics, error) {
	if len(bands) == 0 {
		return Optics{}, refuse("optics of no bands")
	}
	kept := make(map[Band]Reading, len(bands))
	for band, read := range bands {
		if read.wavelength == 0 {
			return Optics{}, refuse("the %s reading was never made", band)
		}
		kept[band] = read
	}
	return Optics{bands: kept}, nil
}

// At is the reading for a band, a reading never made when the optics
// have none for it
func (sight Optics) At(band Band) Reading { return sight.bands[band] }

// Has is whether the optics have a reading for a band
func (sight Optics) Has(band Band) bool {
	_, there := sight.bands[band]
	return there
}

// negative is whether a number is below zero, infinite, or not a number
func negative(number float64) bool {
	return !(number >= 0) || math.IsInf(number, 0)
}
