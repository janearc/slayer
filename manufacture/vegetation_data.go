package manufacture

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

//go:embed plant-profiles.yaml
var plantProfilesRaw []byte

//go:embed plant-choices.yaml
var plantChoicesRaw []byte

var plantChoices = readPlantChoices(plantChoicesRaw)
var plantProfiles = readPlantProfiles(plantProfilesRaw)

// plantProfile is a private, authored construction book, not a growth survey.
type plantProfile struct {
	Kind, Form, Status, By, Date                  string
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
}

// readPlantProfiles checks the authored status and names before registration.
func readPlantProfiles(raw []byte) map[string]plantProfile {
	var book struct{ Profiles []plantProfile }
	if err := yamlfile.Read("plant-profiles.yaml", raw, &book); err != nil {
		panic(err)
	}
	profiles := map[string]plantProfile{}
	for _, profile := range book.Profiles {
		if profile.Status != "picked" || profile.By == "" || profile.Date == "" ||
			profile.Kind == "" || profiles[profile.Kind].Kind != "" {
			panic(fmt.Errorf("manufacture: invalid authored plant profile %q", profile.Kind))
		}
		if err := profile.valid(); err != nil {
			panic(err)
		}
		profiles[profile.Kind] = profile
	}
	return profiles
}

// readPlantChoices keeps shared numerical policies in Puzzlemouth data.
func readPlantChoices(raw []byte) numbers.Numbers {
	book, err := numbers.Read("plant-choices.yaml", raw)
	if err != nil {
		panic(err)
	}
	return book
}

// plantValue reads a picked construction or detail policy.
func plantValue(name string) float64 {
	value, err := plantChoices.Get(name)
	if err != nil {
		panic(err)
	}
	return value.Value
}
