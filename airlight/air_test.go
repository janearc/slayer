package airlight

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/airs"
	"github.com/janearc/slayer/skylight"
)

// The numbers of the column the mapping is checked on: each different,
// so a number put in another's place is seen.
const (
	mappedRadius     = 6.0e6 // picked: a planet's size
	mappedAlbedo     = 0.06  // picked: a dark ground
	mappedBottom     = 100   // picked: metres
	mappedTop        = 200   // picked: metres
	mappedScattering = 3e-5  // picked: a metre
	mappedAbsorption = 4e-6  // picked: a metre
	mappedExtinction = 5e-4  // picked: a metre
	mappedParticle   = 0.6   // picked: an albedo
	mappedAsymmetry  = 0.7   // picked: a lean forward
	mappedNudge      = 0.01  // picked: a change no field holds already
	unknownAir       = "titan"
	venusName        = "venus"
	somewhereVisible = 550 // picked: nanometres, in the middle
)

// mappedColumn is a column of package airs with every number different.
func mappedColumn() airs.Column {
	return airs.Column{Radius: mappedRadius, GroundAlbedo: mappedAlbedo,
		Layers: []airs.Layer{{Bottom: mappedBottom, Top: mappedTop,
			GasScattering: mappedScattering, GasAbsorption: mappedAbsorption,
			ParticleExtinction: mappedExtinction,
			ParticleAlbedo:     mappedParticle,
			ParticleAsymmetry:  mappedAsymmetry}}}
}

// every number of package airs's column reaches the solver's in its
// own place.
func TestTheAdapterMapsEveryField(t *testing.T) {
	want := skylight.Column{Radius: mappedRadius, GroundAlbedo: mappedAlbedo,
		Shells: []skylight.Shell{{Bottom: mappedBottom, Top: mappedTop,
			GasScattering: mappedScattering, GasAbsorption: mappedAbsorption,
			ParticleExtinction: mappedExtinction,
			ParticleAlbedo:     mappedParticle,
			ParticleAsymmetry:  mappedAsymmetry}}}
	if got := mapped(mappedColumn()); !reflect.DeepEqual(got, want) {
		t.Errorf("mapped %+v, want %+v", got, want)
	}
}

// the anti-test: each number of a layer, and the radius and the albedo,
// changed alone, changes the mapped column, so none is dropped.
func TestEachNumberChangesTheMapping(t *testing.T) {
	plain := mapped(mappedColumn())
	layerType := reflect.TypeFor[airs.Layer]()
	for field := range layerType.NumField() {
		column := mappedColumn()
		number := reflect.ValueOf(&column.Layers[none]).Elem().Field(field)
		number.SetFloat(number.Float() + mappedNudge)
		if reflect.DeepEqual(mapped(column), plain) {
			t.Errorf("a change to %s is lost", layerType.Field(field).Name)
		}
	}
	for name, change := range map[string]func(*airs.Column){
		"radius": func(column *airs.Column) { column.Radius++ },
		"albedo": func(column *airs.Column) {
			column.GroundAlbedo += mappedNudge
		},
	} {
		column := mappedColumn()
		change(&column)
		if reflect.DeepEqual(mapped(column), plain) {
			t.Errorf("a change to the %s is lost", name)
		}
	}
}

// sameNumbers is nil when two struct types hold the same fields by name
// and type, and otherwise says one that only one of them holds.
func sameNumbers(first, second reflect.Type) error {
	held := func(kind reflect.Type) map[string]reflect.Type {
		made := map[string]reflect.Type{}
		for index := range kind.NumField() {
			made[kind.Field(index).Name] = kind.Field(index).Type
		}
		return made
	}
	these, those := held(first), held(second)
	for _, name := range slices.Sorted(maps.Keys(these)) {
		if those[name] != these[name] {
			return fmt.Errorf("%s holds %s and %s does not", first, name,
				second)
		}
	}
	if len(these) != len(those) {
		return fmt.Errorf("%s holds a number %s does not", second, first)
	}
	return nil
}

