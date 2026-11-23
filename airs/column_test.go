package airs

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

// The wavelengths the tests look at, in nanometres.
var (
	visible = []float64{400, 450, 500, 550, 600, 630, 650, 700, 750, 800, 1000} // picked: the visible and the first near-infrared
	// tableWavelengths are the four the report's table is for.
	tableWavelengths = []float64{450, 550, 650, 750} // defined: the brief's four
)

// faults are what is wrong with a column's shells as a stack: one that
// does not begin on the ground, a gap or an overlap, a shell with no
// thickness, or a top that is not the air's.
func faults(column Column, top float64) []string {
	var found []string
	height := 0.0
	for index, layer := range column.Layers {
		if layer.Bottom != height {
			found = append(found, fmt.Sprintf("shell %d begins at %g, the last ended at %g",
				index, layer.Bottom, height))
		}
		if !(layer.Top > layer.Bottom) {
			found = append(found, fmt.Sprintf("shell %d has no thickness", index))
		}
		height = layer.Top
	}
	if height != top {
		found = append(found, fmt.Sprintf("the stack ends at %g, not %g", height, top))
	}
	return found
}

// The shells run without a gap from the ground to the top of the air,
// at every wavelength.
func TestShellsRunFromTheGroundToNinetyKilometresWithoutGaps(t *testing.T) {
	for _, nanometres := range visible {
		column := columnAt(t, venusRead, nanometres)
		if found := faults(column, 90000); len(found) > 0 { // defined: the brief's 90 km
			t.Errorf("%g nm: %v", nanometres, found)
		}
		for _, layer := range column.Layers {
			if layer.Top-layer.Bottom > venusRead.Gas.Shell.Value+1e-9 { // picked: rounding only
				t.Errorf("a shell %g to %g is thicker than the shell the data allows",
					layer.Bottom, layer.Top)
			}
		}
	}
}

// A stack with a shell taken out, or one set over another, is found.
func TestAStackWithAGapOrAnOverlapIsFound(t *testing.T) {
	column := columnAt(t, venusRead, 550)
	gap := Column{Layers: append(append([]Layer{}, column.Layers[:10]...), column.Layers[11:]...)} // picked: the eleventh shell taken out
	if len(faults(gap, 90000)) == 0 {
		t.Error("a gap was not found")
	}
	overlap := Column{Layers: append([]Layer{column.Layers[0]}, column.Layers...)}
	if len(faults(overlap, 90000)) == 0 {
		t.Error("an overlap was not found")
	}
	empty := Column{Layers: []Layer{{Bottom: 0, Top: 0}}}
	if len(faults(empty, 0)) == 0 {
		t.Error("a shell with no thickness was not found")
	}
	if len(faults(Column{Layers: column.Layers[:5]}, 90000)) == 0 { // picked: five shells, a short stack
		t.Error("a short stack was not found")
	}
}

// The air's particles, cloud and haze together, are 30 optical depths at
// 630 nm, whatever Mie says of the modes.
func TestParticleDepthAt630IsThirty(t *testing.T) {
	got := columnAt(t, venusRead, 630).particleDepth()
	if math.Abs(got-30) > 1e-9 { // defined: the brief's total; picked: rounding only
		t.Errorf("optical depth %g at 630 nm", got)
	}
}

// A depth picked a little other than the layers' own would not be 30:
// the picks are the sum.
func TestAPickedDepthChangesTheTotal(t *testing.T) {
	changed := copied(t)
	changed.Layers[1].Depth.Value = 11 // picked: inside the lower cloud's range
	got := columnAt(t, changed, 630).particleDepth()
	if math.Abs(got-30) < 0.5 { // picked: well over rounding
		t.Errorf("optical depth %g after a pick moved by a unit", got)
	}
}

// At 450 and at 750 nm the particles' depth differs from 630's only as
// Mie's shape does: within 15 percent.
func TestParticleDepthAtOtherColoursIsMiesShape(t *testing.T) {
	for _, nanometres := range []float64{450, 750} { // defined: the brief's two
		got := columnAt(t, venusRead, nanometres).particleDepth()
		if math.Abs(got-30) > 0.15*30 { // defined: the brief's 15 percent of 30
			t.Errorf("optical depth %g at %g nm", got, nanometres)
		}
	}
}

// And it does differ: the same scale at every wavelength is not the same
// depth at every wavelength, which is the shape.
func TestParticleDepthIsNotTheSameAtEveryColour(t *testing.T) {
	blue := columnAt(t, venusRead, 450).particleDepth()
	red := columnAt(t, venusRead, 750).particleDepth()
	if math.Abs(blue-red) < 0.1 { // picked: a tenth of a depth
		t.Errorf("depth %g at 450 nm and %g at 750", blue, red)
	}
}

