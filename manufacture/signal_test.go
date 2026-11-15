package manufacture

import (
	"testing"
	"time"
)

// A programme for the tests: six seconds, the first of them every light
// lit, then twelve lights chasing two at once, a lap in a second.
func someProgramme(place int) signal {
	return signal{cycle: 6 * time.Second, steady: time.Second,
		lap: time.Second, place: place, places: 12, lit: 2, low: 0.25}
}

// At each moment of a programme a light is lit or low, worked here by
// hand: every light is lit in the steady part; then the lit ones go
// round, two at once, a place every twelfth of a second; and the cycle
// starts again. The anti-test is in each moment of the chase: a light
// whose turn it is not is low, and not lit.
func TestALightKeepsItsPartInAProgramme(t *testing.T) {
	ms := time.Millisecond
	moments := []struct {
		place int
		at    time.Duration
		want  float64
		why   string
	}{
		{0, 0, 1, "the cycle's start: all lit"},
		{7, 500 * ms, 1, "inside the steady part: all lit"},
		{0, 1010 * ms, 1, "the chase begins at place nought"},
		{11, 1010 * ms, 1, "and the one behind it is the second lit"},
		{1, 1010 * ms, 0.25, "the one ahead is not lit yet"},
		{6, 1010 * ms, 0.25, "nor is one across the rim"},
		{1, 1100 * ms, 1, "a twelfth of a second on, the next is lit"},
		{11, 1200 * ms, 0.25, "and the first one behind has gone low"},
		{6, 1510 * ms, 1, "half a lap on, the lit ones are across the rim"},
		{0, 2010 * ms, 1, "a lap on, back at place nought"},
		{0, 6000 * ms, 1, "the cycle starts again: all lit"},
		{6, 7510 * ms, 1, "and the second cycle is as the first"},
		{3, -5500 * ms, 1, "a time before nought is in a cycle too"},
	}
	for _, each := range moments {
		if got := someProgramme(each.place).share(each.at); got != each.want {
			t.Errorf("place %d at %v is %v, want %v: %s", each.place, each.at,
				got, each.want, each.why)
		}
	}
}

// Through the chase two of the twelve are lit at every moment and ten
// are low, whatever the moment. That is its anti-test too: the
// programme never drops a light and never lights a third.
func TestTheChaseKeepsItsCount(t *testing.T) {
	for at := time.Second; at < 6*time.Second; at += 37 * time.Millisecond {
		lit := 0
		for place := range 12 {
			if someProgramme(place).share(at) == 1 {
				lit++
			}
		}
		if lit != 2 {
			t.Fatalf("at %v there are %d lit, want 2", at, lit)
		}
	}
}

// A light with no programme is always lit. The anti-test: a light that
// is never one of the lit is low for the whole chase, as the Larson's
// belly lamp is.
func TestALightOutsideTheChase(t *testing.T) {
	for _, at := range []time.Duration{0, time.Second, 4 * time.Second, time.Hour} {
		if got := (signal{}).share(at); got != 1 {
			t.Errorf("a light with no programme is %v at %v", got, at)
		}
	}
	belly := signal{cycle: 6 * time.Second, steady: time.Second,
		lap: time.Second, places: 1, low: 0.25}
	if got := belly.share(500 * time.Millisecond); got != 1 {
		t.Errorf("in the steady part the belly lamp is %v", got)
	}
	for at := time.Second; at < 6*time.Second; at += 250 * time.Millisecond {
		if got := belly.share(at); got != 0.25 {
			t.Fatalf("in the chase at %v the belly lamp is %v, want low", at, got)
		}
	}
}

// A programme that cannot be kept is refused, and a sound one is taken.
func TestAProgrammeThatCannotBeKeptIsRefused(t *testing.T) {
	sound := someProgramme(3)
	bent := map[string]func(*signal){
		"no cycle":                func(s *signal) { s.cycle = 0 },
		"no lap":                  func(s *signal) { s.lap = 0 },
		"a steady past the cycle": func(s *signal) { s.steady = 7 * time.Second },
		"a steady before nought":  func(s *signal) { s.steady = -time.Second },
		"no places":               func(s *signal) { s.places = 0 },
		"a place past the places": func(s *signal) { s.place = 12 },
		"a place before nought":   func(s *signal) { s.place = -1 },
		"more lit than there are": func(s *signal) { s.lit = 13 },
		"a low above the light":   func(s *signal) { s.low = 1.5 },
		"a low below nought":      func(s *signal) { s.low = -0.1 },
	}
	lamp, err := rimLight()
	if err != nil {
		t.Fatal(err)
	}
	for name, bend := range bent {
		part := sound
		bend(&part)
		if _, err := lamp.signalling(part); err == nil {
			t.Errorf("%s: taken", name)
		}
	}
	taken, err := lamp.signalling(sound)
	if err != nil {
		t.Fatal(err)
	}
	if taken.share(1010*time.Millisecond) != 0.25 || lamp.share(1010*time.Millisecond) != 1 {
		t.Error("the light given a part does not keep it, or the one not given does")
	}
}

// The Larson's own lights keep the programme its numbers give: at the
// start every one is lit; in the chase three running lights are, and
// the belly lamp and the other nine are low. The anti-test: the cabin's
// light is in no chase, and is as lit in it as at the start.
func TestTheLarsonsLightsKeepTheirProgramme(t *testing.T) {
	made := mustMake(t, "mcrn-larson", 5)
	shares := func(at time.Duration) (lit, low int, cabin float64) {
		for _, each := range made.Descriptions() {
			share := each.Light().Share(at)
			switch {
			case each.Name() == cabinName:
				cabin = share
			case share == 1:
				lit++
			default:
				low++
			}
		}
		return lit, low, cabin
	}
	if lit, low, cabin := shares(0); lit != 13 || low != 0 || cabin != 1 {
		t.Errorf("at the start %d are lit and %d low, the cabin %v; want 13, 0, 1",
			lit, low, cabin)
	}
	if lit, low, cabin := shares(3 * time.Second); lit != 3 || low != 10 || cabin != 1 {
		t.Errorf("in the chase %d are lit and %d low, the cabin %v; want 3, 10, 1",
			lit, low, cabin)
	}
}
