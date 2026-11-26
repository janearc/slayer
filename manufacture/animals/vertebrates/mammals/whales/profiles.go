package whales

import (
	_ "embed"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed profiles.yaml
var profileRaw []byte

// Palette supplies the picked or sourced linear reflectances of the orca
// study. It changes appearance without changing the fixed orca anatomy.
type Palette struct {
	Basis, Source                                       string
	Black, White, Saddle, Pale, Iris, Pupil, Gum, Ivory maths.Light
}

type Profile struct {
	Kind, Form, Status, By, Date, Source string
	Cycle                                float64 `yaml:"cycle-seconds"`
	Satin                                float64 `yaml:"satin"`
	Palette                              Palette
	Stations                             [][4]float64 `yaml:"stations"`
	model                                *orcaModel
}

// Book owns checked whale profiles. The current form uses the
// authored orca anatomy; it is not a general scaffold for all cetaceans.
type Book struct{ profiles map[string]Profile }

// Read snapshots and checks the species book.
func Read(file string, raw []byte) (*Book, error) {
	rows, err := profilecheck.Rows[Profile](file, raw, "whale book", "whale profile", nil)
	if err != nil {
		return nil, err
	}
	book := &Book{profiles: make(map[string]Profile, len(rows))}
	for _, p := range rows {
		if !profilecheck.Authored(strings.TrimSpace(p.Kind), p.Status,
			strings.TrimSpace(p.By), strings.TrimSpace(p.Date), strings.TrimSpace(p.Source)) {
			return nil, fmt.Errorf("%s: invalid whale provenance for %q", file, p.Kind)
		}
		if _, found := book.profiles[p.Kind]; found {
			return nil, fmt.Errorf("%s: duplicate whale %q", file, p.Kind)
		}
		if p.Form != "orca" || math.IsNaN(p.Cycle) || math.IsInf(p.Cycle, 0) || p.Cycle < .01 || p.Cycle > 10000 || math.IsNaN(p.Satin) || math.IsInf(p.Satin, 0) || p.Satin < 0 || p.Satin > 1 {
			return nil, fmt.Errorf("%s: invalid whale profile %q", file, p.Kind)
		}
		if !profilecheck.Authored(p.Kind, p.Palette.Basis, p.By, p.Date, strings.TrimSpace(p.Palette.Source)) {
			return nil, fmt.Errorf("%s: invalid whale palette provenance for %q", file, p.Kind)
		}
		for _, colour := range []struct {
			name  string
			value maths.Light
		}{
			{"black", p.Palette.Black}, {"white", p.Palette.White}, {"saddle", p.Palette.Saddle}, {"pale", p.Palette.Pale},
			{"iris", p.Palette.Iris}, {"pupil", p.Palette.Pupil}, {"gum", p.Palette.Gum}, {"ivory", p.Palette.Ivory},
		} {
			for _, channel := range []float64{colour.value.R, colour.value.G, colour.value.B} {
				if math.IsNaN(channel) || math.IsInf(channel, 0) || channel < 0 || channel > 1 {
					return nil, fmt.Errorf("%s: invalid whale palette %s for %q", file, colour.name, p.Kind)
				}
			}
		}
		model, err := newOrcaModel(p.Stations)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", file, p.Kind, err)
		}
		if canonicalOrca != nil && slices.Equal(model.stations, canonicalOrca.stations) {
			model = canonicalOrca
		}
		p.model = model
		book.profiles[p.Kind] = p
	}
	return book, nil
}

// Embedded returns an independent copy of the authored book.
func Embedded() []byte { return append([]byte(nil), profileRaw...) }

// Get returns a checked profile and its immutable model.
func (book *Book) Get(name string) (Profile, bool) {
	if book == nil {
		return Profile{}, false
	}
	p, found := book.profiles[name]
	p.Stations = slices.Clone(p.Stations)
	return p, found
}

// Swim constructs the selected profile's swimming model.
func (p Profile) Swim(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	return p.model.swimming(phase, at, p.Palette, p.Satin)
}

// SpyHop constructs the selected profile in its spy-hop pose.
func (p Profile) SpyHop(at maths.Subspace) ([]thing.Thing, error) {
	return p.model.spyHop(at, p.Palette, p.Satin)
}

// Bones constructs the selected profile's skeleton.
func (p Profile) Bones(at maths.Subspace) ([]thing.Thing, error) { return p.model.orcaBones(at) }

var canonicalOrca *orcaModel

func defaultOrcaModel() *orcaModel { return canonicalOrca }

var defaultBook = func() *Book {
	book, err := Read("profiles.yaml", profileRaw)
	if err != nil {
		panic(err)
	}
	canonicalOrca = book.profiles["orca"].model
	return book
}()

// Default is the checked embedded book.
func Default() *Book { return defaultBook }
