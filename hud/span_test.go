package hud

import (
	"image"
	"math"
	"testing"
	"time"
)

// a span holds the times from its start up to, not including, its end;
// each edge is tried on both sides.
func TestASpanHoldsFromItsStartUpToItsEnd(t *testing.T) {
	span := Between(2*time.Second, 5*time.Second)
	for at, want := range map[time.Duration]bool{
		0: false, 2*time.Second - 1: false, 2 * time.Second: true,
		4 * time.Second: true, 5*time.Second - 1: true,
		5 * time.Second: false, time.Hour: false} {
		if span.Holds(at) != want {
			t.Errorf("Between(2s, 5s).Holds(%v) = %v", at, !want)
		}
	}
}

// a span with one end open is open on that end only, and the zero span
// holds every time.
func TestASpanWithAnOpenEnd(t *testing.T) {
	from, until := From(3*time.Second), Until(3*time.Second)
	cases := []struct {
		name string
		span Span
		at   time.Duration
		want bool
	}{
		{"from, before", from, 2 * time.Second, false},
		{"from, at", from, 3 * time.Second, true},
		{"from, long after", from, time.Hour, true},
		{"until, at zero", until, 0, true},
		{"until, before", until, 3*time.Second - 1, true},
		{"until, at", until, 3 * time.Second, false},
		{"zero, at zero", Span{}, 0, true},
		{"zero, long after", Span{}, time.Hour, true},
	}
	for _, each := range cases {
		if each.span.Holds(each.at) != each.want {
			t.Errorf("%s: Holds = %v", each.name, !each.want)
		}
	}
}

// the near miss: a span that ends where it starts, or before, holds no
// time at all.
func TestASpanThatEndsBeforeItStartsHoldsNothing(t *testing.T) {
	for _, span := range []Span{
		Between(4*time.Second, 4*time.Second),
		Between(5*time.Second, 2*time.Second)} {
		for _, at := range []time.Duration{0, 2 * time.Second,
			4 * time.Second, 5 * time.Second} {
			if span.Holds(at) {
				t.Errorf("%v holds %v", span, at)
			}
		}
	}
}

// painted is the picture with a hud of one element drawn at a time.
func paintedAt(t *testing.T, at time.Duration, element Element) *image.RGBA {
	t.Helper()
	img := grey(200, 100)
	err := Hud{Elements: []Element{element}}.Picture(img, 10, at, dusk,
		View{})
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// each kind of element is drawn inside its span and leaves the picture
// as it was outside it.
func TestEachKindIsDrawnOnlyInsideItsSpan(t *testing.T) {
	within := Between(time.Second, 2*time.Second)
	kinds := map[string]Element{
		"text": Text{Words: Words("hello"), Ink: "ink", Within: within},
		"frame": Frame{Box: Box{0, 0, 1, 1}, Ink: "ink",
			Within: within},
		"bar": Bar{Box: Box{0, 0, 0.5, 0.2}, Value: Fixed(1), Fill: "ok",
			Empty: "ground", Within: within},
	}
	plain := grey(200, 100)
	for name, element := range kinds {
		inside := paintedAt(t, time.Second, element)
		if same(inside, plain) {
			t.Errorf("%s: nothing drawn inside its span", name)
		}
		for _, at := range []time.Duration{0, 2 * time.Second} {
			if !same(paintedAt(t, at, element), plain) {
				t.Errorf("%s: drawn at %v, outside its span", name, at)
			}
		}
	}
}

// an element outside its span asks the theme for no role: a role the
// theme has not is not found while the element is away.
func TestAnElementAwayAsksForNoRole(t *testing.T) {
	away := Text{Words: Words("x"), Ink: "no-such-role",
		Within: From(time.Hour)}
	img := grey(200, 100)
	if err := (Hud{Elements: []Element{away}}).Picture(img, 10, 0, dusk,
		View{}); err != nil {
		t.Errorf("an element away was laid out: %v", err)
	}
	here := Text{Words: Words("x"), Ink: "no-such-role"}
	if err := (Hud{Elements: []Element{here}}).Picture(img, 10, 0, dusk,
		View{}); err == nil {
		t.Error("an element in its span with no role was drawn")
	}
}

// a bar's gauge out of range is still refused inside the span, and not
// looked at outside it.
func TestABarsValueIsCheckedOnlyInsideItsSpan(t *testing.T) {
	bar := Bar{Box: Box{0, 0, 0.5, 0.2}, Value: Fixed(math.NaN()),
		Fill: "ok", Empty: "ground", Within: From(time.Second)}
	img := grey(200, 100)
	if err := (Hud{Elements: []Element{bar}}).Picture(img, 10, 0, dusk,
		View{}); err != nil {
		t.Errorf("a bar away was checked: %v", err)
	}
	if err := (Hud{Elements: []Element{bar}}).Picture(img, 10,
		time.Second, dusk, View{}); err == nil {
		t.Error("a bar with no value was drawn inside its span")
	}
}

// same is whether two pictures hold the same bytes.
func same(a, b *image.RGBA) bool {
	return string(a.Pix) == string(b.Pix)
}
