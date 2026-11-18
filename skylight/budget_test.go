package skylight

import (
	"fmt"
	"math"
	"testing"
)

// The air the budgets are walked through.
const (
	thinDepth      = 0.1   // picked: about Earth's Rayleigh depth in green
	lowShare       = 0.6   // picked: most of the depth low, as in real air
	thinGround     = 0.3   // picked: about Earth's mean albedo
	thickDepth     = 30    // brief: the cloud's optical depth
	thickAlbedo    = 0.999 // brief: the cloud's single-scattering albedo
	thickAsymmetry = 0.7   // brief: the cloud's asymmetry
	thickTop       = 1e4   // picked: ten kilometres of cloud
	thickGround    = 0.2   // picked: some light back from below the cloud
	thickPhotons   = 4000  // picked: a few hundred events each, a second
	darkDepth      = 0.3   // picked: a hazy sky's depth
	darkAbsorbed   = 0.05  // picked: a little gas absorption with it
	darkAbsorber   = 0.02  // picked: a shell that only absorbs
	whiteDepth     = 0.5   // picked: enough air to send light back down
	bareGround     = 0.5   // picked: a grey ground under no air
	cappedDepth    = 50    // picked: deep enough that photons stay in
	cappedMost     = 5     // picked: a cap most photons reach
)

// a budgetCase is a column whose budget must balance, walked with a cap
// on events per photon.
type budgetCase struct {
	name    string
	column  Column
	photons int
	most    int
}

// thinRayleigh is Earth-like air: two shells of Rayleigh gas over a
// ground of Earth's albedo.
func thinRayleigh() Column {
	low := Shell{Top: lowTop,
		GasScattering: lowShare * thinDepth / lowTop}
	high := Shell{Bottom: lowTop, Top: airTop,
		GasScattering: (one - lowShare) * thinDepth / (airTop - lowTop)}
	return Column{Radius: earthRadius, Shells: []Shell{low, high},
		GroundAlbedo: thinGround}
}

// darkGround is hazy air that absorbs a little, under a shell that only
// absorbs, over a black ground.
func darkGround() Column {
	low := Shell{Top: lowTop, GasScattering: darkDepth / lowTop,
		GasAbsorption: darkAbsorbed / lowTop}
	high := Shell{Bottom: lowTop, Top: airTop,
		GasAbsorption: darkAbsorber / (airTop - lowTop)}
	return Column{Radius: earthRadius, Shells: []Shell{low, high}}
}

// budgetCases are the brief's five airs and one with a cap low enough
// that most photons reach it.
func budgetCases() []budgetCase {
	thick := Shell{ParticleExtinction: thickDepth / thickTop,
		ParticleAlbedo: thickAlbedo, ParticleAsymmetry: thickAsymmetry}
	white := Shell{GasScattering: whiteDepth / airTop}
	trap := Shell{ParticleExtinction: cappedDepth / thickTop,
		ParticleAlbedo: one}
	return []budgetCase{
		{"thin rayleigh", thinRayleigh(), fewPhotons, mostEvents},
		{"thick particles", uniformColumn(earthRadius, thickTop, thick,
			thickGround), thickPhotons, mostEvents},
		{"dark ground", darkGround(), fewPhotons, mostEvents},
		{"white ground", uniformColumn(earthRadius, airTop, white, one),
			fewPhotons, mostEvents},
		{"no air", uniformColumn(earthRadius, airTop, Shell{},
			bareGround), fewPhotons, mostEvents},
		{"capped", uniformColumn(earthRadius, thickTop, trap, one),
			thickPhotons, cappedMost},
	}
}

// conserved is nil when the budget's ends add to what was launched,
// within what Russian roulette can move them, and none is negative.
//
// Every step but roulette moves power from a photon to one end, exactly.
// Roulette at a weight w under the lightest kept takes w or adds it,
// with even odds: a change of mean nought and of variance w squared, so
// under the lightest weight squared.
//
// A photon dies at its first lost toss, so on average it meets roulette
// at most twice. Over n photons of power p the sum's standard deviation
// is at most the root of 2n, times the lightest weight, times p.
func conserved(spent Budget, photons int) error {
	ended := spent.Escaped + spent.Air + spent.Ground + spent.Capped
	power := spent.Launched / float64(photons)
	deviation := math.Sqrt(two*float64(photons)) * lightest * power
	allowed := sigmas*deviation + rounding*spent.Launched
	if !(math.Abs(ended-spent.Launched) <= allowed) {
		return fmt.Errorf("%.9g ended of %.9g launched, allowed %.3g",
			ended, spent.Launched, allowed)
	}
	if min(spent.Escaped, spent.Air, spent.Ground, spent.Capped) < none {
		return fmt.Errorf("a negative end in %+v", spent)
	}
	return nil
}

// rounding is the share of a sum its rounding may move: far above what
// millions of terms can, far below any physics.
const rounding = 1e-12 // picked: see above

// every photon is accounted for: what escapes, what the air and the
// ground absorb and what is capped add to what came in, for thin air,
// a thick cloud, a black ground, a white one, none, and a low cap.
func TestBudgetsBalance(t *testing.T) {
	for _, each := range budgetCases() {
		_, spent, _, err := solve(each.column, groundGrid(),
			each.photons, seedOf, workersOf, each.most)
		if err != nil {
			t.Fatalf("%s: %v", each.name, err)
		}
		if err := conserved(spent, each.photons); err != nil {
			t.Errorf("%s: %v", each.name, err)
		}
		if each.most < mostEvents && !(spent.Capped > none) {
			t.Errorf("%s: nothing was capped", each.name)
		}
		t.Logf("%s: %+v", each.name, spent)
	}
}

// the anti-test: a budget that forgets what the black ground absorbed
// does not balance, so the check would catch an end left uncounted.
func TestAnUncountedEndIsCaught(t *testing.T) {
	_, spent, err := Solve(darkGround(), groundGrid(), fewPhotons, seedOf,
		workersOf)
	if err != nil {
		t.Fatal(err)
	}
	spent.Ground = none
	if conserved(spent, fewPhotons) == nil {
		t.Errorf("a budget without its ground balances: %+v", spent)
	}
}
