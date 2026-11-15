package job

import (
	_ "embed"
	"slices"

	"github.com/janearc/puzzlemouth/yamlfile"
)

//go:embed presets.yaml
var presetsRaw []byte

// shelf is the presets as presets.yaml writes them, in its order.
type shelf struct {
	Presets []presetWire `yaml:"presets"`
}

type presetWire struct {
	Name   string `yaml:"name"`
	Width  int    `yaml:"width"`
	Height int    `yaml:"height"`
	Rate   int    `yaml:"rate"`
	By     string `yaml:"by"`
}

// presets is the table, read once.
var presets, presetsError = readPresets("presets.yaml", presetsRaw)

// readPresets reads a presets table, refusing a key it does not know.
func readPresets(name string, raw []byte) ([]Preset, error) {
	var read shelf
	if err := yamlfile.Read(name, raw, &read); err != nil {
		return nil, err
	}
	table := make([]Preset, len(read.Presets))
	for i, each := range read.Presets {
		table[i] = Preset{name: each.Name, width: each.Width,
			height: each.Height, rate: each.Rate, by: each.By}
	}
	return table, nil
}

// Preset is a named film: a picture's size in pixels and, for some, a
// rate in frames a second, and who picked it and when, or where it came
// from.
type Preset struct {
	name          string
	width, height int
	rate          int
	by            string
}

// Presets is the table of presets, in the order presets.yaml writes it,
// for the editor and for a listing.
func Presets() []Preset { return slices.Clone(presets) }

// Name is the preset's name, as a job's film writes it.
func (made Preset) Name() string { return made.name }

// Size is the preset's width and height, in pixels.
func (made Preset) Size() (width, height int) {
	return made.width, made.height
}

// Rate is the preset's frames a second, and whether it gives one.
func (made Preset) Rate() (int, bool) {
	return made.rate, made.rate != nothing
}

// By is who picked the preset and when, or where it came from.
func (made Preset) By() string { return made.by }

// presetNamed is the preset of that name, and whether there is one.
func presetNamed(name string) (Preset, bool) {
	at := slices.IndexFunc(presets, func(each Preset) bool {
		return each.name == name
	})
	if at < nothing {
		return Preset{}, false
	}
	return presets[at], true
}

// preset lays the film's preset under what the job says: its size where
// the job says none, and its rate where a film says none.
func (wire *document) preset() {
	if wire.Film == nil {
		return
	}
	chosen, found := presetNamed(wire.Film.Preset)
	if !found {
		return
	}
	if wire.Film.Width == nothing {
		wire.Film.Width = chosen.width
	}
	if wire.Film.Height == nothing {
		wire.Film.Height = chosen.height
	}
	if wire.film() && wire.FPS == nothing {
		wire.FPS = chosen.rate
	}
}
