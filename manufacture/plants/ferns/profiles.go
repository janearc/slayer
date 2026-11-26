// Package ferns describes the authored species used by the shared plant maker.
package ferns

import (
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
)

// FloatRange bounds a seeded physical measurement in metres.
type FloatRange struct{ Minimum, Maximum float64 }

// CountRange bounds a seeded whole-number count.
type CountRange struct{ Minimum, Maximum int }

// Anatomy holds the fern's authored crown, stalk and leaflet proportions.
type Anatomy struct {
	CrownSpan         float64 `yaml:"crown-span"`
	CrownRadius       float64 `yaml:"crown-radius"`
	StalkTip          float64 `yaml:"stalk-tip"`
	LeafletTip        float64 `yaml:"leaflet-tip"`
	LeafletReach      float64 `yaml:"leaflet-reach"`
	LeafletShortening float64 `yaml:"leaflet-shortening"`
	LeafletGirth      float64 `yaml:"leaflet-girth"`
	LeafletMinimum    float64 `yaml:"leaflet-minimum"`
}

func (a Anatomy) valid() bool {
	between := func(n, low, high float64) bool { return n >= low && n <= high }
	return between(a.CrownSpan, .001, 1) && between(a.CrownRadius, .001, 16) &&
		between(a.StalkTip, 0, 1) && between(a.LeafletTip, 0, 1) &&
		between(a.LeafletReach, .001, 1) && between(a.LeafletShortening, 0, .999) &&
		between(a.LeafletGirth, .001, 16) && between(a.LeafletMinimum, .001, 1)
}

// Profile gives one fern's dimensions, count ranges, colour and provenance.
type Profile struct {
	Kind, Status, By, Date, Source string
	FrondLength                    FloatRange `yaml:"frond-length"`
	StalkRadius                    FloatRange `yaml:"stalk-radius"`
	Fronds, Leaflets               CountRange
	Colour                         maths.Light
	Anatomy                        Anatomy
}

// Book owns validated fern profiles without exposing mutable shared state.
type Book struct{ profiles map[string]Profile }

// Read decodes and checks one authored fern profile book.
func Read(file string, raw []byte) (*Book, error) {
	rows, err := profilecheck.PlantRows[Profile](file, raw, nil)
	if err != nil {
		return nil, err
	}
	return newBook(file, rows)
}

// newBook validates and copies the rows from a checked authored book.
func newBook(file string, rows []Profile) (*Book, error) {
	profiles := make(map[string]Profile, len(rows))
	for _, profile := range rows {
		if err := profile.validate(); err != nil {
			return nil, fmt.Errorf("manufacture: %s: %w", file, err)
		}
		if _, exists := profiles[profile.Kind]; exists {
			return nil, fmt.Errorf("manufacture: %s: duplicate fern %q", file, profile.Kind)
		}
		profiles[profile.Kind] = profile
	}
	return &Book{profiles: profiles}, nil
}

func (profile Profile) validate() error {
	if !profilecheck.Authored(profile.Kind, profile.Status, profile.By, profile.Date, profile.Source) {
		return fmt.Errorf("invalid fern provenance for %q", profile.Kind)
	}
	if !profile.Anatomy.valid() {
		return fmt.Errorf("invalid fern anatomy for %q", profile.Kind)
	}
	for _, r := range []FloatRange{profile.FrondLength, profile.StalkRadius} {
		if !(r.Minimum > 0) || math.IsInf(r.Minimum, 0) || !(r.Maximum > 0) ||
			math.IsInf(r.Maximum, 0) || r.Maximum < r.Minimum || r.Maximum > 1000 {
			return fmt.Errorf("invalid fern dimensions for %q", profile.Kind)
		}
	}
	// The stalk and frond are meshed together; a nearly vanishing one
	// collapses after coordinate rounding despite being positive.
	largest := math.Max(profile.FrondLength.Maximum, profile.StalkRadius.Maximum)
	minimum := math.Max(1e-9, largest*1e-9)
	if profile.FrondLength.Minimum < minimum || profile.StalkRadius.Minimum < minimum {
		return fmt.Errorf("fern dimensions too small for stable geometry for %q", profile.Kind)
	}
	for _, r := range []CountRange{profile.Fronds, profile.Leaflets} {
		if r.Minimum < 1 || r.Maximum < r.Minimum || r.Maximum > 256 {
			return fmt.Errorf("invalid fern counts for %q", profile.Kind)
		}
	}
	if profile.Fronds.Maximum*profile.Leaflets.Maximum > 4096 {
		return fmt.Errorf("excessive fern foliage for %q", profile.Kind)
	}
	for _, value := range []float64{profile.Colour.R, profile.Colour.G, profile.Colour.B} {
		if math.IsNaN(value) || value < 0 || value > 1 {
			return fmt.Errorf("invalid fern colour for %q", profile.Kind)
		}
	}
	return nil
}

// Get returns a copy of one validated profile.
func (book *Book) Get(name string) (Profile, bool) {
	if book == nil {
		return Profile{}, false
	}
	profile, ok := book.profiles[name]
	return profile, ok
}

// Species lists the profile names in a stable order.
func (book *Book) Species() []string {
	if book == nil {
		return nil
	}
	return slices.Sorted(maps.Keys(book.profiles))
}

// habit arranges paired leaflets along a fern's fronds.
func (anatomy Anatomy) habit() branching.Habit {
	return branching.Habit{
		Turn: measure.Radians(math.Pi), Start: 0.12, Top: 0.98,
		Thinning: 0, Fewest: 1, Stray: 0.02,
		Low: measure.Degrees(70), Lift: measure.Degrees(15), Sway: measure.Degrees(4),
		Reach: anatomy.LeafletReach, Shortening: anatomy.LeafletShortening, Uneven: 0.06,
		Girth: anatomy.LeafletGirth, Least: anatomy.LeafletMinimum,
	}
}
