package airlight

import (
	"slices"
	"testing"

	"github.com/janearc/slayer/airlight/mocks"
	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/skylight"
	"go.uber.org/mock/gomock"
)

// The thin air the plan and vacuum tests walk: one shell over a planet,
// lit by a flat sun.
const (
	thinName    = "thin"
	thinRadius  = 6.0e6 // picked: a planet's size
	thinTop     = 1e5   // picked: an air a hundred kilometres deep
	thinDepth   = 0.1   // picked: about Earth's Rayleigh depth at 550 nm
	thinSun     = 1.5   // picked: a sun of so many watts a nanometre
	thinGround  = 250   // picked: a ground a little above the floor
	fewPhotons  = 2000  // picked: enough to light a bin, in a moment
	otherName   = "other"
	otherBand   = 10 // picked: a band unlike the plan's
	otherSolver = "skylight 0, airlight 0"
)

// testKind is the kind the key tests' keys are of; nothing is kept
// under it.
var testKind = diskcache.MustKind("airlight-test-keys", testCap,
	"the key tests keep nothing")

// testCap is the test kind's cap, in bytes.
const testCap = 1 // picked: nothing is kept

// thinAir is a generated mock of an air of one shell, its gas
// scattering so much a metre, at every wavelength.
func thinAir(t *testing.T, scattering float64) *mocks.MockAir {
	air := mocks.NewMockAir(gomock.NewController(t))
	air.EXPECT().Name().Return(thinName).AnyTimes()
	air.EXPECT().Column(gomock.Any()).Return(skylight.Column{
		Radius: thinRadius, Shells: []skylight.Shell{{Top: thinTop,
			GasScattering: scattering}}}, nil).AnyTimes()
	air.EXPECT().Sun(gomock.Any()).Return(thinSun, nil).AnyTimes()
	return air
}

// thinKnown is the thin air scattering so much a metre, walked with so
// many photons on Venus's bins.
func thinKnown(t *testing.T, scattering float64, photons int) Known {
	t.Helper()
	return Known{Air: thinAir(t, scattering), Photons: photons,
		Views: venusViews, Azimuths: venusAzimuths}
}

// thinPlan is the plan of the thin air over its ground.
func thinPlan(t *testing.T) Plan {
	t.Helper()
	plan, err := NewPlan(thinKnown(t, thinDepth/thinTop, fewPhotons),
		thinGround)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

// a plan holds the air's own bins, Earth's finer than Venus's.
func TestAPlanHoldsTheAirsBins(t *testing.T) {
	known := thinKnown(t, thinDepth/thinTop, fewPhotons)
	known.Views, known.Azimuths = earthViews, earthAzimuths
	plan, err := NewPlan(known, thinGround)
	if err != nil || plan.grid.Views != earthViews ||
		plan.grid.Azimuths != earthAzimuths {
		t.Errorf("the plan's bins %+v, %v", plan.grid, err)
	}
}

// the anti-test: an odd count of view bins is refused, since its middle
// bin would hold the light from above and below the horizon in one.
func TestAnOddCountOfViewBinsIsRefused(t *testing.T) {
	known := thinKnown(t, thinDepth/thinTop, fewPhotons)
	known.Views = venusViews + one
	if _, err := NewPlan(known, thinGround); err == nil {
		t.Errorf("%d view bins were taken", known.Views)
	}
}

// keyName is the name of a plan's key, or the test stopped where the
// key is refused.
func keyName(t *testing.T, plan Plan) string {
	t.Helper()
	key := plan.Keyed(diskcache.NewKey(testKind))
	if err := key.Err(); err != nil {
		t.Fatal(err)
	}
	return key.Name()
}

// two plans of the same air, ground, photons and the rest have the same
// key, and the plan holds every wavelength of the span, each column cut
// at the ground.
func TestTheSamePlanKeysAlike(t *testing.T) {
	first, second := thinPlan(t), thinPlan(t)
	if keyName(t, first) != keyName(t, second) {
		t.Error("one plan made twice keys two ways")
	}
	waves := len(first.wavelengths)
	if first.wavelengths[none] != firstNanometre ||
		first.wavelengths[waves-one] != lastNanometre ||
		len(first.columns) != waves || len(first.sun) != waves {
		t.Errorf("the plan's wavelengths %v", first.wavelengths)
	}
	if got := first.columns[none].Radius; got != thinRadius+thinGround {
		t.Errorf("the plan's ground stands at radius %g", got)
	}
}

// changed is a plan with its own copies of every slice, so a change to
// it leaves the plan it came from as it was.
func changed(plan Plan) Plan {
	plan.wavelengths = slices.Clone(plan.wavelengths)
	plan.sun = slices.Clone(plan.sun)
	plan.columns = slices.Clone(plan.columns)
	for index := range plan.columns {
		plan.columns[index].Shells = slices.Clone(plan.columns[index].Shells)
	}
	plan.grid.Heights = slices.Clone(plan.grid.Heights)
	return plan
}

// the anti-tests, one an input: each changed alone moves the key.
func TestEachInputMovesTheKey(t *testing.T) {
	plain := thinPlan(t)
	before := keyName(t, plain)
	for name, change := range map[string]func(*Plan){
		"the air's name": func(plan *Plan) { plan.name = otherName },
		"a column's number": func(plan *Plan) {
			plan.columns[none].Shells[none].GasScattering *= two
		},
		"the sun":            func(plan *Plan) { plan.sun[none] *= two },
		"the ground":         func(plan *Plan) { plan.ground++ },
		"the wavelengths":    func(plan *Plan) { plan.wavelengths[none]++ },
		"the band":           func(plan *Plan) { plan.band = otherBand },
		"the zenith bins":    func(plan *Plan) { plan.grid.Zeniths++ },
		"the view bins":      func(plan *Plan) { plan.grid.Views++ },
		"the azimuth bins":   func(plan *Plan) { plan.grid.Azimuths++ },
		"the kept height":    func(plan *Plan) { plan.grid.Heights[none]++ },
		"the photons":        func(plan *Plan) { plan.photons++ },
		"the seed":           func(plan *Plan) { plan.seed++ },
		"the workers":        func(plan *Plan) { plan.workers++ },
		"the solver version": func(plan *Plan) { plan.version = otherSolver },
	} {
		made := changed(plain)
		change(&made)
		if keyName(t, made) == before {
			t.Errorf("a change to %s keeps the key", name)
		}
		if keyName(t, plain) != before {
			t.Fatalf("a change to %s moved the plan it came from", name)
		}
	}
}
