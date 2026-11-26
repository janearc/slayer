package squid

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

type profile struct {
	Kind, Status, By, Date, Source string
	Mantle                         float64 `yaml:"mantle-metres"`
	Coat, Mottle                   maths.Light
	Anatomy                        Anatomy
}

// Book is an immutable, checked book of illustrative squid profiles.
// Profiles select size, anatomy and appearance on the shared posed scaffold.
type Book struct{ profiles map[string]Recipe }

// Read checks and owns the data in one bounded species book.
func Read(file string, raw []byte) (*Book, error) {
	rows, err := profilecheck.Rows[profile](file, raw, "squid profile book", "squid profile", nil)
	if err != nil {
		return nil, err
	}
	book := &Book{profiles: make(map[string]Recipe, len(rows))}
	for _, profile := range rows {
		if !profilecheck.Authored(strings.TrimSpace(profile.Kind), profile.Status,
			strings.TrimSpace(profile.By), strings.TrimSpace(profile.Date), strings.TrimSpace(profile.Source)) {
			return nil, fmt.Errorf("%s: invalid squid provenance for %q", file, profile.Kind)
		}
		if _, found := book.profiles[profile.Kind]; found {
			return nil, fmt.Errorf("%s: duplicate squid %q", file, profile.Kind)
		}
		if !validSquidLength(profile.Mantle) {
			return nil, fmt.Errorf("%s: invalid mantle length for %q", file, profile.Kind)
		}
		a := profile.Anatomy
		if !squidBetween(a.MantleRadius, .12, .36) || !squidBetween(a.HeadWidth, .15, .3) ||
			!squidBetween(a.FinSpan, .18, .34) || !squidBetween(a.ArmLength, .85, 1.15) ||
			!squidBetween(a.TentacleLength, 1.25, 1.65) {
			return nil, fmt.Errorf("%s: invalid squid anatomy for %q", file, profile.Kind)
		}
		for _, pair := range [][2]float64{{profile.Coat.R, profile.Mottle.R}, {profile.Coat.G, profile.Mottle.G}, {profile.Coat.B, profile.Mottle.B}} {
			base, variation := pair[0], pair[1]
			if math.IsNaN(base) || math.IsInf(base, 0) || math.IsNaN(variation) || math.IsInf(variation, 0) ||
				variation < 0 || base < variation || base+variation > 1 {
				return nil, fmt.Errorf("%s: invalid squid coat for %q", file, profile.Kind)
			}
		}
		book.profiles[profile.Kind] = Recipe{spec: Spec{Mantle: profile.Mantle, Coat: profile.Coat, Mottle: profile.Mottle, Anatomy: a, Rest: sync.OnceValues(func() (*shape.Mesh, error) { return BuildRest(a) })}}
	}
	return book, nil
}

func squidBetween(value, least, most float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= least && value <= most
}

// validSquidLength is a numerical construction bound in metres, not a
// biological range. It keeps scaled details and normal calculations stable.
func validSquidLength(length float64) bool {
	return !math.IsNaN(length) && !math.IsInf(length, 0) && length >= 0.01 && length <= 100
}

// Recipe keeps a checked profile and its cached rest surface private.
type Recipe struct{ spec Spec }

// Build constructs the specimen at the requested illustrative phase.
func (recipe Recipe) Build(phase float64) ([]thing.Thing, error) {
	return NewSquid(recipe.spec, phase)
}

// Get returns the checked recipe for a species.
func (book *Book) Get(species string) (Recipe, bool) {
	if book == nil {
		return Recipe{}, false
	}
	profile, ok := book.profiles[species]
	return profile, ok
}
