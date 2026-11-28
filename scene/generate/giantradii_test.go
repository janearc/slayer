package generate

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/scene/bodies/template"
)

// saturnKilometres is Saturn's mean radius (NASA's Saturn fact sheet,
// Williams, NSSDCA), the figure beside Jupiter's 69911 km mean.
const saturnKilometres = 58232

// hotJupiterYear is the hot jupiter template's orbit in years: its
// eighteen hours, as template.HotJupiter's own comment says.
const hotJupiterYear = 18.0 / (365.25 * 24)

// likeSaturn is Saturn's numbers: 95.16 Earth masses, a core of about
// fifteen in a hundred of it, at 9.58 astronomical units.
func likeSaturn() Planet {
	return Planet{
		Name:   "like saturn",
		Mass:   measure.Kilograms(5.683e26),
		Makeup: Makeup{Metal: 0.03, Rock: 0.05, Ice: 0.07, Gas: 0.85},
		Sun:    sunLike,
		Orbit:  Orbit{Distance: measure.Metres(9.58 * au)},
		Age:    4.5e9,
		Seed:   10,
	}
}

// kilometresAt is a planet's radius in kilometres, or the test's end.
func kilometresAt(t *testing.T, planet Planet) float64 {
	t.Helper()
	derived, err := Derive(planet)
	if err != nil {
		t.Fatal(err)
	}
	return derived.Radius.InKilometres()
}

func TestTablesAreReadWhole(t *testing.T) {
	grid := fortney()
	if len(grid.ages) != 3 || len(grid.distances) != 5 ||
		len(grid.cores) != 5 {
		t.Fatalf("ages %d, distances %d, cores %d; want 3, 5, 5",
			len(grid.ages), len(grid.distances), len(grid.cores))
	}
	for age := range grid.radius {
		for distance, rows := range grid.radius[age] {
			if len(rows) != len(grid.cores) {
				t.Fatalf("age %d distance %d has %d rows", age, distance,
					len(rows))
			}
			for core, row := range rows {
				if len(row) != len(giantMasses) {
					t.Errorf("age %d distance %d core %d: %d cells",
						age, distance, core, len(row))
				}
			}
		}
	}
}

func TestTablesHoldThePapersCells(t *testing.T) {
	grid := fortney()
	// table 4, 1.0 AU, core 25, 318 Earth masses; table 2, 0.045 AU,
	// no core, 17 Earth masses, the largest radius in the three
	if got := grid.radius[2][3][2][6]; got != 1.026 {
		t.Errorf("table 4 cell %v, want 1.026", got)
	}
	if got := grid.radius[0][1][0][0]; got != 2.795 {
		t.Errorf("table 2 cell %v, want 2.795", got)
	}
	// the star where table 2's 0.02 AU coreless model did not converge
	if got := grid.radius[0][0][0][0]; !math.IsNaN(got) {
		t.Errorf("empty cell read as %v", got)
	}
}

func TestLikeSaturnComesOutNearSaturn(t *testing.T) {
	radius := kilometresAt(t, likeSaturn())
	if !near(radius, saturnKilometres, 0.10) {
		t.Errorf("radius %.0f km, want %d within 10%%", radius,
			saturnKilometres)
	}
}

func TestAHotGiantIsLargerThanACoolOne(t *testing.T) {
	cool := likeJupiter()
	hot := likeJupiter()
	hot.Orbit = Orbit{Distance: measure.Metres(0.05 * au)}
	cool.Orbit = Orbit{Distance: measure.Metres(5 * au)}
	close, far := kilometresAt(t, hot), kilometresAt(t, cool)
	t.Logf("jupiter's mass: %.0f km at 0.05 AU, %.0f km at 5 AU", close, far)
	t.Logf("jupiter itself: %.0f km", kilometresAt(t, likeJupiter()))
	t.Logf("saturn itself: %.0f km", kilometresAt(t, likeSaturn()))
	if close <= far {
		t.Errorf("at 0.05 AU %.0f km, at 5 AU %.0f km; want larger near",
			close, far)
	}
}

// cellRange is the smallest and largest radius the tables print, in
// Jupiter radii.
func cellRange() (float64, float64) {
	least, most := math.Inf(1), 0.0
	for _, age := range fortney().radius {
		for _, distance := range age {
			for _, row := range distance {
				for _, cell := range row {
					if !math.IsNaN(cell) {
						least, most = min(least, cell), max(most, cell)
					}
				}
			}
		}
	}
	return least, most
}

