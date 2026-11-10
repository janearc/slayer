package main

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/yamlfile"
)

//go:embed catalogue.yaml
var catalogueRaw []byte

//go:embed catalogue.schema.json
var catalogueSchema []byte

// catalogue is the embedded preview menu, checked before use.
type catalogue struct {
	Preview struct {
		Width         int     `json:"width"`
		Height        int     `json:"height"`
		Rays          int     `json:"rays"`
		Cells         string  `json:"cells"`
		PixelAspect   float64 `json:"pixel-aspect" yaml:"pixel-aspect"`
		ShutdownGrace float64 `json:"shutdown-grace" yaml:"shutdown-grace"`
		SampleBudget  int64   `json:"sample-budget" yaml:"sample-budget"`
		Limits        struct {
			Width  int `json:"width"`
			Height int `json:"height"`
			Rays   int `json:"rays"`
		} `json:"limits"`
	} `json:"preview"`
	Demos []preset `json:"demos"`
}

// preset is one scene on the decoded wire; runtime state stays in preview.
type preset struct {
	Name       string   `json:"name"`
	Command    string   `json:"command"`
	About      string   `json:"about"`
	Plan       string   `json:"plan"`
	Arguments  []string `json:"arguments"`
	RawPreview bool     `json:"raw-preview" yaml:"raw-preview"`
	SceneSized bool     `json:"scene-sized" yaml:"scene-sized"`
}

// readCatalogue checks the embedded schema and refuses duplicate menu names.
func readCatalogue() (*catalogue, error) {
	schema, err := yamlfile.NewSchema(
		"https://github.com/janearc/slayer/schemas/demos-v1.json",
		catalogueSchema)
	if err != nil {
		return nil, err
	}
	node, err := schema.Read("catalogue.yaml", catalogueRaw)
	if err != nil {
		return nil, err
	}
	var menu catalogue
	if err := node.Decode(&menu); err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, preset := range menu.Demos {
		if names[preset.Name] {
			return nil, fmt.Errorf("duplicate demo %q", preset.Name)
		}
		names[preset.Name] = true
	}
	return &menu, nil
}
