package preview

import (
	"fmt"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/ferruccio/internal/assets"
	"gopkg.in/yaml.v3"
)

type outputPreset struct {
	Name   string `yaml:"name"`
	Width  int    `yaml:"width"`
	Height int    `yaml:"height"`
	Rate   int    `yaml:"rate"`
	By     string `yaml:"by"`
}

var outputPresets, outputPresetError = loadOutputPresets()

// loadOutputPresets reads the pinned producer table.
func loadOutputPresets() ([]outputPreset, error) {
	const name = "job-presets.yaml"
	raw, err := assets.Read(name)
	if err != nil {
		return nil, err
	}
	var table struct {
		Presets []outputPreset `yaml:"presets"`
	}
	if err := yamlfile.Read(name, raw, &table); err != nil {
		return nil, err
	}
	return table.Presets, nil
}

// readOutputPreset accepts bare numeric names the same way as the producer.
func readOutputPreset(node *yaml.Node) (outputPreset, error) {
	var preset outputPreset
	if node == nil {
		return preset, nil
	}
	if outputPresetError != nil {
		return preset, outputPresetError
	}
	var name string
	if err := node.Decode(&name); err != nil {
		return preset, err
	}
	for _, size := range outputPresets {
		if size.Name == name {
			return size, nil
		}
	}
	return preset, fmt.Errorf("unknown authored film preset %q", name)
}