// package airs's layers and the solver's shells hold the same numbers,
// so a layer maps to a shell with nothing left over.
func TestTheShellsHoldWhatTheLayersHold(t *testing.T) {
	if err := sameNumbers(reflect.TypeFor[airs.Layer](),
		reflect.TypeFor[skylight.Shell]()); err != nil {
		t.Error(err)
	}
}

// the anti-test: a layer with a number the shell has not, as an ozone
// absorption would be, is caught.
func TestANumberOnlyOneSideHoldsIsCaught(t *testing.T) {
	type withOzone struct {
		Bottom, Top                       float64
		GasScattering, GasAbsorption      float64
		ParticleExtinction                float64
		ParticleAlbedo, ParticleAsymmetry float64
		Ozone                             float64
	}
	if sameNumbers(reflect.TypeFor[skylight.Shell](),
		reflect.TypeFor[withOzone]()) == nil {
		t.Error("a layer with an ozone absorption maps to a shell")
	}
}

// The flat sun the tests that need no measured spectrum are lit by, in
// the spectrum's own form.
const (
	sunHeading = "wavelength (nm),irradiance (W/m^2/nm)"
	flatFirst  = 300 // picked: below the first wavelength's band
	flatLast   = 900 // picked: above the last's
	flatPower  = 1.5 // picked: watts a square metre a nanometre
	earthName  = "earth"
)

// flatSun is a sun of flatPower at every whole nanometre from flatFirst
// to flatLast, read as the shelf's spectrum is.
func flatSun(t *testing.T) airs.Sun {
	t.Helper()
	var text strings.Builder
	text.WriteString(sunHeading + "\n")
	for nanometres := flatFirst; nanometres <= flatLast; nanometres++ {
		fmt.Fprintf(&text, "%d,%g\n", nanometres, flatPower)
	}
	sun, err := airs.ReadSun(strings.NewReader(text.String()))
	if err != nil {
		t.Fatal(err)
	}
	return sun
}

// an air no set may name is refused, and the refusal names the airs
// there are.
func TestAnUnknownAirIsRefusedNamingTheKnown(t *testing.T) {
	_, err := Named(unknownAir, flatSun(t))
	if err == nil || !strings.Contains(err.Error(), venusName) ||
		!strings.Contains(err.Error(), earthName) ||
		!strings.Contains(err.Error(), unknownAir) {
		t.Errorf("an unknown air: %v", err)
	}
}

// the anti-test: Venus is known, by its name, with its photons, its
// bins, its column from package airs and the sun at its top.
func TestVenusIsKnown(t *testing.T) {
	known, err := Named(venusName, flatSun(t))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(Names(), venusName) ||
		known.Air.Name() != venusName || known.Photons != venusPhotons ||
		known.Views != venusViews || known.Azimuths != venusAzimuths {
		t.Errorf("venus is %+v among %v", known, Names())
	}
	column, err := known.Air.Column(somewhereVisible)
	if err != nil || len(column.Shells) == none || !(column.Radius > none) {
		t.Errorf("venus's column at %d nm: %+v, %v", somewhereVisible,
			column.Radius, err)
	}
	if sun, err := known.Air.Sun(somewhereVisible); err != nil ||
		!(sun > none) {
		t.Errorf("the sun at venus: %g, %v", sun, err)
	}
}

// Earth is known, by its name, with its photons, its bins and its
// column from package airs, and the sun at its top is the spectrum's
// own, at one astronomical unit.
func TestEarthIsKnown(t *testing.T) {
	known, err := Named(earthName, flatSun(t))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(Names(), earthName) ||
		known.Air.Name() != earthName || known.Photons != earthPhotons ||
		known.Views != earthViews || known.Azimuths != earthAzimuths {
		t.Errorf("earth is %+v among %v", known, Names())
	}
	column, err := known.Air.Column(somewhereVisible)
	if err != nil || len(column.Shells) == none || !(column.Radius > none) {
		t.Errorf("earth's column at %d nm: %+v, %v", somewhereVisible,
			column.Radius, err)
	}
	if sun, err := known.Air.Sun(somewhereVisible); err != nil ||
		math.Abs(sun-flatPower) > handShare {
		t.Errorf("the sun at earth: %g, %v, want %g", sun, err, flatPower)
	}
}

