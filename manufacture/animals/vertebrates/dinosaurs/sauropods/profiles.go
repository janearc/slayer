// Package sauropods holds checked, illustrative sauropod profiles.
package sauropods

import (
	_ "embed"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
)

// Station is a cross-section in metres, from tail to head. X and Y are
// its centre, Width is the lateral half-radius, Height the vertical half-radius.
type Station struct{ X, Y, Width, Height float64 }

// Limb describes a paired forelimb or hindlimb. Station names its body
// attachment; the feet are centered at (FootX, 0, ±FootZ), with the given
// full length and width in metres.
type Limb struct {
	Station                             int
	FootX, FootZ, FootLength, FootWidth float64
}

// Head positions the eyes at an authored station. EyeHeight is a share of
// the cross-section height and EyeRadius is in metres.
type Head struct {
	EyeStation           int
	EyeHeight, EyeRadius float64
}

// Profile is one static reconstruction; the selected soft-tissue shape,
// pose and colours are not measurements of a living animal.
type Profile struct {
	Kind, Scientific, Basis, Source, Notes string
	Stations                               []Station
	Fore, Hind                             Limb
	Head                                   Head
	Skin, Underside                        maths.Light
}

//go:embed profiles.yaml
var embedded []byte

// Embedded returns a copy of the embedded authoring book.
func Embedded() (string, []byte) { return "sauropod-profiles.yaml", append([]byte(nil), embedded...) }

// Book owns its checked profiles independently of the source and callers.
type Book struct{ profiles map[string]Profile }

// Read checks and snapshots a book before returning it.
func Read(file string, raw []byte) (*Book, error) {
	rows, err := profilecheck.Rows[Profile](file, raw, "sauropod profile book", "sauropod profile", nil)
	if err != nil {
		return nil, err
	}
	book := &Book{profiles: make(map[string]Profile, len(rows))}
	for _, p := range rows {
		if err := p.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if _, exists := book.profiles[p.Kind]; exists {
			return nil, fmt.Errorf("%s: duplicate sauropod profile %q", file, p.Kind)
		}
		book.profiles[p.Kind] = p
	}
	return book, nil
}

// Species returns stable profile keys, not global Kind aliases.
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

// Profile returns an independently owned reconstruction.
func (book *Book) Profile(name string) (Profile, error) {
	if book == nil {
		return Profile{}, fmt.Errorf("sauropod: no profile book")
	}
	p, ok := book.profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("sauropod: unknown species %q", name)
	}
	p.Stations = append([]Station(nil), p.Stations...)
	return p, nil
}

func within(value, low, high float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= low && value <= high
}

// Validate checks the builder's resource and geometry limits; they are not
// claims about the largest, smallest, or complete anatomy of a taxon.
func (p Profile) Validate() error {
	bad := func(field string) error { return fmt.Errorf("sauropod %q: invalid %s", p.Kind, field) }
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
	if len(p.Stations) < 8 || len(p.Stations) > 64 {
		return bad("stations count")
	}
	for i, s := range p.Stations {
		if !within(s.X, -100, 100) || (i > 0 && s.X-p.Stations[i-1].X < .00001) {
			return bad(fmt.Sprintf("stations[%d].x", i))
		}
		if !within(s.Y, .02, 50) {
			return bad(fmt.Sprintf("stations[%d].y", i))
		}
		if !within(s.Width, .005, 10) {
			return bad(fmt.Sprintf("stations[%d].width", i))
		}
		if !within(s.Height, .005, 10) || s.Height >= s.Y {
			return bad(fmt.Sprintf("stations[%d].height", i))
		}
	}
	// The actual sampled axial outline is checked before a book is accepted.
	// This refuses folded contours; it is not a complete 3D intersection test.
	ss := sections(p.Stations)
	for _, s := range ss {
		if s.center.Y <= s.height {
			return bad("body section reaches the ground")
		}
	}
	if crossingSections(ss) {
		return bad("crossing axial silhouette")
	}
	if p.Stations[len(p.Stations)-1].X-p.Stations[0].X > 100 {
		return bad("length")
	}
	if p.Hind.Station < 1 || p.Fore.Station >= len(p.Stations)-1 || p.Fore.Station-p.Hind.Station < 2 {
		return bad("limb stations")
	}
	for name, limb := range map[string]Limb{"fore": p.Fore, "hind": p.Hind} {
		station := p.Stations[limb.Station]
		before, after := p.Stations[limb.Station-1], p.Stations[limb.Station+1]
		if math.Abs((station.Y-before.Y)/(station.X-before.X)) > .75 || math.Abs((after.Y-station.Y)/(after.X-station.X)) > .75 {
			return bad(name + " attachment slope")
		}
		if !within(limb.FootX, station.X-station.Width, station.X+station.Width) {
			return bad(name + ".footx")
		}
		if !within(limb.FootZ, .3*station.Width, 1.5*station.Width) {
			return bad(name + ".footz")
		}
		if !within(limb.FootLength, .005, 2*station.Width) {
			return bad(name + ".footlength")
		}
		if !within(limb.FootWidth, .005, 2*station.Width) || limb.FootWidth >= 2*limb.FootZ {
			return bad(name + ".footwidth")
		}
	}
	if p.Fore.FootX-p.Hind.FootX <= (p.Fore.FootLength+p.Hind.FootLength)/2 {
		return bad("overlapping feet")
	}
	if p.Head.EyeStation < p.Fore.Station+2 || p.Head.EyeStation >= len(p.Stations)-1 {
		return bad("head.eyestation")
	}
	if !within(p.Head.EyeHeight, -.8, .8) {
		return bad("head.eyeheight")
	}
	head := p.Stations[p.Head.EyeStation]
	if !within(p.Head.EyeRadius, .001, .3*math.Min(head.Width, head.Height)) {
		return bad("head.eyeradius")
	}
	for name, colour := range map[string]maths.Light{"skin": p.Skin, "underside": p.Underside} {
		if !within(colour.R, 0, 1) || !within(colour.G, 0, 1) || !within(colour.B, 0, 1) {
			return bad(name)
		}
	}
	return nil
}
