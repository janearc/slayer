package template

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
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
		r.note(path, fmt.Sprintf("is a %v, want a %v", got.Type(), want.Type()))
		return
	}
	if got.Type() == reflect.TypeOf(maths.Light{}) && !r.lights {
		return
	}
	switch got.Kind() {
	case reflect.Func:
		if got.IsNil() != want.IsNil() {
			r.note(path, fmt.Sprintf("func is nil: %v, want nil: %v", got.IsNil(), want.IsNil()))
		}
	case reflect.Pointer, reflect.Interface:
		if got.IsNil() != want.IsNil() {
			r.note(path, fmt.Sprintf("is nil: %v, want nil: %v", got.IsNil(), want.IsNil()))
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
			r.note(path, fmt.Sprintf("has %d, want %d", got.Len(), want.Len()))
			return
		}
		for i := range got.Len() {
			r.walk(fmt.Sprintf("%s[%d]", path, i), got.Index(i), want.Index(i))
		}
	case reflect.Map:
		if got.Len() != want.Len() {
			r.note(path, fmt.Sprintf("has %d, want %d", got.Len(), want.Len()))
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
				r.note(path, fmt.Sprintf("is %v, want %v", time.Duration(g), time.Duration(w)))
			}
		} else if g != w {
			r.note(path, fmt.Sprintf("is %d, want %d", g, w))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if got.Uint() != want.Uint() {
			r.note(path, fmt.Sprintf("is %d, want %d", got.Uint(), want.Uint()))
		}
	case reflect.String:
		if got.String() != want.String() {
			r.note(path, fmt.Sprintf("is %q, want %q", got.String(), want.String()))
		}
	case reflect.Bool:
		if got.Bool() != want.Bool() {
			r.note(path, fmt.Sprintf("is %v, want %v", got.Bool(), want.Bool()))
		}
	default:
		r.note(path, fmt.Sprintf("is of a kind, %v, this walk does not read", got.Kind()))
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

// the walk sees what it should: a changed number, a part gone, a moon
// dropped, a function missing, a map entry changed, a name changed, a
// Duration out by more than a microsecond.
//
// it does not see a number out by one part in 1e13, a Duration out by
// a nanosecond, or a colour, unless colours are asked for.
func TestTheWalkSeesAChangeAndNotARounding(t *testing.T) {
	if got := apart(Venus(), Venus()); len(got) != 0 {
		t.Errorf("the same body differs: %v", got)
	}
	for _, c := range []struct {
		what   string
		of     func() bodies.Body
		change func(b *bodies.Body)
	}{
		{"a radius", Venus, func(b *bodies.Body) { b.Radius *= 1 + 1e-9 }},
		{"a part gone", Venus, func(b *bodies.Body) { b.Air = nil }},
		{"a moon dropped", Earth, func(b *bodies.Body) { b.Moons = nil }},
		{"a function missing", Venus, func(b *bodies.Body) { b.Ground.Features = nil }},
		{"a day out by two microseconds", Venus, func(b *bodies.Body) { b.Day += 2000 }},
		{"a name", Venus, func(b *bodies.Body) { b.Name += "!" }},
		{"a map entry", Venus, func(b *bodies.Body) {
			scribble(reflect.ValueOf(&b.Ground.Radiant).Elem())
		}},
	} {
		b := c.of()
		c.change(&b)
		if len(apart(b, c.of())) == 0 {
			t.Errorf("the walk does not see %s", c.what)
		}
	}
	for _, c := range []struct {
		what   string
		change func(b *bodies.Body)
	}{
		{"a rounding", func(b *bodies.Body) { b.Radius *= 1 + 1e-13 }},
		{"a nanosecond", func(b *bodies.Body) { b.Day++ }},
		{"a colour", func(b *bodies.Body) { b.Air.Colour.R++ }},
	} {
		b := Venus()
		c.change(&b)
		if got := apart(b, Venus()); len(got) != 0 {
			t.Errorf("the walk sees %s: %v", c.what, got)
		}
	}
	b := Venus()
	b.Air.Colour.R++
	if len(apartWithColours(b, Venus())) == 0 {
		t.Error("the walk with colours does not see a colour")
	}
}
