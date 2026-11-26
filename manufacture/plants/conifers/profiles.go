package conifers

import (
	_ "embed"
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/research"
)

type Profile struct {
	Kind, Status, By, Date, Source       string
	Height, Slenderness, Between, Lowest [2]float64
	Habit                                HabitProfile
	Trunk, Bark, Clad                    SleeveProfile
	Branch                               LimbProfile
	Shoots                               *ShootsProfile
	Foliage                              string
	Needle                               NeedleProfile
	Flare                                float64
	BarkName                             string     `yaml:"bark-name"`
	BarkColour                           [3]float64 `yaml:"bark-colour"`
	LeafName                             string     `yaml:"leaf-name"`
	LeafColour                           [3]float64 `yaml:"leaf-colour"`
}

// NeedleProfile holds species-specific foliage dimensions in metres.
type NeedleProfile struct {
	Length                           [2]float64
	LengthStatus                     string    `yaml:"length-status"`
	LengthSource                     string    `yaml:"length-source"`
	LengthReferences                 [2]string `yaml:"length-references"`
	references                       []numbers.Number
	Width, Thickness, Spacing, Angle float64
	Rows                             int
	WoodSides                        int `yaml:"wood-sides"`
}

type HabitProfile struct {
	Turn, Start, Top, Thinning, Stray, Low, Lift, Sway float64
	Fewest                                             int
	Reach, Shortening, Uneven, Girth, Least, Own       float64
}

type SleeveProfile struct {
	Joints [][2]float64
	Sides  int
}

type LimbProfile struct {
	Pieces        int
	Droop, Wander float64
}
type ShootsProfile struct {
	Per   [2]float64
	Shoot LimbProfile
	Habit HabitProfile
	Wear  SleeveProfile
}

//go:embed conifer-profiles.yaml
var profilesRaw []byte

// Tree is the checked whorled architecture and its numerical ranges.
type Tree struct {
	Tall, Slender, Between, Lowest [2]float64
	Hangs                          branching.Habit
	Grows                          Architecture
	FoliageStyle                   int
	Flare                          float64
}

type rangeValue struct{ least, most float64 }

func positive(n float64) bool { return n > 0 && !math.IsInf(n, 0) }
func finite(n float64) bool   { return !math.IsNaN(n) && !math.IsInf(n, 0) }

// Book owns checked conifer profiles and the caller's shared number policy.
type Book struct {
	profiles map[string]Profile
	value    func(string) float64
	maximum  float64
}

// Read checks an authored family book and retains the original choice callback.
func Read(file string, raw []byte, value func(string) float64, maximum float64) (*Book, error) {
	if value == nil {
		return nil, fmt.Errorf("%s: no conifer choice policy", file)
	}
	rows, err := profilecheck.PlantRows[Profile](file, raw, nil)
	if err != nil {
		return nil, err
	}
	profiles := make(map[string]Profile, len(rows))
	for _, p := range rows {
		if !profilecheck.Authored(p.Kind, p.Status, p.By, p.Date, p.Source) {
			return nil, fmt.Errorf("%s: invalid conifer provenance for %q", file, p.Kind)
		}
		if _, exists := profiles[p.Kind]; exists {
			return nil, fmt.Errorf("%s: duplicate conifer %q", file, p.Kind)
		}
		if _, err := p.tree(value, maximum); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", file, p.Kind, err)
		}
		refs, err := p.Needle.verifiedReferences()
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", file, p.Kind, err)
		}
		p.Needle.references = refs
		profiles[p.Kind] = p
	}
	return &Book{profiles: profiles, value: value, maximum: maximum}, nil
}

// Embedded returns an independent copy of the default authored book.
func Embedded() []byte { return slices.Clone(profilesRaw) }

// Has reports whether this checked book contains a species.
func (book *Book) Has(name string) bool {
	if book == nil {
		return false
	}
	_, ok := book.profiles[name]
	return ok
}

// Species lists checked species in stable order.
func (book *Book) Species() []string {
	if book == nil {
		return nil
	}
	return slices.Sorted(maps.Keys(book.profiles))
}