// the anti-test: the same sun at Venus is brighter than at Earth, so the
// two airs do not share one distance.
func TestEarthsSunIsFartherThanVenuss(t *testing.T) {
	earth, err := Named(earthName, flatSun(t))
	if err != nil {
		t.Fatal(err)
	}
	venus, err := Named(venusName, flatSun(t))
	if err != nil {
		t.Fatal(err)
	}
	far, farError := earth.Air.Sun(somewhereVisible)
	near, nearError := venus.Air.Sun(somewhereVisible)
	if farError != nil || nearError != nil || !(near > far) {
		t.Errorf("the sun at earth %g, at venus %g: %v, %v", far, near,
			farError, nearError)
	}
}

// The column the ground is cut from: three shells over a planet.
const (
	cutRadius  = 6.0e6 // picked: a planet's size
	cutFirst   = 1000  // picked: the first shell's top, in metres
	cutSecond  = 3000  // picked: the second's
	cutTop     = 10000 // picked: the air's top
	cutGround  = 1500  // picked: a ground in the second shell
	cutBelow   = -200  // picked: a ground below the column's floor
	cutGas     = 1e-5  // picked: the first shell's scattering a metre
	cutLayered = 2e-5  // picked: the second's
	cutHigh    = 3e-5  // picked: the third's
)

// cutColumn is three shells, each its own scattering.
func cutColumn() skylight.Column {
	return skylight.Column{Radius: cutRadius, Shells: []skylight.Shell{
		{Bottom: none, Top: cutFirst, GasScattering: cutGas},
		{Bottom: cutFirst, Top: cutSecond, GasScattering: cutLayered},
		{Bottom: cutSecond, Top: cutTop, GasScattering: cutHigh}}}
}

// a ground in the second shell drops the first, cuts the second there
// and counts every height from it; a ground below the floor stretches
// the first shell down to it.
func TestTheGroundCutsTheColumn(t *testing.T) {
	for _, each := range []struct {
		ground float64
		want   skylight.Column
	}{{cutGround, skylight.Column{Radius: cutRadius + cutGround,
		Shells: []skylight.Shell{
			{Top: cutSecond - cutGround, GasScattering: cutLayered},
			{Bottom: cutSecond - cutGround, Top: cutTop - cutGround,
				GasScattering: cutHigh}}}},
		{cutBelow, skylight.Column{Radius: cutRadius + cutBelow,
			Shells: []skylight.Shell{
				{Top: cutFirst - cutBelow, GasScattering: cutGas},
				{Bottom: cutFirst - cutBelow, Top: cutSecond - cutBelow,
					GasScattering: cutLayered},
				{Bottom: cutSecond - cutBelow, Top: cutTop - cutBelow,
					GasScattering: cutHigh}}}},
	} {
		got, err := grounded(cutColumn(), each.ground)
		if err != nil || !reflect.DeepEqual(got, each.want) {
			t.Errorf("ground %g: %+v, %v", each.ground, got, err)
		}
	}
	if plain := cutColumn(); plain.Shells[none].Top != cutFirst {
		t.Error("cutting changed the column it was given")
	}
}

// the anti-test: a ground at or above the air's top, and a column with
// no shells, are refused.
func TestAGroundWithNoAirAboveIsRefused(t *testing.T) {
	if _, err := grounded(cutColumn(), cutTop); err == nil {
		t.Error("a ground at the air's top was cut")
	}
	_, err := grounded(skylight.Column{Radius: cutRadius}, cutGround)
	if !errors.Is(err, skylight.ErrShells) {
		t.Errorf("a column of no shells: %v", err)
	}
}
