package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/island"
)

//go:embed scene.schema.json
var sceneSchema []byte

type location struct {
	Longitude float64 `yaml:"longitude-deg" json:"longitude-deg"`
	Latitude  float64 `yaml:"latitude-deg" json:"latitude-deg"`
	Height    float64 `yaml:"height-metres" json:"height-metres"`
}

// geographic gives the authored angles and radial height their frame units.
func (place location) geographic() frame.Location {
	return frame.Location{Longitude: measure.Degrees(place.Longitude),
		Latitude: measure.Degrees(place.Latitude), Height: measure.Metres(place.Height)}
}

type sourcePlan struct {
	File   string  `yaml:"file" json:"file"`
	SHA256 string  `yaml:"sha256" json:"sha256"`
	CRS    int     `yaml:"geographic-crs" json:"geographic-crs"`
	Datum  string  `yaml:"height-datum" json:"height-datum"`
	Offset float64 `yaml:"height-offset-metres" json:"height-offset-metres"`
}

type plan struct {
	Model        string       `yaml:"height-model" json:"height-model"`
	Radius       float64      `yaml:"radius-metres" json:"radius-metres"`
	Anchor       location     `yaml:"anchor" json:"anchor"`
	Sources      []sourcePlan `yaml:"sources" json:"sources"`
	RasterBudget struct {
		CacheBytes    int64 `yaml:"cache-total-bytes" json:"cache-total-bytes"`
		CacheTiles    int   `yaml:"cache-tiles-per-source" json:"cache-tiles-per-source"`
		TileBytes     int64 `yaml:"tile-bytes" json:"tile-bytes"`
		MetadataBytes int64 `yaml:"metadata-bytes" json:"metadata-bytes"`
		Index         *struct {
			Block int   `yaml:"block-side" json:"block-side"`
			Bytes int64 `yaml:"total-bytes" json:"total-bytes"`
		} `yaml:"height-index" json:"height-index,omitempty"`
	} `yaml:"raster-budget" json:"raster-budget"`
	Region struct {
		West  float64 `yaml:"west" json:"west"`
		South float64 `yaml:"south" json:"south"`
		East  float64 `yaml:"east" json:"east"`
		North float64 `yaml:"north" json:"north"`
	} `yaml:"region" json:"region"`
	Detail struct {
		Patches       int     `yaml:"patches" json:"patches"`
		Evaluations   int     `yaml:"evaluations" json:"evaluations"`
		Depth         int     `yaml:"depth" json:"depth"`
		Side          int     `yaml:"side" json:"side"`
		Points        int     `yaml:"points" json:"points"`
		CellPixels    float64 `yaml:"cell-pixels" json:"cell-pixels"`
		MinimumHeight float64 `yaml:"minimum-height-metres" json:"minimum-height-metres"`
		MaximumHeight float64 `yaml:"maximum-height-metres" json:"maximum-height-metres"`
		Normals       *struct {
			Longitude float64 `yaml:"longitude-deg" json:"longitude-deg"`
			Latitude  float64 `yaml:"latitude-deg" json:"latitude-deg"`
		} `yaml:"normal-step" json:"normal-step,omitempty"`
	} `yaml:"detail" json:"detail"`
	Camera struct {
		At   location `yaml:"at" json:"at"`
		Look location `yaml:"look-at" json:"look-at"`
		Near float64  `yaml:"near-metres" json:"near-metres"`
	} `yaml:"camera" json:"camera"`
	Film struct {
		Width  int     `yaml:"width" json:"width"`
		Height int     `yaml:"height" json:"height"`
		Rays   int     `yaml:"rays" json:"rays"`
		Wide   float64 `yaml:"wide-deg" json:"wide-deg"`
	} `yaml:"film" json:"film"`
	Sun struct {
		Azimuth  float64    `yaml:"azimuth-deg" json:"azimuth-deg"`
		Altitude float64    `yaml:"altitude-deg" json:"altitude-deg"`
		Strength [3]float64 `yaml:"strength-rgb" json:"strength-rgb"`
	} `yaml:"sun" json:"sun"`
	Ground [3]float64 `yaml:"ground-rgb" json:"ground-rgb"`
	Sky    [3]float64 `yaml:"sky-rgb" json:"sky-rgb"`
	Ocean  struct {
		Height float64    `yaml:"height-metres" json:"height-metres"`
		Colour [3]float64 `yaml:"colour-rgb" json:"colour-rgb"`
		Gloss  float64    `yaml:"gloss" json:"gloss"`
		Index  float64    `yaml:"index" json:"index"`
	} `yaml:"ocean" json:"ocean"`
	Exposure            float64      `yaml:"exposure" json:"exposure"`
	Landcover           []heightBand `yaml:"landcover" json:"landcover,omitempty"`
	LandcoverTransition float64      `yaml:"landcover-transition-metres" json:"landcover-transition-metres"`
	Site                *sitePlan    `yaml:"site" json:"site,omitempty"`
	Time                float64      `yaml:"time-seconds" json:"time-seconds"`
	RayReference        *float64     `yaml:"secondary-ray-reference-metres" json:"secondary-ray-reference-metres,omitempty"`
	raw                 []byte
	hash                string
}

// readPlan checks the schema and cross-field rules, resolves source paths and
// retains the exact input bytes for provenance.
func readPlan(path string) (*plan, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(absolute)
	if err != nil {
		return nil, err
	}
	schema, err := yamlfile.NewSchema("island-scene-v1.json", sceneSchema)
	if err != nil {
		return nil, err
	}
	node, err := schema.Read(path, raw)
	if err != nil {
		return nil, err
	}
	var plan plan
	if err := node.Decode(&plan); err != nil {
		return nil, err
	}
	for index := range plan.Sources {
		if !filepath.IsAbs(plan.Sources[index].File) {
			plan.Sources[index].File = filepath.Join(filepath.Dir(absolute), plan.Sources[index].File)
		}
	}
	if plan.Detail.Side < 3 || (plan.Detail.Side-1)&(plan.Detail.Side-2) != 0 ||
		plan.Region.West >= plan.Region.East || plan.Region.South >= plan.Region.North ||
		plan.Detail.MinimumHeight > plan.Detail.MaximumHeight {
		return nil, fmt.Errorf("island: invalid terrain region, lattice or height range")
	}
	if err := plan.checkComposition(); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	plan.raw, plan.hash = raw, hex.EncodeToString(digest[:])
	return &plan, nil
}

// terrain connects source heights to the declared sphere and detail budgets.
func (plan plan) terrain(sphere frame.Sphere, sources island.Heights) (*island.Terrain, error) {
	detail := island.Detail{
		Patches: plan.Detail.Patches, Evaluations: plan.Detail.Evaluations,
		Depth: plan.Detail.Depth, Side: plan.Detail.Side, Points: plan.Detail.Points,
		CellPixels: plan.Detail.CellPixels, MinimumHeight: plan.Detail.MinimumHeight, MaximumHeight: plan.Detail.MaximumHeight,
	}
	if plan.Detail.Normals != nil {
		detail.NormalLongitude = plan.Detail.Normals.Longitude
		detail.NormalLatitude = plan.Detail.Normals.Latitude
	}
	return island.New(sphere, sources, island.Region{West: plan.Region.West,
		South: plan.Region.South, East: plan.Region.East, North: plan.Region.North}, detail)
}

// rgb carries the scene's linear-light channels in red, green, blue order.
func rgb(value [3]float64) maths.Light { return maths.RGB(value[0], value[1], value[2]) }
