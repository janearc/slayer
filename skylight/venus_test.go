package skylight

import (
	"math"
	"sync"
	"testing"
	"time"
)

// A column like Venus's at 550 nm, for the speed measure.
const (
	venusRadius = 6.0518e6 // picked: Venus's mean radius, 6,051.8 km
	cloudBottom = 4.8e4    // brief: the cloud from 48 km
	cloudTop    = 7e4      // brief: to 70 km
	venusTop    = 1e5      // brief: the air to 100 km
	lowShells   = 24       // picked: two kilometres each below the cloud
	cloudShells = 11       // picked: two kilometres each in it
	highShells  = 5        // picked: six kilometres each above it,
	// forty shells in all
	cloudDepth     = 30     // brief: the cloud's optical depth
	cloudAlbedo    = 0.999  // brief
	cloudAsymmetry = 0.75   // brief
	rayleighBelow  = 16     // brief: Rayleigh depth below the cloud
	scaleHeight    = 1.59e4 // picked: about Venus's scale height near
	// the ground, 15.9 km, so the gas thins with height
	venusGround   = 0.06  // brief
	venusPhotons  = 40000 // picked: some seconds of four workers
	venusZeniths  = 36    // brief
	venusViews    = 16    // brief
	venusAzimuths = 8     // brief
	countsWanted  = 100   // brief: a mean of a hundred counts a bin
)

// venusShells are n shells from bottom to top, each holding the gas
// that thins with height over it, and the cloud's share if in it.
func venusShells(bottom, top float64, n int, cloud float64) []Shell {
	surface := rayleighBelow / (scaleHeight *
		-math.Expm1(-cloudBottom/scaleHeight))
	width := (top - bottom) / float64(n)
	shells := make([]Shell, n)
	for index := range shells {
		low := bottom + float64(index)*width
		high := low + width
		depth := surface * scaleHeight * (math.Exp(-low/scaleHeight) -
			math.Exp(-high/scaleHeight))
		shells[index] = Shell{Bottom: low, Top: high,
			GasScattering: depth / width, ParticleExtinction: cloud,
			ParticleAlbedo: cloudAlbedo, ParticleAsymmetry: cloudAsymmetry}
	}
	return shells
}

// venus is the forty shells: clear gas below the cloud, gas and cloud
// in it, clear gas above, over a dark ground.
func venus() Column {
	cloud := cloudDepth / (cloudTop - cloudBottom)
	shells := venusShells(none, cloudBottom, lowShells, none)
	shells = append(shells, venusShells(cloudBottom, cloudTop,
		cloudShells, cloud)...)
	shells = append(shells, venusShells(cloudTop, venusTop, highShells,
		none)...)
	return Column{Radius: venusRadius, Shells: shells,
		GroundAlbedo: venusGround}
}

// a venusRun is the Venus walk's field, budget, count and time.
type venusRun struct {
	field   *Field
	spent   Budget
	counted census
	took    time.Duration
}

// venusWalk is the Venus walk, made once for the test and its anti-test.
var venusWalk = sync.OnceValues(func() (venusRun, error) {
	grid := Grid{Heights: []float64{none}, Zeniths: venusZeniths,
		Views: venusViews, Azimuths: venusAzimuths}
	started := time.Now()
	field, spent, counted, err := solve(venus(), grid, venusPhotons,
		seedOf, workersOf, mostEvents)
	return venusRun{field, spent, counted, time.Since(started)}, err
})

// walkedVenus is the Venus walk, skipped when short.
func walkedVenus(t *testing.T) venusRun {
	t.Helper()
	if testing.Short() {
		t.Skip("the Venus walk takes some seconds")
	}
	run, err := venusWalk()
	if err != nil {
		t.Fatal(err)
	}
	return run
}

// downwardCounts is the mean count over the bins at the ground that
// look up, the downward hemisphere of light.
func downwardCounts(field *Field) float64 {
	total, bins := 0.0, 0.0
	for z := range field.zeniths {
		for v := field.views / two; v < field.views; v++ {
			for a := range field.azimuths {
				total += float64(field.counts[field.index(none, z, v, a)])
				bins++
			}
		}
	}
	return total / bins
}

// the Venus-like column balances its budget, and says how fast it is
// walked: photons a second, the share that reach the ground, the mean
// events a photon, and the photons a hundred counts a bin would take.
func TestVenusBalancesAndIsTimed(t *testing.T) {
	run := walkedVenus(t)
	if err := conserved(run.spent, venusPhotons); err != nil {
		t.Error(err)
	}
	mean := downwardCounts(run.field)
	t.Logf("%.0f photons a second, %.4f reach the ground, %.1f events "+
		"a photon, %.3g counts a bin down, %.3g photons for %d",
		venusPhotons/run.took.Seconds(),
		float64(run.counted.grounded)/venusPhotons,
		float64(run.counted.events)/venusPhotons, mean,
		venusPhotons*countsWanted/mean, countsWanted)
	t.Logf("budget %+v; sun at the ground, overhead: %.3g",
		run.spent, run.field.Transmittance(none, none))
}

// the anti-test: the Venus budget without what the air absorbed does
// not balance, so under a deep cloud the check still sees a lost end.
func TestVenusWithoutItsAirIsCaught(t *testing.T) {
	run := walkedVenus(t)
	spent := run.spent
	spent.Air = none
	if conserved(spent, venusPhotons) == nil {
		t.Errorf("a Venus budget without its air balances: %+v", spent)
	}
}
