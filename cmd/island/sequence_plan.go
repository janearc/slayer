package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/maths"
)

//go:embed sequence.schema.json
var sequenceSchema []byte

type sequenceKey struct {
	Seconds float64  `yaml:"at-seconds" json:"at-seconds"`
	At      location `yaml:"at" json:"at"`
	Look    location `yaml:"look-at" json:"look-at"`
	Wide    float64  `yaml:"wide-deg" json:"wide-deg"`
	Ease    string   `yaml:"ease" json:"ease"`
}

type sequencePlan struct {
	Scene    string        `yaml:"scene" json:"scene"`
	Frames   int           `yaml:"frames" json:"frames"`
	FPS      int           `yaml:"fps" json:"fps"`
	Duration float64       `yaml:"scene-duration-seconds" json:"scene-duration-seconds"`
	Keys     []sequenceKey `yaml:"keyframes" json:"keyframes"`
	raw      []byte
	hash     string
	zooms    []*camera.Zoom
}

// readSequence checks the track schema, endpoint clocks and non-wrapping legs,
// then prepares each segment's angular zoom interpolation.
func readSequence(path string) (*sequencePlan, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(absolute)
	if err != nil {
		return nil, err
	}
	schema, err := yamlfile.NewSchema("island-sequence-v1.json", sequenceSchema)
	if err != nil {
		return nil, err
	}
	node, err := schema.Read(path, raw)
	if err != nil {
		return nil, err
	}
	var plan sequencePlan
	if err := node.Decode(&plan); err != nil {
		return nil, err
	}
	if plan.Keys[0].Seconds != 0 || plan.Keys[len(plan.Keys)-1].Seconds != plan.Duration {
		return nil, fmt.Errorf("island: keyframes must run from zero to scene duration")
	}
	for index, from := range plan.Keys[:len(plan.Keys)-1] {
		to := plan.Keys[index+1]
		if to.Seconds <= from.Seconds || math.Abs(to.At.Longitude-from.At.Longitude) > 180 || math.Abs(to.Look.Longitude-from.Look.Longitude) > 180 {
			return nil, fmt.Errorf("island: increasing times and non-wrapping geographic segments required")
		}
		zoom, err := camera.NewZoom(from.Wide, to.Wide, time.Duration((to.Seconds-from.Seconds)*float64(time.Second)), from.Ease)
		if err != nil {
			return nil, err
		}
		plan.zooms = append(plan.zooms, zoom)
	}
	if !filepath.IsAbs(plan.Scene) {
		plan.Scene = filepath.Join(filepath.Dir(absolute), plan.Scene)
	}
	plan.raw, plan.hash = raw, configHash(raw)
	return &plan, nil
}

// sample interpolates longitude, latitude and radial height independently.
// This is an authored geographic trajectory, not Cartesian constant-speed
// travel, an orbital solution or a terrain-avoidance planner. Up is rebuilt
// radially by the renderer at every sample rather than held at the first view.
func (sequence sequencePlan) sample(base *plan, index int) (*plan, error) {
	seconds := sequence.Duration * float64(index) / float64(sequence.Frames-1)
	if index == sequence.Frames-1 {
		seconds = sequence.Duration
	}
	leg := len(sequence.Keys) - 2
	for i := 0; i < leg; i++ {
		if seconds < sequence.Keys[i+1].Seconds {
			leg = i
			break
		}
	}
	from, to := sequence.Keys[leg], sequence.Keys[leg+1]
	share := math.Max(0, math.Min(1, (seconds-from.Seconds)/(to.Seconds-from.Seconds)))
	if from.Ease == "smooth" {
		share = maths.Smooth(0, 1, share)
	}
	result := *base
	result.Camera.At = geographicBetween(from.At, to.At, share)
	result.Camera.Look = geographicBetween(from.Look, to.Look, share)
	wide, err := sequence.zooms[leg].At(time.Duration((seconds - from.Seconds) * float64(time.Second)))
	if err != nil {
		return nil, err
	}
	result.Film.Wide = wide
	result.Time = base.Time + seconds
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	result.raw = append(raw, '\n')
	result.hash = configHash(result.raw)
	return &result, nil
}

// geographicBetween interpolates each coordinate and preserves exact endpoints.
func geographicBetween(from, to location, share float64) location {
	if share <= 0 {
		return from
	}
	if share >= 1 {
		return to
	}
	return location{Longitude: from.Longitude + (to.Longitude-from.Longitude)*share,
		Latitude: from.Latitude + (to.Latitude-from.Latitude)*share,
		Height:   from.Height + (to.Height-from.Height)*share}
}

// configHash identifies exact configuration bytes, including their formatting.
func configHash(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}
