package seasonal

import (
	_ "embed"
	"fmt"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
	"gopkg.in/yaml.v3"
	"maps"
	"slices"
	"strings"
)

//go:embed profiles.yaml
var raw []byte

func Embedded() []byte { return append([]byte(nil), raw...) }

type Policy struct {
	Value func(string) float64
	Sow   func(uint64) branching.Seed
}

func (profile Profile) value(name string) float64 { return profile.policy.Value(name) }

// Profile is a private, authored construction book, not a growth survey.
type Profile struct {
	cells                                         *plantCellStore
	policy                                        *Policy
	Kind, Form, Status, By, Date, Source          string
	Scale                                         struct{ Minimum, Maximum float64 }
	Slenderness, Trunk, Reach, Contraction, Taper float64
	RadiusPower                                   float64 `yaml:"radius-power"`
	Fork, Twist, Upward                           float64
	LeaderGain                                    float64 `yaml:"leader-gain"`
	StemSpread                                    float64 `yaml:"stem-spread"`
	Orders, Children, Stems                       int
	Leader, Deciduous                             bool
	Bark, Green, Autumn, Dry                      maths.Light
	LeafShape                                     string  `yaml:"leaf-shape"`
	LeafLength                                    float64 `yaml:"leaf-length"`
	LeafWidth                                     float64 `yaml:"leaf-width"`
	LeafThickness                                 float64 `yaml:"leaf-thickness"`
	Lobes                                         int
	LobeDepth                                     float64 `yaml:"lobe-depth"`
	CellLength                                    float64 `yaml:"cell-length"`
	CellSpacing                                   float64 `yaml:"cell-spacing"`
	TwigCount                                     int     `yaml:"twig-count"`
	TwigLength                                    float64 `yaml:"twig-length"`
	TwigRadius                                    float64 `yaml:"twig-radius"`
	LeafPairs                                     int     `yaml:"leaf-pairs"`
	RosetteLeaves                                 int     `yaml:"rosette-leaves"`
	RosetteAngle                                  float64 `yaml:"rosette-angle"`
	Roots                                         *Roots
}

// Book is a checked book of species profiles. Its data is private
// so a constructed plant is independent of later edits to the source bytes.
type Book struct{ profiles map[string]Profile }

// Read reads an authored profile book without registering its
// species globally. A malformed, empty or oversized book is refused.
func Read(file string, raw []byte, policy *Policy) (*Book, error) {
	if policy == nil || policy.Value == nil || policy.Sow == nil {
		return nil, fmt.Errorf("manufacture: incomplete seasonal plant policy")
	}
	snapshot := *policy
	policy = &snapshot
	rows, err := profilecheck.PlantRows[Profile](file, raw, optionalPlantNumber)
	if err != nil {
		return nil, err
	}
	profiles := make(map[string]Profile, len(rows))
	cells := &plantCellStore{}
	for _, profile := range rows {
		if !profilecheck.Authored(profile.Kind, profile.Status, profile.By, profile.Date, profile.Source) ||
			profiles[profile.Kind].Kind != "" {
			return nil, fmt.Errorf("manufacture: %s: invalid authored plant profile %q", file, profile.Kind)
		}
		profile.policy = policy
		if profile.Roots != nil {
			profile.Roots.policy = policy
		}
		if err := profile.valid(); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		profile.cells = cells
		profiles[profile.Kind] = profile
	}
	return &Book{profiles: profiles}, nil
}

// Fields for the other foliage form need not be written. A non-leader also
// has no leader gain; explicit null is refused even for optional fields.
func optionalPlantNumber(profile *yaml.Node, path string) bool {
	leader := profilecheck.Alias(profilecheck.Field(profile, "leader"))
	if path == "leader-gain" && (leader == nil || strings.EqualFold(leader.Value, "false")) {
		return true
	}
	form := profilecheck.Alias(profilecheck.Field(profile, "form"))
	if form != nil {
		switch form.Value {
		case "twig":
			return path == "rosette-leaves" || path == "rosette-angle"
		case "rosette":
			return path == "twig-count" || path == "twig-length" || path == "leaf-pairs"
		}
	}
	return false
}

func (book *Book) Get(name string) (Profile, bool) {
	if book == nil {
		return Profile{}, false
	}
	p, ok := book.profiles[name]
	if p.Roots != nil {
		roots := *p.Roots
		p.Roots = &roots
	}
	return p, ok
}
func (book *Book) Species() []string {
	if book == nil {
		return nil
	}
	return slices.Sorted(maps.Keys(book.profiles))
}