// Select returns an independent profile and its freshly checked tree.
func (book *Book) Select(name string) (Profile, Tree, error) {
	if book == nil {
		return Profile{}, Tree{}, fmt.Errorf("manufacture: no conifer profile book")
	}
	p, found := book.profiles[name]
	if !found {
		return Profile{}, Tree{}, fmt.Errorf("manufacture: no conifer profile %q", name)
	}
	tree, err := p.tree(book.value, book.maximum)
	if err != nil {
		return Profile{}, Tree{}, err
	}
	p.Trunk.Joints = slices.Clone(p.Trunk.Joints)
	p.Bark.Joints = slices.Clone(p.Bark.Joints)
	p.Clad.Joints = slices.Clone(p.Clad.Joints)
	if p.Shoots != nil {
		shoots := *p.Shoots
		shoots.Wear.Joints = slices.Clone(shoots.Wear.Joints)
		p.Shoots = &shoots
	}
	return p, tree, nil
}

func (h HabitProfile) make(maximumPlantFigure float64) (branching.Habit, error) {
	values := []float64{h.Turn, h.Start, h.Top, h.Thinning, h.Stray, h.Low, h.Lift, h.Sway, h.Reach, h.Shortening, h.Uneven, h.Girth, h.Least, h.Own}
	for _, n := range values {
		if !finite(n) || math.Abs(n) > maximumPlantFigure {
			return branching.Habit{}, fmt.Errorf("invalid branch habit")
		}
	}
	if h.Fewest < 1 || h.Fewest > 64 {
		return branching.Habit{}, fmt.Errorf("invalid minimum branches")
	}
	made := branching.Habit{Turn: measure.Radians(h.Turn), Start: h.Start, Top: h.Top, Thinning: h.Thinning,
		Fewest: h.Fewest, Stray: h.Stray, Low: measure.Degrees(h.Low), Lift: measure.Degrees(h.Lift),
		Sway: measure.Degrees(h.Sway), Reach: h.Reach, Shortening: h.Shortening, Uneven: h.Uneven,
		Girth: h.Girth, Least: h.Least, Own: h.Own}
	if err := made.Sound(); err != nil {
		return branching.Habit{}, err
	}
	return made, nil
}

func (s SleeveProfile) make(maximumPlantFigure float64) (Round, error) {
	if s.Sides < 3 || s.Sides > 64 || len(s.Joints) < 2 || len(s.Joints) > 9 {
		return Round{}, fmt.Errorf("invalid sleeve")
	}
	joints := make([]branching.Joint, len(s.Joints))
	for i, p := range s.Joints {
		if !finite(p[0]) || !finite(p[1]) || p[0] < 0 || p[1] < 0 || p[0] > maximumPlantFigure || p[1] > maximumPlantFigure {
			return Round{}, fmt.Errorf("invalid sleeve joint")
		}
		joints[i] = branching.Joint{OfGirth: p[0], OfLength: p[1]}
	}
	if !branching.Jointed(joints) {
		return Round{}, fmt.Errorf("invalid sleeve joints")
	}
	return Round{Joints: joints, Sides: s.Sides}, nil
}

func (l LimbProfile) make() (Limb, error) {
	if l.Pieces < 1 || l.Pieces > 8 || !finite(l.Droop) || l.Droop < 0 || l.Droop > 10 || !finite(l.Wander) || l.Wander < 0 || l.Wander > 10 {
		return Limb{}, fmt.Errorf("invalid limb")
	}
	return Limb{Pieces: l.Pieces, Droop: l.Droop, Wander: l.Wander}, nil
}

func coniferSpan(pair [2]float64, high float64, whole bool) (rangeValue, error) {
	if !positive(pair[0]) || pair[0] < 1e-9 || !positive(pair[1]) || pair[1] < pair[0] || pair[1] > high || (whole && (math.Trunc(pair[0]) != pair[0] || math.Trunc(pair[1]) != pair[1])) {
		return rangeValue{}, fmt.Errorf("invalid conifer range")
	}
	return rangeValue{least: pair[0], most: pair[1]}, nil
}

