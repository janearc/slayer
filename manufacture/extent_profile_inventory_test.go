package manufacture

import (
	"fmt"
	"testing"

	"github.com/janearc/slayer/maths"
	"gopkg.in/yaml.v3"
)

// assertPlannable checks declarations without growing or meshing a specimen.
func assertPlannable(t *testing.T, name string, row Kind, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return
	}
	if row.how == nil {
		t.Errorf("%s: no recipe", name)
		return
	}
	if _, ok := row.how.(interface{ extent(seed) (maths.Box, error) }); !ok && !declaredGreedy(row.how) {
		t.Errorf("%s: recipe %T has neither an extent nor a declared greedy strategy", name, row.how)
	}
}

// The sources rather than a hand-maintained species list drive this audit, so
// an added profile is checked even if it is deliberately absent from table().
func TestCheckedProfileKindsDeclarePlanningStrategy(t *testing.T) {
	for _, family := range []string{"seasonal", "conifer", "fern", "cactus"} {
		file, raw, err := EmbeddedPlantBook(family)
		if err != nil {
			t.Fatal(err)
		}
		book, err := ReadPlantBook(family, file, raw)
		if err != nil {
			t.Fatal(err)
		}
		// The checked book identifies exactly which conifers support a loose shoot.
		foliage := map[string]string{}
		if family == "conifer" {
			var source struct {
				Profiles []struct{ Kind, Foliage string } `yaml:"profiles"`
			}
			if err := yaml.Unmarshal(raw, &source); err != nil {
				t.Fatal(err)
			}
			for _, profile := range source.Profiles {
				foliage[profile.Kind] = profile.Foliage
			}
		}
		names := book.Species()
		if len(names) == 0 {
			t.Fatalf("%s is empty", family)
		}
		for _, name := range names {
			row, err := book.New(PlantOptions{Species: name})
			assertPlannable(t, family+"/"+name, row, err)
			if family == "conifer" {
				shoot := "shoot"
				row, err := book.New(PlantOptions{Species: name, Representation: &shoot})
				supported := foliage[name] == "flat-spray" || foliage[name] == "scales"
				if supported {
					assertPlannable(t, family+"/"+name+"/shoot", row, err)
				} else if err == nil {
					t.Errorf("%s: unexpected shoot for foliage %q", name, foliage[name])
				}
			}
			if family == "seasonal" {
				for _, season := range SeasonNames() {
					row, err := book.New(PlantOptions{Species: name, Season: &season})
					assertPlannable(t, family+"/"+name+"/"+season, row, err)
				}
			}
		}
	}
	for _, family := range []string{"fish", "bird", "pinniped", "theropod", "squid", "whale", "reptile", "snake", "sauropod"} {
		file, raw, err := EmbeddedAnimalBook(family)
		if err != nil {
			t.Fatal(err)
		}
		book, err := ReadAnimalBook(family, file, raw)
		if err != nil {
			t.Fatal(err)
		}
		var data struct {
			Species map[string]struct {
				Schools []int `yaml:"schools"`
			} `yaml:"species"`
			Profiles []struct {
				Kind    string               `yaml:"kind"`
				Studies map[string]yaml.Node `yaml:"studies"`
			} `yaml:"profiles"`
		}
		if err := yaml.Unmarshal(raw, &data); err != nil {
			t.Fatal(err)
		}
		names := make([]string, 0, len(data.Species)+len(data.Profiles))
		for name := range data.Species {
			names = append(names, name)
		}
		for _, profile := range data.Profiles {
			names = append(names, profile.Kind)
		}
		if len(names) == 0 {
			t.Fatalf("%s has no species", family)
		}
		for _, name := range names {
			check := func(label string, options AnimalOptions) {
				options.Species = name
				row, err := book.New(options)
				assertPlannable(t, family+"/"+name+"/"+label, row, err)
			}
			check("default", AnimalOptions{})
			switch family {
			case "fish":
				for _, pose := range []string{"swim", "glide", "bones"} {
					check(pose, AnimalOptions{Pose: &pose})
				}
				count := 2
				row, err := book.New(AnimalOptions{Species: name, Count: &count})
				if len(data.Species[name].Schools) > 0 {
					assertPlannable(t, family+"/"+name+"/school", row, err)
				} else if err == nil {
					t.Errorf("%s: accepted a school without school data", name)
				}
			case "bird":
				for _, pose := range []string{"standing", "gliding", "bones", "outline"} {
					check(pose, AnimalOptions{Pose: &pose})
				}
			case "pinniped":
				for _, representation := range []string{"skin", "bones", "volumes"} {
					row, err := book.New(AnimalOptions{Species: name, Representation: &representation})
					if err == nil {
						assertPlannable(t, family+"/"+name+"/"+representation, row, nil)
					} else if representation == "skin" || (representation == "bones" && (name == "harbor-seal" || name == "walrus")) || (representation == "volumes" && name == "walrus") {
						t.Errorf("%s/%s: %v", name, representation, err)
					}
				}
				for _, profile := range data.Profiles {
					if profile.Kind == name {
						for study := range profile.Studies {
							check(study, AnimalOptions{Study: &study})
						}
					}
				}
			case "whale":
				for _, pose := range []string{"swim", "spy-hop"} {
					check(pose, AnimalOptions{Pose: &pose})
				}
				bones := "bones"
				check(bones, AnimalOptions{Representation: &bones})
			case "reptile":
				studies := map[string][]string{"pliosaur": {"baseline", "continuous"}, "purussaurus": {"study"}, "hatzegopteryx": {"cage", "density"}}
				for _, study := range studies[name] {
					check(study, AnimalOptions{Study: &study})
				}
			}
		}
	}
	file, raw, err := EmbeddedAlgaBook("kelp")
	if err != nil {
		t.Fatal(err)
	}
	kelp, err := ReadKelpProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, condition := range []string{"slack", "moderate", "surge"} {
		for _, forest := range []bool{false, true} {
			row, err := kelp.NewKelp(KelpOptions{Species: "macrocystis-pyrifera", Condition: condition, Forest: forest})
			assertPlannable(t, fmt.Sprintf("kelp/%s/forest=%t", condition, forest), row, err)
		}
	}
}
