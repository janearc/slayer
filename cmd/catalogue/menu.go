package main

import (
	_ "embed"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
)

//go:embed menu.yaml
var menuRaw []byte

//go:embed menu.schema.json
var menuSchema []byte

type menu struct {
	Entries []entry `json:"entries" yaml:"entries"`
}

type reference struct {
	Title string `json:"title" yaml:"title"`
	URL   string `json:"url" yaml:"url"`
}

type entry struct {
	Kind       string      `json:"kind" yaml:"kind"`
	Animal     *animalSpec `json:"animal,omitempty" yaml:"animal"`
	Plant      *plantSpec  `json:"plant,omitempty" yaml:"plant"`
	Alga       *algaSpec   `json:"alga,omitempty" yaml:"alga"`
	Group      string      `json:"group" yaml:"group"`
	About      string      `json:"about" yaml:"about"`
	Notes      string      `json:"notes" yaml:"notes"`
	Details    []string    `json:"details" yaml:"details"`
	References []reference `json:"references" yaml:"references"`
	Yaw        float64     `json:"view-yaw-deg" yaml:"view-yaw-deg"`
	Up         string      `json:"view-up,omitempty" yaml:"view-up"`
	Pitch      *float64    `json:"view-elevation-deg" yaml:"view-elevation-deg"`
	Staging    *staging    `json:"staging,omitempty" yaml:"staging"`
	checked    *manufacture.Kind
	source     string
}

// staging is an entry's own sky and lamps, lit in place of the plan's
// studio: a spacecraft in space, say, with one sun and nothing behind it.
type staging struct {
	Sky     sky.Gradient `json:"sky" yaml:"sky"`
	Sources []source     `json:"sources" yaml:"sources"`
}

// staged is the plan an entry is lit by: the plan, with the entry's own
// sky and lamps in place of the studio's when it has them, unless the
// plan asks for its studio whatever the entry has.
func (entry entry) staged(plan plan) plan {
	if entry.Staging != nil && !plan.Studio && plan.Presentation == "" {
		plan.Sky = entry.Staging.Sky
		plan.Sources = entry.Staging.Sources
	}
	return plan
}

// camera raises a deep optical tube into view without changing its heading.
func (entry entry) camera(direction maths.Vec) maths.Vec {
	if entry.Pitch != nil {
		horizontal := math.Hypot(direction.X, direction.Z)
		direction.Y = horizontal * math.Tan(*entry.Pitch*maths.Degree)
	}
	return direction
}

// vertical lets a loose horizontal shoot read upright in the specimen book.
func (entry entry) vertical() maths.Vec {
	if entry.Up == "x" {
		return maths.XAxis
	}
	return maths.YAxis
}

// readMenu checks both the schema and the actual factory vocabulary.
func readMenu() (*menu, error) {
	return parseMenuWithModels("menu.yaml", menuRaw, "")
}

func menuForCall(call pzm.Call) (*menu, error) {
	if !call.Given("models") {
		return readMenu()
	}
	if strings.TrimSpace(call.Value("models")) == "" {
		return nil, fmt.Errorf("models directory must not be empty")
	}
	root, err := filepath.Abs(call.Value("models"))
	if err != nil {
		return nil, err
	}
	return parseMenuWithModels("menu.yaml", menuRaw, root)
}

// parseMenu is readMenu's checks over the bytes of any menu.
func parseMenu(name string, raw []byte) (*menu, error) {
	return parseMenuWithModels(name, raw, "")
}

// parseMenuWithModels resolves every profile once and retains its checked
// constructor and source identity for the lifetime of the returned menu.
func parseMenuWithModels(name string, raw []byte, modelRoot string) (*menu, error) {
	var made menu
	err := readDocument(name, raw, menuSchema, &made)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	books := &modelBooks{root: modelRoot, bytes: map[string][]byte{}}
	for index := range made.Entries {
		entry := &made.Entries[index]
		if strings.TrimSpace(entry.About) == "" {
			return nil, fmt.Errorf("catalogue kind %q needs an about description", entry.Kind)
		}
		if strings.TrimSpace(entry.Notes) == "" {
			return nil, fmt.Errorf("catalogue kind %q needs notes", entry.Kind)
		}
		if entry.Animal != nil && (entry.Plant != nil || entry.Alga != nil) || entry.Plant != nil && entry.Alga != nil {
			return nil, fmt.Errorf("catalogue %q cannot name multiple profile categories", entry.Kind)
		}
		if entry.Animal != nil {
			if err := entry.bindAnimal(books); err != nil {
				return nil, fmt.Errorf("catalogue %q: %w", entry.Kind, err)
			}
		} else if entry.Plant != nil {
			if err := entry.bindPlant(books); err != nil {
				return nil, fmt.Errorf("catalogue %q: %w", entry.Kind, err)
			}
		} else if entry.Alga != nil {
			if err := entry.bindAlga(books); err != nil {
				return nil, fmt.Errorf("catalogue %q: %w", entry.Kind, err)
			}
		} else if _, err := manufacture.KindNamed(entry.Kind); err != nil {
			return nil, err
		}
		if names[entry.Kind] {
			return nil, fmt.Errorf("duplicate catalogue kind %q",
				entry.Kind)
		}
		names[entry.Kind] = true
		levels := manufacture.Complexities()
		for _, detail := range entry.Details {
			if !slices.Contains(levels, detail) {
				return nil, fmt.Errorf("unknown detail %q",
					detail)
			}
		}
		if entry.Staging != nil {
			if err := checkSources(entry.Staging.Sources); err != nil {
				return nil, fmt.Errorf("catalogue kind %q is staged "+
					"with %w", entry.Kind, err)
			}
		}
	}
	for _, name := range manufacture.KindNames() {
		if !names[name] {
			return nil, fmt.Errorf("catalogue is missing kind %q",
				name)
		}
	}
	return &made, nil
}

// selectEntries preserves menu order and refuses every unknown selector.
func (menu menu) selectEntries(names []string) ([]entry, error) {
	if len(names) == noEntries {
		return slices.Clone(menu.Entries), nil
	}
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[name] = true
	}
	var selected []entry
	for _, entry := range menu.Entries {
		if wanted[entry.Kind] {
			selected = append(selected, entry)
			delete(wanted, entry.Kind)
		}
	}
	if len(wanted) != noEntries {
		var unknown []string
		for name := range wanted {
			unknown = append(unknown, name)
		}
		slices.Sort(unknown)
		return nil, fmt.Errorf("unknown catalogue kinds: %v", unknown)
	}
	return selected, nil
}

// readDocument keeps YAML and JSON input on the same Puzzlemouth schema path.
func readDocument(name string, raw, rules []byte, target any) error {
	schema, err := yamlfile.NewSchema("catalogue.schema.json", rules)
	if err != nil {
		return err
	}
	node, err := schema.Read(name, raw)
	if err != nil {
		return err
	}
	return node.Decode(target)
}