// Every albedo is between nought and one and every asymmetry between
// minus one and one.
func TestAlbedosAndAsymmetriesAreInRange(t *testing.T) {
	for _, nanometres := range visible {
		column := columnAt(t, venusRead, nanometres)
		if column.GroundAlbedo < 0 || column.GroundAlbedo > 1 {
			t.Errorf("%g nm: ground albedo %g", nanometres, column.GroundAlbedo)
		}
		for _, layer := range column.Layers {
			if layer.ParticleAlbedo < 0 || layer.ParticleAlbedo > 1 ||
				layer.ParticleAsymmetry < -1 || layer.ParticleAsymmetry > 1 ||
				layer.GasScattering <= 0 || layer.GasAbsorption < 0 {
				t.Errorf("%g nm: %+v", nanometres, layer)
			}
		}
	}
}

// The clear air has no particles: no extinction, and so no albedo and no
// asymmetry to speak of; the cloud's shells have all three.
func TestClearAirHasNoParticles(t *testing.T) {
	column := columnAt(t, venusRead, 550)
	clear, cloud := column.Layers[0], column.Layers[len(column.Layers)/2+5] // picked: the lowest shell, and one in the cloud
	if clear.ParticleExtinction != 0 || clear.ParticleAlbedo != 0 || clear.ParticleAsymmetry != 0 {
		t.Errorf("the lowest shell: %+v", clear)
	}
	if cloud.ParticleExtinction <= 0 || cloud.ParticleAlbedo <= 0 || cloud.ParticleAsymmetry <= 0 {
		t.Errorf("a shell of the cloud: %+v", cloud)
	}
}

// The droplets do not absorb, so what they take out they scatter; and
// with some absorption in them the albedo is under one.
func TestAcidDropletsAreConservative(t *testing.T) {
	for _, layer := range columnAt(t, venusRead, 550).Layers {
		if layer.ParticleExtinction > 0 && math.Abs(layer.ParticleAlbedo-1) > 1e-9 { // picked: rounding only
			t.Errorf("albedo %g with no absorption", layer.ParticleAlbedo)
		}
	}
	changed := copied(t)
	changed.Particles.Imaginary.Value = 0.01 // picked: a little absorption, a tenth of the way to soot
	for _, layer := range columnAt(t, changed, 550).Layers {
		if layer.ParticleExtinction > 0 && layer.ParticleAlbedo > 0.999 { // picked: visibly under one
			t.Errorf("albedo %g with absorption", layer.ParticleAlbedo)
		}
	}
}

// An absorber takes light out only where it is and shortward of its
// limit, and what it takes is the particles' to lose: their albedo falls.
func TestAbsorbersAbsorbWhereTheyAre(t *testing.T) {
	changed := copied(t)
	changed.Absorbers[0].Absorption.Value = 1e-5 // picked: a visible amount a metre
	changed.Absorbers[1].Absorption.Value = 1e-5 // picked: as above
	base, with := columnAt(t, venusRead, 450), columnAt(t, changed, 450)
	for index, layer := range with.Layers {
		middle := (layer.Bottom + layer.Top) / 2
		inside := (middle > 64000 && middle < 70000) || (middle > 15000 && middle < 40000) // defined: the data's two ranges
		extra := layer.ParticleExtinction - base.Layers[index].ParticleExtinction
		if inside && (math.Abs(extra-1e-5) > 1e-12 || layer.ParticleAlbedo >= 1) {
			t.Errorf("shell %d at %g: extra %g albedo %g", index, middle, extra, layer.ParticleAlbedo)
		}
		if !inside && layer != base.Layers[index] {
			t.Errorf("shell %d at %g changed outside the absorbers", index, middle)
		}
	}
	ultraviolet := copied(t)
	ultraviolet.Absorbers[0].Absorption.Value = 1e-5 // picked: as above
	if columnAt(t, ultraviolet, 550).particleDepth() != columnAt(t, venusRead, 550).particleDepth() {
		t.Error("an absorber with a limit of 500 nm absorbed at 550 nm")
	}
}

// With the absorbers at nought, as shipped, the column's depths are as
// if they were not there: the anti-test of the one above, where they are
// not nought and the depth grows.
func TestAbsorbersAtNoughtChangeNothing(t *testing.T) {
	changed := copied(t)
	changed.Absorbers = nil
	with, without := columnAt(t, venusRead, 450), columnAt(t, changed, 450)
	if relative(with.particleDepth(), without.particleDepth()) > 1e-9 || // picked: rounding only
		relative(with.gasDepth(), without.gasDepth()) > 1e-9 {
		t.Error("absorbers at nought changed the column")
	}
	changed = copied(t)
	changed.Absorbers[1].Absorption.Value = 1e-5 // picked: as above
	added := columnAt(t, changed, 450).particleDepth() - with.particleDepth()
	if math.Abs(added-0.25) > 1e-9 { // defined: 25 km at 1e-5 a metre; picked: rounding only
		t.Errorf("a blue absorber of 1e-5 a metre adds %g of a depth, not 0.25", added)
	}
}

