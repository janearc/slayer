// Package regionset reads the rectangles used to compare part of a picture.
package regionset

import (
	"fmt"
	"io"
	"math"
	"strings"

	"gopkg.in/yaml.v3"
)

// Set identifies the job, count threshold and regions to compare.
type Set struct {
	Job       string
	Threshold float64
	Regions   []Region
}

// Region is one rectangle in the complete frame, with its purpose.
type Region struct {
	Name, Observation        string
	Left, Top, Width, Height int
}

// Read reads exactly one set and checks it against the effective frame size.
func Read(input io.Reader, width, height int) (Set, error) {
	if input == nil || width <= 0 || height <= 0 {
		return Set{}, fmt.Errorf("regionset: input and positive frame dimensions are required")
	}
	decoder := yaml.NewDecoder(input)
	decoder.KnownFields(true)
	var document encodedSet
	if err := decoder.Decode(&document); err != nil {
		return Set{}, fmt.Errorf("regionset: read set: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return Set{}, fmt.Errorf("regionset: read after set: %w", err)
		}
		return Set{}, fmt.Errorf("regionset: only one document is allowed")
	}
	return document.checked(width, height)
}

// Pointer fields distinguish an explicitly supplied zero from a missing key.
type encodedSet struct {
	Job       *string         `yaml:"job"`
	Threshold *float64        `yaml:"threshold"`
	Regions   []encodedRegion `yaml:"regions"`
}

type encodedRegion struct {
	Name        *string `yaml:"name"`
	Observation *string `yaml:"observation"`
	Left        *int    `yaml:"left"`
	Top         *int    `yaml:"top"`
	Width       *int    `yaml:"width"`
	Height      *int    `yaml:"height"`
}

// checked validates a decoded set and its rectangles.
func (document encodedSet) checked(width, height int) (Set, error) {
	if document.Job == nil || strings.TrimSpace(*document.Job) == "" {
		return Set{}, fmt.Errorf("regionset: job is required")
	}
	if document.Threshold == nil || math.IsNaN(*document.Threshold) ||
		math.IsInf(*document.Threshold, 0) || *document.Threshold <= 0 {
		return Set{}, fmt.Errorf("regionset: threshold must be finite and positive")
	}
	if len(document.Regions) == 0 {
		return Set{}, fmt.Errorf("regionset: at least one region is required")
	}
	result := Set{Job: *document.Job, Threshold: *document.Threshold}
	names := make(map[string]bool, len(document.Regions))
	for index, entry := range document.Regions {
		region, err := entry.checked(width, height)
		if err != nil {
			return Set{}, fmt.Errorf("regionset: region %d: %w", index+1, err)
		}
		if names[region.Name] {
			return Set{}, fmt.Errorf("regionset: duplicate region %q", region.Name)
		}
		names[region.Name] = true
		result.Regions = append(result.Regions, region)
	}
	return result, nil
}

// checked validates one decoded rectangle against the frame.
func (entry encodedRegion) checked(width, height int) (Region, error) {
	if entry.Name == nil || *entry.Name == "" || strings.TrimSpace(*entry.Name) != *entry.Name {
		return Region{}, fmt.Errorf("region name must be nonempty without surrounding whitespace")
	}
	if entry.Observation == nil || strings.TrimSpace(*entry.Observation) == "" {
		return Region{}, fmt.Errorf("region %q needs an observation", *entry.Name)
	}
	if entry.Left == nil || entry.Top == nil || entry.Width == nil || entry.Height == nil {
		return Region{}, fmt.Errorf("region %q needs all four coordinates", *entry.Name)
	}
	left, top, across, down := *entry.Left, *entry.Top, *entry.Width, *entry.Height
	if left < 0 || top < 0 || across <= 0 || down <= 0 ||
		left >= width || top >= height || across > width-left || down > height-top {
		return Region{}, fmt.Errorf("region %q is outside the frame", *entry.Name)
	}
	return Region{Name: *entry.Name, Observation: *entry.Observation,
		Left: left, Top: top, Width: across, Height: down}, nil
}
