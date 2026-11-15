package fixture_test

import (
	"encoding/csv"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/janearc/libtheme-css/primitives/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
)

// observerTable is the cie 1931 two-degree observer at one nanometre, as
// the colour and vision research laboratory keeps it: the cie's data, in
// the harness's own testdata, read here by the harness's own code.
const observerTable = "testdata/ciexyz31_1.csv"

// observer is x-bar, y-bar and z-bar by wavelength.
type observer map[int][3]float64

// readObserver is the table, every row.
func readObserver(t *testing.T) observer {
	t.Helper()
	file, err := os.Open(observerTable)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	seen := observer{}
	for _, row := range rows {
		nm, err := strconv.Atoi(row[0])
		if err != nil {
			t.Fatalf("row %v: %v", row, err)
		}
		var weights [3]float64
		for i := range weights {
			if weights[i], err = strconv.ParseFloat(row[i+1], 64); err != nil {
				t.Fatalf("row %v: %v", row, err)
			}
		}
		seen[nm] = weights
	}
	return seen
}

// integrate is a spectrum's X, Y and Z the cie's way: each wavelength's
// power times the observer at that wavelength, summed.
func (seen observer) integrate(spectrum map[int]float64) swatch.Swatch {
	var x, y, z float64
	for nm, power := range spectrum {
		weights, there := seen[nm]
		if !there {
			continue
		}
		x, y, z = x+power*weights[0], y+power*weights[1], z+power*weights[2]
	}
	return swatch.FromXYZ(x, y, z)
}

// apart is how far two colours are, in just-noticeable differences,
// once both are brought to the same luminance, so only colour counts.
func apart(a, b swatch.Swatch) float64 {
	_, ya, _ := a.XYZ()
	_, yb, _ := b.XYZ()
	ax, ay, az := a.XYZ()
	bx, by, bz := b.XYZ()
	one := swatch.FromXYZ(ax/ya, ay/ya, az/ya)
	other := swatch.FromXYZ(bx/yb, by/yb, bz/yb)
	first, second := ok.FromSwatch(one), ok.FromSwatch(other)
	dl, da, db := first.L-second.L, first.A-second.A, first.B-second.B
	return math.Sqrt(dl*dl+da*da+db*db) / ok.Eye
}

// blue is a paint that gives back short wavelengths and little else.
func blue(nm int) float64 { return blueAt(float64(nm)) }

// blueAt is blue at any wavelength, in nanometres.
func blueAt(nm float64) float64 {
	return 0.05 + 0.85/(1+math.Exp((nm-490)/12))
}

// times is a spectrum with each wavelength's power times a share.
func times(spectrum map[int]float64, share func(int) float64) map[int]float64 {
	product := make(map[int]float64, len(spectrum))
	for nm, power := range spectrum {
		product[nm] = power * share(nm)
	}
	return product
}

// Libtheme puts a spectrum's power into bands and weighs each band once;
// the cie weighs each wavelength. For daylight, a warm lamp and blue
// paint under it, the two may differ by less than one just-noticeable
// difference.
func TestLibthemeAgreesWithTheCIETable(t *testing.T) {
	seen := readObserver(t)
	lamp := swatch.Blackbody(2700)
	for name, spectrum := range map[string]map[int]float64{
		"d65": swatch.D65(), "a 2700 K lamp": lamp,
		"blue paint under it": times(lamp, blue)} {
		banded := bands.Split(spectrum).Swatch()
		gap := apart(banded, seen.integrate(spectrum))
		t.Logf("%s: libtheme %.3f JND from the cie table", name, gap)
		if gap > 1 {
			t.Errorf("%s: libtheme is %.2f JND from the cie table", name, gap)
		}
	}
}
