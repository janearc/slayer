package main

import (
	_ "embed"
	"fmt"
	"math"
	"slices"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
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
	Group      string      `json:"group" yaml:"group"`
	About      string      `json:"about" yaml:"about"`
	Notes      string      `json:"notes" yaml:"notes"`
	Details    []string    `json:"details" yaml:"details"`
	References []reference `json:"references" yaml:"references"`
	Yaw        float64     `json:"view-yaw-deg" yaml:"view-yaw-deg"`
	Up         string      `json:"view-up,omitempty" yaml:"view-up"`
	Pitch      *float64    `json:"view-elevation-deg" yaml:"view-elevation-deg"`
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
	var made menu
	err := readDocument("menu.yaml", menuRaw, menuSchema, &made)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, entry := range made.Entries {
		if _, err := manufacture.KindNamed(entry.Kind); err != nil {
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
