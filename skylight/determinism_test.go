package skylight

import (
	"reflect"
	"testing"
)

// The air the walk is repeated in.
const (
	mixedGas        = 0.2  // picked: Rayleigh depth of the low shell
	mixedParticles  = 0.5  // picked: a haze's depth beside it
	mixedAlbedo     = 0.9  // picked: a haze that absorbs a little
	mixedAsymmetry  = 0.6  // picked: a haze that leans forward
	mixedHigh       = 0.05 // picked: Rayleigh depth of the high shell
	mixedGround     = 0.2  // picked: a ground that sends some back
	repeatPhotons   = 3001 // picked: not a multiple of the workers
	repeatWorkers   = 3    // picked: three, so the remainder is shared
	anotherSeedFrom = 1    // picked: the next seed
)

// mixed is a column with gas and particles in its low shell and gas
// above, so each kind of scattering is drawn.
func mixed() Column {
	low := Shell{Top: lowTop, GasScattering: mixedGas / lowTop,
		ParticleExtinction: mixedParticles / lowTop,
		ParticleAlbedo:     mixedAlbedo, ParticleAsymmetry: mixedAsymmetry}
	high := Shell{Bottom: lowTop, Top: airTop,
		GasScattering: mixedHigh / (airTop - lowTop)}
	return Column{Radius: earthRadius, Shells: []Shell{low, high},
		GroundAlbedo: mixedGround}
}

// repeat walks the mixed column from a seed with three workers.
func repeat(t *testing.T, seed uint64) (*Field, Budget) {
	t.Helper()
	field, spent, err := Solve(mixed(), keptGrid(), repeatPhotons, seed,
		repeatWorkers)
	if err != nil {
		t.Fatal(err)
	}
	return field, spent
}

// the same seed and workers give the same field and budget, to the bit.
func TestTheSameSeedGivesTheSameSky(t *testing.T) {
	first, firstSpent := repeat(t, seedOf)
	second, secondSpent := repeat(t, seedOf)
	if !reflect.DeepEqual(first, second) || firstSpent != secondSpent {
		t.Errorf("one seed, two skies: %+v and %+v", firstSpent,
			secondSpent)
	}
}

// the anti-test: another seed gives another field and budget, so the
// check would see a walk that ignored its seed.
func TestAnotherSeedGivesAnotherSky(t *testing.T) {
	first, firstSpent := repeat(t, seedOf)
	second, secondSpent := repeat(t, seedOf+anotherSeedFrom)
	if reflect.DeepEqual(first.radiance, second.radiance) ||
		firstSpent == secondSpent {
		t.Errorf("two seeds, one sky: %+v", firstSpent)
	}
}