func (p Profile) tree(value func(string) float64, maximumPlantFigure float64) (Tree, error) {
	height, e1 := coniferSpan(p.Height, 1000, false)
	slender, e2 := coniferSpan(p.Slenderness, 1, false)
	between, e3 := coniferSpan(p.Between, 1000, false)
	lowest, e4 := coniferSpan(p.Lowest, 64, true)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
		height.most/between.least > 256 || height.least/between.most < 0.5 {
		return Tree{}, fmt.Errorf("invalid tree dimensions or whorl budget")
	}
	h, err := p.Habit.make(maximumPlantFigure)
	if err != nil {
		return Tree{}, err
	}
	trunk, err := p.Trunk.make(maximumPlantFigure)
	if err != nil {
		return Tree{}, err
	}
	bark, err := p.Bark.make(maximumPlantFigure)
	if err != nil {
		return Tree{}, err
	}
	clad, err := p.Clad.make(maximumPlantFigure)
	if err != nil {
		return Tree{}, err
	}
	branch, err := p.Branch.make()
	if err != nil {
		return Tree{}, err
	}
	if len(trunk.Joints) != 2 || len(bark.Joints) != branch.Pieces+1 || len(clad.Joints) != branch.Pieces+1 {
		return Tree{}, fmt.Errorf("limb and sleeve joints disagree")
	}
	if !finite(p.Flare) || p.Flare < 0 || p.Flare > 100 || p.BarkName == "" || p.LeafName == "" {
		return Tree{}, fmt.Errorf("invalid appearance")
	}
	for _, c := range append(p.BarkColour[:], p.LeafColour[:]...) {
		if !finite(c) || c < 0 || c > 1 {
			return Tree{}, fmt.Errorf("invalid colour")
		}
	}
	grow := Architecture{Trunk: trunk, Bark: bark, Branch: branch, Clad: clad}
	maxSprays := 0
	if p.Shoots != nil {
		each := p.Shoots
		per, err := coniferSpan(each.Per, 32, true)
		if err != nil {
			return Tree{}, err
		}
		shoot, err := each.Shoot.make()
		if err != nil {
			return Tree{}, err
		}
		set, err := each.Habit.make(maximumPlantFigure)
		if err != nil {
			return Tree{}, err
		}
		wear, err := each.Wear.make(maximumPlantFigure)
		if err != nil {
			return Tree{}, err
		}
		if len(wear.Joints) != shoot.Pieces+1 {
			return Tree{}, fmt.Errorf("shoot and foliage joints disagree")
		}
		grow.Shoots = &Branchlets{Per: [2]float64{per.least, per.most}, Shoot: shoot, Set: set, Wear: wear}
		maxSprays = int(per.most) * max(1, set.Fewest) * shoot.Pieces
	}
	whorls := int(math.Ceil(height.most / between.least))
	// Each branch piece carries bark and clad; a shoot whorl can carry
	// more than one shoot when its branching.Habit says a higher minimum.
	parts := int64(whorls) * int64(max(int(lowest.most), h.Fewest)) *
		int64(branch.Pieces) * int64(2+maxSprays)
	if parts > 100000 {
		return Tree{}, fmt.Errorf("conifer exceeds branch budget")
	}
	if err := p.Needle.valid(p.Foliage, value); err != nil {
		return Tree{}, err
	}
	style := 0
	switch p.Foliage {
	case "needles":
	case "flat-spray":
		style = 1
	case "scales":
		style = 2
	default:
		return Tree{}, fmt.Errorf("unknown foliage %q", p.Foliage)
	}
	return Tree{Tall: [2]float64{height.least, height.most}, Slender: [2]float64{slender.least, slender.most},
		Between: [2]float64{between.least, between.most}, Lowest: [2]float64{lowest.least, lowest.most},
		Hangs: h, Grows: grow, FoliageStyle: style, Flare: p.Flare}, nil
}

// geometry excludes provenance, which does not change a mesh or cache key.
func (n NeedleProfile) Geometry() ShootProfile {
	return ShootProfile{Low: n.Length[0], High: n.Length[1], Width: n.Width,
		Thickness: n.Thickness, Spacing: n.Spacing, Angle: n.Angle,
		Rows: n.Rows, WoodSides: n.WoodSides}
}