// The ground's albedo is linear between the table's points and flat past
// its ends.
func TestGroundAlbedoIsLinearBetweenPointsAndFlatBeyond(t *testing.T) {
	ground := venusRead.Ground
	for name, each := range map[string][2]float64{
		"between": {585, 0.0421875}, // picked: 0.4375 of the way from 550 to 630
		"a point": {630, 0.045},
		"below":   {300, 0.04},
		"above":   {1500, 0.145},
	} {
		if got := ground.albedo(each[0]); math.Abs(got-each[1]) > 1e-12 { // picked: rounding only
			t.Errorf("%s: albedo %g at %g nm, want %g", name, got, each[0], each[1])
		}
	}
	if ground.albedo(585) == ground.albedo(550) || ground.albedo(585) == ground.albedo(630) {
		t.Error("the albedo between two points is a point's own")
	}
}

// Data that could not be read stops Venus and the sun at Venus, and
// says why; once it is read they go on.
func TestDataThatCouldNotBeReadStopsVenusAndTheSun(t *testing.T) {
	before := venusError
	t.Cleanup(func() { venusError = before })
	venusError = errors.New("the file was not read")
	if _, err := Venus(550); !errors.Is(err, venusError) { // picked: the middle of the visible
		t.Errorf("Venus with unread data: %v", err)
	}
	if _, err := fixtureSun(t).AtVenus(400, 400.05); !errors.Is(err, venusError) {
		t.Errorf("the sun at Venus with unread data: %v", err)
	}
	venusError = before
	if _, err := Venus(550); err != nil {
		t.Errorf("Venus with the data read: %v", err)
	}
}

// The column carries the planet's radius, from the data.
func TestColumnCarriesTheRadius(t *testing.T) {
	if got := columnAt(t, venusRead, 550).Radius; got != 6051800 { // defined: scene/bodies/template/venus.go's 6051.8 km
		t.Errorf("radius %g", got)
	}
}

// What the column's means are, for a column made by hand.
func TestMeansOfAColumnMadeByHand(t *testing.T) {
	column := Column{Layers: []Layer{
		{Bottom: 0, Top: 1, ParticleExtinction: 1, ParticleAlbedo: 1, ParticleAsymmetry: 0.5},
		{Bottom: 1, Top: 3, ParticleExtinction: 1, ParticleAlbedo: 0.5, ParticleAsymmetry: 0.8},
	}}
	if got := column.meanAlbedo(); math.Abs(got-2.0/3.0) > 1e-12 { // defined: (1 + 2 x 0.5) over 3
		t.Errorf("mean albedo %g", got)
	}
	if got := column.meanAsymmetry(); math.Abs(got-0.65) > 1e-12 { // defined: (1 x 0.5 + 1 x 0.8) over 2
		t.Errorf("mean asymmetry %g", got)
	}
	if got := (Column{}).meanAlbedo(); got != 0 {
		t.Errorf("an empty column's mean albedo %g", got)
	}
	if got := (Column{}).meanAsymmetry(); got != 0 {
		t.Errorf("an empty column's mean asymmetry %g", got)
	}
}

// The table of Venus's column, at four colours. It is logged, not
// asserted: the report copies it.
func TestTableOfTheColumn(t *testing.T) {
	var table strings.Builder
	fmt.Fprintf(&table, "%6s %10s %10s %8s %8s %8s %12s\n", "nm", "rayleigh", "cloud",
		"albedo", "g", "ground", "sun W/m2/nm")
	sun, sunError := openSun()
	for _, nanometres := range tableWavelengths {
		column := columnAt(t, venusRead, nanometres)
		irradiance := math.NaN()
		if sunError == nil {
			var err error
			irradiance, err = sun.AtVenus(nanometres-tableBand, nanometres+tableBand)
			if err != nil {
				t.Fatal(err)
			}
		}
		fmt.Fprintf(&table, "%6.0f %10.3f %10.3f %8.4f %8.4f %8.4f %12.4f\n", nanometres,
			column.gasDepth(), column.particleDepth(), column.meanAlbedo(),
			column.meanAsymmetry(), column.GroundAlbedo, irradiance)
	}
	t.Log("\n" + table.String())
}

// tableBand is the half width, in nanometres, of the band the table's sun
// is averaged over. Picked: ten nanometres in all smooths the
// absorption lines the spectrum has at 0.1 nm.
const tableBand = 5.0