func TestColdRadiusStaysInThePapersRange(t *testing.T) {
	least, most := cellRange()
	const steps = 40
	ratio := math.Pow(giantMasses[len(giantMasses)-1]/giantMasses[0],
		1.0/steps)
	for _, au := range []float64{0.01, 0.02, 0.07, 0.4, 3, 9.5, 30} {
		for _, gigayears := range []float64{0.1, 0.3, 2, 4.5, 9} {
			mass := giantMasses[0]
			for range steps + 1 {
				got := coldGiantRadius(mass, 0.1*mass, au, gigayears)
				if got < least || got > most {
					t.Errorf("%.0f Earths, %v AU, %v Gyr: %.3f, "+
						"want %.3f to %.3f", mass, au, gigayears, got,
						least, most)
				}
				mass *= ratio
			}
		}
	}
}

func TestInflationMeetsTheColdRadiusWithoutAStep(t *testing.T) {
	// table 1 of Sestovic et al.: the 0.98 to 2.5 Jupiter mass bin
	// starts at log10 F of 5.82; 1.57 Jupiter masses is in its middle
	flux := math.Pow(10, 5.82)
	if below := inflation(1.57, flux*0.999); below != 0 {
		t.Errorf("below the threshold: %v", below)
	}
	if above := inflation(1.57, flux*1.001); above <= 0 || above > 0.001 {
		t.Errorf("just above the threshold: %v, want a hair over zero",
			above)
	}
}

func TestTheMiddleOfEachBinIsThePapersNumbers(t *testing.T) {
	// table 1 of Sestovic et al.: A and log10 Fs of each mass bin, taken
	// at a mass in the middle of it, clear of every blend
	for _, each := range []struct{ mass, rise, threshold float64 }{
		{0.15, 0, 5.52}, {0.6, 0.70, 5.52}, {1.5, 0.52, 5.82},
		{5, 0.22, 5.2},
	} {
		rise, threshold := inflationAt(each.mass)
		if rise != each.rise || threshold != each.threshold {
			t.Errorf("%v Jupiter masses: A %v, log10 Fs %v; want %v, %v",
				each.mass, rise, threshold, each.rise, each.threshold)
		}
	}
}

func TestInflationFadesInAcrossTheLightEdge(t *testing.T) {
	const flux = 1e8
	edge := inflation(0.37, flux)
	below := inflation(0.37*math.Pow(10, -0.125), flux)
	if !(below > 0 && below < edge) {
		t.Errorf("half way into the blend %v, at the edge %v; want "+
			"between nothing and the edge's", below, edge)
	}
}

func TestLightGiantsDoNotInflate(t *testing.T) {
	if got := inflation(0.15, 1e9); got != 0 {
		t.Errorf("a light giant inflated by %v", got)
	}
}

func TestNoSmallStepInMassJumpsTheRadius(t *testing.T) {
	const step, most = 1.01, 0.03
	for _, flux := range []float64{1e4, 1e5, 4e5, 1e6, 1e7, 1e8, 1e9} {
		for mass := 0.1; mass < 10; mass *= step {
			radius := func(jupiters float64) float64 {
				return coldGiantRadius(jupiters*jupiterMass/EarthMass,
					0.1*jupiters*jupiterMass/EarthMass, 1, 4.5) +
					inflation(jupiters, flux)
			}
			low, high := radius(mass), radius(mass*step)
			if math.Abs(high-low) > most*low {
				t.Errorf("flux %g, %.3f to %.3f Jupiter masses: "+
					"%.3f to %.3f radii", flux, mass, mass*step, low, high)
			}
		}
	}
}

func TestHotJupiterTemplateIsTheRelationsRadius(t *testing.T) {
	hot := likeJupiter()
	hot.Name = "a hot jupiter"
	hot.Orbit = Orbit{Year: hotJupiterYear}
	body, err := template.HotJupiter()
	if err != nil {
		t.Fatal(err)
	}
	given := kilometresAt(t, hot)
	t.Logf("template %.0f km, relation %.0f km", body.Radius(), given)
	if !near(body.Radius(), given, 0.01) {
		t.Errorf("template radius %.0f km, the relation gives %.0f km at "+
			"its orbit, to a hundredth", body.Radius(), given)
	}
}