func (n NeedleProfile) valid(form string, value func(string) float64) error {
	if n.LengthStatus != "picked" && n.LengthStatus != "read" && n.LengthStatus != "derived" {
		return fmt.Errorf("invalid foliage provenance")
	}
	if (n.LengthStatus == "read" || n.LengthStatus == "derived") && n.LengthSource == "" {
		return fmt.Errorf("foliage length needs a source")
	}
	if _, err := coniferSpan(n.Length, 1, false); err != nil {
		return fmt.Errorf("invalid foliage length")
	}
	if !positive(n.Width) || n.Width > 1 || !positive(n.Thickness) || n.Thickness > 1 ||
		!positive(n.Spacing) || n.Spacing > 1 || !finite(n.Angle) || n.Angle < 0 || n.Angle > 180 ||
		n.Rows < 1 || n.Rows > 8 || n.WoodSides < 3 || n.WoodSides > 64 || n.WoodSides%n.Rows != 0 ||
		(form == "flat-spray" && n.Rows != 2) {
		return fmt.Errorf("invalid foliage dimensions")
	}
	// Leaves share coordinates with a physical shoot cell. Sub-resolution
	// lengths or cross sections collapse to zero-area mesh triangles.
	minimum := math.Max(1e-9, math.Max(value("shoot-cell-length"), n.Length[1])*1e-8)
	if n.Length[0] < minimum || n.Width < minimum || n.Thickness < minimum {
		return fmt.Errorf("foliage dimensions too small for stable geometry")
	}
	if math.Ceil(value("shoot-cell-length")/n.Spacing)*float64(n.Rows) > 4096 {
		return fmt.Errorf("foliage exceeds its needle budget")
	}
	return nil
}

// verifiedReferences connects named measured records only when both their
// source and numeric values agree with the actual profile dimensions.
// A profile without references remains valid but is not counted as measured
// by the existing named-number maturity report.
func (n NeedleProfile) verifiedReferences() ([]numbers.Number, error) {
	if n.LengthReferences == [2]string{} {
		return nil, nil
	}
	if n.LengthStatus != "read" || n.LengthReferences[0] == "" || n.LengthReferences[1] == "" {
		return nil, fmt.Errorf("foliage length references require two read values")
	}
	records, err := research.Read()
	if err != nil {
		return nil, err
	}
	result := make([]numbers.Number, 0, 2)
	for i, name := range n.LengthReferences {
		value, err := records.Get(name)
		if err != nil {
			return nil, fmt.Errorf("unknown foliage reference %q", name)
		}
		if value.Kind != numbers.Measured || value.From == nil || value.From.Key != n.LengthSource {
			return nil, fmt.Errorf("foliage reference %q has a different source", name)
		}
		metres := value.Value
		switch value.Unit {
		case "m":
		case "cm":
			metres /= 100
		case "mm":
			metres /= 1000
		default:
			return nil, fmt.Errorf("foliage reference %q has unsupported unit %q", name, value.Unit)
		}
		if metres != n.Length[i] {
			return nil, fmt.Errorf("foliage reference %q has a different value", name)
		}
		result = append(result, *value)
	}
	return result, nil
}

func (p Profile) Notes() (measured, picked []numbers.Number) {
	measured = append([]numbers.Number(nil), p.Needle.references...)
	for i := range measured {
		if measured[i].From != nil {
			source := *measured[i].From
			measured[i].From = &source
		}
	}
	if p.Status == "picked" {
		for name, value := range map[string]float64{"width": p.Needle.Width, "thickness": p.Needle.Thickness,
			"spacing": p.Needle.Spacing, "angle": p.Needle.Angle, "rows": float64(p.Needle.Rows), "wood-sides": float64(p.Needle.WoodSides)} {
			picked = append(picked, numbers.Number{Name: p.Kind + ".needle." + name, Value: value,
				Kind: numbers.Picked, By: p.By + ", " + p.Date})
		}
	}
	if p.Needle.LengthStatus == "picked" {
		for i, suffix := range []string{"minimum", "maximum"} {
			picked = append(picked, numbers.Number{Name: p.Kind + ".needle." + suffix, Value: p.Needle.Length[i],
				Kind: numbers.Picked, By: p.By + ", " + p.Date})
		}
	}
	return measured, picked
}
