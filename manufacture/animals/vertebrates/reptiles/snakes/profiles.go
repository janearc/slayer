// Package snakes holds checked snake profiles and their construction.
package snakes

import (
	_ "embed"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/janearc/slayer/bioforms/pigment"
	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
	"gopkg.in/yaml.v3"
)

// Station is a cross-section, tail to snout. Width and Height are
// half-radii as shares of the specimen's total length. The final positive
// radii control the rounding within the last interval; the snout itself
// closes at a zero-radius pole at At=1. All earlier stations are on the skin.
type Station struct {
	At, Width, Height float64
}

// Head locates the eyes; Pit requests illustrative facial pit markings.
type Head struct {
	EyeAt, EyeHeight, EyeSize float64
	Pit                       bool
}

// Pattern uses linear RGB reflectances. Repeat is the approximate number of
// markings along the full body; Contrast is the strength of the markings.
// Mottle varies the dorsal and flank palette in local metres. Warp bends
// the dorsal motifs through a separate, seeded field in the same chart.
type Pattern struct {
	Kind                    string
	Base, Mark, Belly, Edge maths.Light
	Repeat, Contrast        float64
	Mottle, Warp            pigment.Mottle
}

// Profile describes one illustrative specimen, not the limits of a species.
type Profile struct {
	Kind, Scientific, Basis, Source, Notes string
	Length                                 float64
	Stations                               []Station
	Head                                   Head
	Pattern                                Pattern
}

//go:embed profiles.yaml
var embedded []byte

// Embedded returns a separately owned copy of the default book.
func Embedded() (string, []byte) { return "snake-profiles.yaml", append([]byte(nil), embedded...) }

// Book owns checked, independent specimen profiles.
type Book struct{ profiles map[string]Profile }

// Read checks a book and retains no references to the input bytes.
func Read(file string, raw []byte) (*Book, error) {
	rows, err := profilecheck.Rows[Profile](file, raw, "snake profile book", "snake profile",
		func(_ *yaml.Node, path string) bool {
			return path == "pattern.mottle" || path == "pattern.warp"
		})
	if err != nil {
		return nil, err
	}
	// A missing boolean otherwise decodes as false; a checked profile must
	// explicitly state whether this reconstruction includes pit marks.
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	sequence := profilecheck.Alias(profilecheck.Field(document.Content[0], "profiles"))
	for i, node := range sequence.Content {
		pit := profilecheck.Field(profilecheck.Field(node, "head"), "pit")
		if pit == nil || profilecheck.Alias(pit).Tag != "!!bool" {
			return nil, fmt.Errorf("%s: profile %d: missing or invalid head.pit", file, i+1)
		}
	}
	book := &Book{profiles: make(map[string]Profile, len(rows))}
	for _, row := range rows {
		if err := row.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if _, exists := book.profiles[row.Kind]; exists {
			return nil, fmt.Errorf("%s: duplicate snake profile %q", file, row.Kind)
		}
		book.profiles[row.Kind] = row
	}
	return book, nil
}

// Species lists the profile keys in stable order.
func (book *Book) Species() []string {
	if book == nil {
		return nil
	}
	names := make([]string, 0, len(book.profiles))
	for name := range book.profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Profile returns an independently owned specimen, or refuses an unknown key.
func (book *Book) Profile(name string) (Profile, error) {
	if book == nil {
		return Profile{}, fmt.Errorf("snake: no profile book")
	}
	p, ok := book.profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("snake: unknown species %q", name)
	}
	p.Stations = append([]Station(nil), p.Stations...)
	return p, nil
}

func within(value, low, high float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= low && value <= high
}

// Subdivided loft rings need distinct positions even at the largest scale.
const stationGap = 1e-6

// Validate protects construction from malformed books and caller-mutated values.
// These numerical limits bound this implementation; they are not claims about
// species biology. The authored morphology and colours are reconstructions.
func (p Profile) Validate() error {
	bad := func(field string) error { return fmt.Errorf("snake %q: invalid %s", p.Kind, field) }
	if strings.TrimSpace(p.Kind) == "" {
		return bad("kind")
	}
	if strings.TrimSpace(p.Scientific) == "" {
		return bad("scientific")
	}
	if p.Basis != "picked" {
		return bad("basis")
	}
	if strings.TrimSpace(p.Source) == "" {
		return bad("source")
	}
	if strings.TrimSpace(p.Notes) == "" {
		return bad("notes")
	}
	if !within(p.Length, .001, 50) {
		return bad("length")
	}
	if len(p.Stations) < 4 || len(p.Stations) > 64 {
		return bad("stations count")
	}
	for i, station := range p.Stations {
		if !within(station.At, 0, 1) || (i > 0 && station.At-p.Stations[i-1].At < stationGap) {
			return bad(fmt.Sprintf("stations[%d].at", i))
		}
		if !within(station.Width, .00001, .25) {
			return bad(fmt.Sprintf("stations[%d].width", i))
		}
		if !within(station.Height, .00001, .25) {
			return bad(fmt.Sprintf("stations[%d].height", i))
		}
	}
	if p.Stations[0].At != 0 || p.Stations[len(p.Stations)-1].At != 1 {
		return bad("station endpoints")
	}
	if !within(p.Head.EyeAt, .8, 1) {
		return bad("head.eyeat")
	}
	if !within(p.Head.EyeHeight, -.8, .8) {
		return bad("head.eyeheight")
	}
	if !within(p.Head.EyeSize, .00001, .03) {
		return bad("head.eyesize")
	}
	// Use the smaller adjoining radii, so this stays conservative when a
	// loft rounds between stations. The eye may move inward by up to .75 of
	// its radius when seated on a sloping surface, and must stay behind the
	// terminal nose taper and small relative to the local cross-section.
	for i := 1; i < len(p.Stations); i++ {
		a, b := p.Stations[i-1], p.Stations[i]
		if p.Head.EyeAt <= b.At {
			localRadius := math.Min(math.Min(a.Width, b.Width), math.Min(a.Height, b.Height))
			if p.Head.EyeSize > .3*localRadius || 1.75*p.Head.EyeSize > p.Stations[len(p.Stations)-2].At-p.Head.EyeAt {
				return bad("head.eyesize")
			}
			break
		}
	}
	switch p.Pattern.Kind {
	case "plain", "saddles", "spots", "reticulated", "triangles", "bands":
	default:
		return bad("pattern.kind")
	}
	for name, color := range map[string]maths.Light{"base": p.Pattern.Base, "mark": p.Pattern.Mark, "belly": p.Pattern.Belly, "edge": p.Pattern.Edge} {
		if !within(color.R, 0, 1) || !within(color.G, 0, 1) || !within(color.B, 0, 1) {
			return bad("pattern." + name)
		}
	}
	if !within(p.Pattern.Repeat, .001, 256) {
		return bad("pattern.repeat")
	}
	if !within(p.Pattern.Contrast, 0, 1) {
		return bad("pattern.contrast")
	}
	if !p.Pattern.Mottle.Valid() {
		return bad("pattern.mottle")
	}
	if !p.Pattern.Warp.Valid() {
		return bad("pattern.warp")
	}
	return nil
}
