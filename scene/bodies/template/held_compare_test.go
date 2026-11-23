package template

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// the walk that holds a body to a literal written beside it. a number
// is held to a part in 1e12, which a product of two typed numbers can
// differ by and a changed figure cannot; a Duration to a microsecond,
// since hours truncates to whole nanoseconds.
//
// a function is held only to being there or not, and a maths.Light not
// at all unless asked. the body tests hold no colour: nothing the
// mutants change reaches one.

// reading is one walk: the places where got and want part.
type reading struct {
	lights bool
	apart  []string
}

// walk compares got to want, noting each place they differ under path.
func (r *reading) walk(path string, got, want reflect.Value) {
	if got.IsValid() != want.IsValid() {
		r.note(path, "one is there and the other is not")
		return
	}
	if !got.IsValid() {
		return
	}
	if got.Type() != want.Type() {
		gotType := got.Type()
		wantType := want.Type()
		r.note(path, fmt.Sprintf("is a %v, want a %v", gotType, wantType))
		return
	}
	if got.Type() == reflect.TypeOf(maths.Light{}) && !r.lights {
		return
	}
	switch got.Kind() {
	case reflect.Func:
		if got.IsNil() != want.IsNil() {
			gotNil := got.IsNil()
			wantNil := want.IsNil()
			r.note(path, fmt.Sprintf("func is nil: %v, want nil: %v", gotNil, wantNil))
		}
	case reflect.Pointer, reflect.Interface:
		if got.IsNil() != want.IsNil() {
			gotNil := got.IsNil()
			wantNil := want.IsNil()
			r.note(path, fmt.Sprintf("is nil: %v, want nil: %v", gotNil, wantNil))
			return
		}
		if !got.IsNil() {
			r.walk(path, got.Elem(), want.Elem())
		}
	case reflect.Struct:
		for i := range got.NumField() {
			r.walk(path+"."+got.Type().Field(i).Name, got.Field(i), want.Field(i))
		}
	case reflect.Slice, reflect.Array:
		if got.Len() != want.Len() {
			gotLength := got.Len()
			wantLength := want.Len()
			r.note(path, fmt.Sprintf("has %d, want %d", gotLength, wantLength))
			return
		}
		for i := range got.Len() {
			r.walk(fmt.Sprintf("%s[%d]", path, i), got.Index(i), want.Index(i))
		}
	case reflect.Map:
		if got.Len() != want.Len() {
			gotLength := got.Len()
			wantLength := want.Len()
			r.note(path, fmt.Sprintf("has %d, want %d", gotLength, wantLength))
			return
		}
		for _, k := range want.MapKeys() {
			g := got.MapIndex(k)
			if !g.IsValid() {
				r.note(fmt.Sprintf("%s[%v]", path, k), "is missing")
				continue
			}
			r.walk(fmt.Sprintf("%s[%v]", path, k), g, want.MapIndex(k))
		}
	case reflect.Float32, reflect.Float64:
		g, w := got.Float(), want.Float()
		if !(math.Abs(g-w) <= 1e-12*math.Abs(w)) {
			r.note(path, fmt.Sprintf("is %v, want %v", g, w))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		g, w := got.Int(), want.Int()
		if got.Type() == reflect.TypeOf(time.Duration(0)) {
			if d := time.Duration(g - w); d < -time.Microsecond || d > time.Microsecond {
				gotDuration := time.Duration(g)
				wantDuration := time.Duration(w)
				r.note(path, fmt.Sprintf("is %v, want %v", gotDuration, wantDuration))
			}
		} else if g != w {
			r.note(path, fmt.Sprintf("is %d, want %d", g, w))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if got.Uint() != want.Uint() {
			gotUint := got.Uint()
			wantUint := want.Uint()
			r.note(path, fmt.Sprintf("is %d, want %d", gotUint, wantUint))
		}
	case reflect.String:
		if got.String() != want.String() {
			gotString := got.String()
			wantString := want.String()
			r.note(path, fmt.Sprintf("is %q, want %q", gotString, wantString))
		}
	case reflect.Bool:
		if got.Bool() != want.Bool() {
			gotBool := got.Bool()
			wantBool := want.Bool()
			r.note(path, fmt.Sprintf("is %v, want %v", gotBool, wantBool))
		}
	default:
		gotKind := got.Kind()
		r.note(path, fmt.Sprintf("is of a kind, %v, this walk does not read", gotKind))
	}
}

// note records one place the two part.
func (r *reading) note(path, what string) {
	r.apart = append(r.apart, strings.TrimPrefix(path, ".")+" "+what)
}

// apart is every place got and want part, none when they agree. colours
// are left out.
func apart(got, want any) []string {
	r := &reading{}
	r.walk("", reflect.ValueOf(got), reflect.ValueOf(want))
	return r.apart
}

// apartWithColours is apart with each colour held too.
func apartWithColours(got, want any) []string {
	r := &reading{lights: true}
	r.walk("", reflect.ValueOf(got), reflect.ValueOf(want))
	return r.apart
}

// holds fails the test for each place got parts from want.
func holds(t *testing.T, what string, got, want any) {
	t.Helper()
	for _, a := range apart(got, want) {
		t.Errorf("%s: %s", what, a)
	}
}

// walked is a body built to order for the walk's own test: every knob
// is a constructor argument, so each case below differs from the base
// in exactly one spelled way.
type walked struct {
	name     string
	radiusKm float64
	day      time.Duration
	air      bool
	featured bool
	mooned   bool
	glow     maths.Light
	gives    float64
}

// base is the walked body every case bends one knob of.
func base() walked {
	return walked{name: "w", radiusKm: 100, day: time.Hour, air: true,
		featured: true, mooned: true, gives: 1}
}

// build is the walked body, through the doors.
func (knobs walked) build(t *testing.T) bodies.Body {
	t.Helper()
	ground := wantGround(t, world.Terrain{Seed: 1, Plates: 3},
		world.Look{Lines: 0.1})
	if knobs.featured {
		ground = ground.Featured(anyFeatures)
	}
	parts := []bodies.Part{ground}
	if knobs.air {
		parts = append(parts, wantAir(t, 8, 0.01))
	}
	if knobs.mooned {
		moonPart := wantMoon(t, want(t, "m", bodies.Rock, 1, 1), 100, 0)
		parts = append(parts, moonPart)
	}
	return want(t, knobs.name, bodies.Rock, knobs.radiusKm, 1e20,
		parts...).
		Turning(knobs.day, measure.Radians(0.5)).
		Glowing(knobs.glow, thing.Spectrum{
			Gives: map[radiation.Range]float64{
				radiation.XRay(): knobs.gives}})
}

// the walk sees what it should: a changed number, a part gone, a moon
// dropped, a function missing, a map entry changed, a name changed, a
// Duration out by more than a microsecond.
//
// it does not see a number out by one part in 1e13, a Duration out by
// a nanosecond, or a colour, unless colours are asked for.
func TestTheWalkSeesAChangeAndNotARounding(t *testing.T) {
	if got := apart(base().build(t), base().build(t)); len(got) != 0 {
		t.Errorf("the same body differs: %v", got)
	}
	bend := func(change func(*walked)) bodies.Body {
		knobs := base()
		change(&knobs)
		return knobs.build(t)
	}
	for _, c := range []struct {
		what   string
		change func(*walked)
	}{
		{"a radius", func(k *walked) { k.radiusKm *= 1 + 1e-9 }},
		{"a part gone", func(k *walked) { k.air = false }},
		{"a moon dropped", func(k *walked) { k.mooned = false }},
		{"a function missing", func(k *walked) { k.featured = false }},
		{"a day out by two microseconds", func(k *walked) { k.day += 2000 }},
		{"a name", func(k *walked) { k.name += "!" }},
		{"a map entry", func(k *walked) { k.gives = 2 }},
	} {
		seen := apart(bend(c.change), base().build(t))
		if len(seen) == 0 {
			t.Errorf("the walk does not see %s", c.what)
		}
	}
	for _, c := range []struct {
		what   string
		change func(*walked)
	}{
		{"a rounding", func(k *walked) { k.radiusKm *= 1 + 1e-13 }},
		{"a nanosecond", func(k *walked) { k.day++ }},
		{"a colour", func(k *walked) { k.glow = maths.Grey(1) }},
	} {
		if got := apart(bend(c.change), base().build(t)); len(got) != 0 {
			t.Errorf("the walk sees %s: %v", c.what, got)
		}
	}
	coloured := bend(func(k *walked) { k.glow = maths.Grey(1) })
	seenWithColours := apartWithColours(coloured, base().build(t))
	if len(seenWithColours) == 0 {
		t.Error("the walk with colours does not see a colour")
	}
}
