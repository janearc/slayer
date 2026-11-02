// Package shortcut describes rendering approximations with explicit error
// budgets: baked noise, empty-space skipping, distance-dependent detail,
// density reuse, and cutoffs. With all shortcuts disabled, smoke renders must
// remain byte-for-byte unchanged.
//
// Settings can be global or per object and can be read from a sheet. Record
// describes enabled settings; Error compares with a reference render. Neither
// is connected to frame records or routine measurement yet.
//
// Apply currently implements baked noise. Other declared settings remain
// inactive until the tracer provides their hooks; Set documents those limits.
package shortcut

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Set is which shortcuts are on.
type Set struct {
	// Baked computes a medium's density once into a grid of Resolution
	// cells a side over its bounds and reads it back, instead of evaluating
	// the noise at every sample. Resolution zero is 96.
	Baked      bool
	Resolution int
	// Skip steps a ray over the empty space inside a medium's bounds.
	// Declared; no effect until render/trace takes it.
	Skip bool
	// Distance takes fewer noise octaves where a sample is far from
	// the eye. Declared; no effect until render/trace takes it.
	Distance bool
	// Once evaluates density once a sample and shares it across bands.
	// Declared; no effect until render/trace takes it.
	Once bool
	// Cutoff stops a ray the moment what is left of its light cannot
	// change a pixel after the tone curve, Threshold saying where; the
	// error is bounded by the remainder. Declared; no effect until
	// render/trace takes it in place of its own fixed faintness.
	Cutoff bool
}

// All is every shortcut on, at the default resolution.
func All() Set {
	return Set{Baked: true, Skip: true, Distance: true, Once: true, Cutoff: true}
}

// None is every shortcut off: the exact render.
func None() Set { return Set{} }

// Any is whether a shortcut is on.
func (s Set) Any() bool {
	return s.Baked || s.Skip || s.Distance || s.Once || s.Cutoff
}

// String is the set as a sheet writes it: "none", "all", or the words.
func (s Set) String() string {
	if !s.Any() {
		return "none"
	}
	if s == All() {
		return "all"
	}
	var words []string
	if s.Baked {
		if s.Resolution > 0 {
			words = append(words, fmt.Sprintf("baked(%d)", s.Resolution))
		} else {
			words = append(words, "baked")
		}
	}
	if s.Skip {
		words = append(words, "skip")
	}
	if s.Distance {
		words = append(words, "distance")
	}
	if s.Once {
		words = append(words, "once")
	}
	if s.Cutoff {
		words = append(words, "cutoff")
	}
	return strings.Join(words, " ")
}

// Parse reads a set as a sheet writes it: none, all, or words from
// baked, baked(N), skip, distance, once and cutoff, in any order; of two
// baked(N), the last. N is from two to largest, past which a bake is
// more memory than a picture is worth, and a sign is allowed.
func Parse(text string) (Set, error) {
	text = strings.TrimSpace(text)
	switch text {
	case "none", "":
		return None(), nil
	case "all":
		return All(), nil
	}
	var s Set
	for _, w := range strings.Fields(text) {
		switch {
		case w == "baked":
			s.Baked = true
		case strings.HasPrefix(w, "baked(") && strings.HasSuffix(w, ")"):
			n, err := strconv.Atoi(w[len("baked(") : len(w)-1])
			if err != nil || n < 2 || n > largest {
				return Set{}, fmt.Errorf("shortcut: baked wants a resolution "+
					"from two to %d, not %q", largest, w)
			}
			s.Baked, s.Resolution = true, n
		case w == "skip":
			s.Skip = true
		case w == "distance":
			s.Distance = true
		case w == "once":
			s.Once = true
		case w == "cutoff":
			s.Cutoff = true
		default:
			return Set{}, fmt.Errorf("shortcut: %q is none of none, all, baked, baked(N), skip, distance, once, cutoff", w)
		}
	}
	return s, nil
}

// Plan is the shortcuts for a picture: the global set, and a set per
// thing by name that stands in for it.
type Plan struct {
	Global Set
	Things map[string]Set
}

// For is the set a thing takes: its own, or the global.
func (p Plan) For(name string) Set {
	if s, ok := p.Things[name]; ok {
		return s
	}
	return p.Global
}

// ForPart is the set a part of a thing takes: the thing's own when the
// part's name begins with the thing's, as a body's parts are named
// "jupiter gas" and "jupiter air", else the global.
func (p Plan) ForPart(name string) Set {
	best, found := "", false
	for thing := range p.Things {
		if (name == thing || strings.HasPrefix(name, thing+" ")) && len(thing) > len(best) {
			best, found = thing, true
		}
	}
	if found {
		return p.Things[best]
	}
	return p.Global
}

// With is the plan with a thing's own set.
func (p Plan) With(name string, s Set) Plan {
	things := map[string]Set{}
	for k, v := range p.Things {
		things[k] = v
	}
	things[name] = s
	return Plan{Global: p.Global, Things: things}
}

// Record is the plan as a frame's record names it: the global set,
// then each thing's own, in name order.
func (p Plan) Record() []string {
	out := []string{"global: " + p.Global.String()}
	names := make([]string, 0, len(p.Things))
	for n := range p.Things {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		out = append(out, n+": "+p.Things[n].String())
	}
	return out
}

// Budgets are how far each shortcut may move a picture from the exact
// render, as the worst oklab distance over the compared pixels: an eye
// each, which is libtheme's two hundredths, so a shortcut is honest
// when no one can see it.
var Budgets = map[string]float64{
	"baked":    0.02,
	"skip":     0.02,
	"distance": 0.02,
	"once":     0.02,
	"cutoff":   0.02,
}
