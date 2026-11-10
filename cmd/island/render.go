package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/footprint"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/island"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
	"github.com/janearc/slayer/trace"
)

type record struct {
	Started             time.Time        `json:"started"`
	Elapsed             time.Duration    `json:"elapsed-nanoseconds"`
	SettingsSHA256      string           `json:"settings-sha256"`
	BinarySHA256        string           `json:"binary-sha256"`
	ImageSHA256         string           `json:"image-sha256"`
	Build               *debug.BuildInfo `json:"go-build-info,omitempty"`
	Sources             []sourcePlan     `json:"sources"`
	HeightModel         string           `json:"height-model"`
	RenderOrigin        maths.Vec        `json:"render-origin-metres"`
	Patches             []terrain.Patch  `json:"patches"`
	Samples             int              `json:"source-samples"`
	NormalStencils      int              `json:"normal-stencils"`
	NormalHeightQueries int              `json:"normal-height-queries"`
	Limited             bool             `json:"cell-target-limited"`
	Profile             gpu.Profile      `json:"gpu-profile"`
	RayReference        float64          `json:"secondary-ray-reference-metres"`
	Site                *siteRecord      `json:"site,omitempty"`
}

// render opens one checked scene, draws it and closes its retained sources.
func render(ctx context.Context, call pzm.Call) (failure error) {
	if len(call.Args()) != 0 {
		return fmt.Errorf("island render takes --scene and --out, without positional arguments")
	}
	name, err := required(call, "scene")
	if err != nil {
		return err
	}
	destination, err := required(call, "out")
	if err != nil {
		return err
	}
	plan, err := readPlan(name)
	if err != nil {
		return err
	}
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(), "%d x %d, %d samples/pixel, %d sources -> %s\n",
			plan.Film.Width, plan.Film.Height, plan.Film.Rays*plan.Film.Rays, len(plan.Sources), destination)
		return err
	}
	started := time.Now()
	session, err := openScene(ctx, plan)
	if err != nil {
		return err
	}
	defer func() { failure = errors.Join(failure, session.Close()) }()
	receipt, err := session.draw(ctx, plan, destination, started)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(call.Out(), "%s: %d patches, %d source points, limited=%t, %s\n",
		filepath.Join(destination, "island.png"), len(receipt.Patches), receipt.Samples, receipt.Limited, receipt.Elapsed)
	return err
}

// draw prepares one camera-relative view and writes its native image, exact
// settings and producer record to a new directory.
func (session *sceneSession) draw(ctx context.Context, plan *plan, destination string, started time.Time) (record, error) {
	sphere := session.sphere
	at, err := sphere.Position(plan.Camera.At.geographic())
	if err != nil {
		return record{}, err
	}
	look, err := sphere.Position(plan.Camera.Look.geographic())
	if err != nil {
		return record{}, err
	}
	basis, err := sphere.Tangent(at)
	if err != nil {
		return record{}, err
	}
	eye, err := maths.View(at, look, basis.Y)
	if err != nil {
		return record{}, err
	}
	view, err := footprint.New(*eye, measure.Degrees(plan.Film.Wide), plan.Film.Width,
		plan.Film.Height, 1, measure.Metres(plan.Camera.Near))
	if err != nil {
		return record{}, err
	}
	origin, err := frame.NewRelative(at)
	if err != nil {
		return record{}, err
	}
	prepared, err := session.ground.Prepare(ctx, view, origin)
	if err != nil {
		return record{}, err
	}
	world, err := plan.world(sphere, origin, prepared)
	if err != nil {
		return record{}, err
	}
	site, err := plan.compose(ctx, world, sphere, session.layers, view, origin, &session.prototypes)
	if err != nil {
		return record{}, err
	}
	localEye, err := origin.Placement(*eye)
	if err != nil {
		return record{}, err
	}
	lens, err := film.PinholeOf(plan.Film.Wide * maths.Degree)
	if err != nil {
		return record{}, err
	}
	if err := os.Mkdir(destination, 0o755); err != nil {
		return record{}, err
	}
	if err := os.WriteFile(filepath.Join(destination, "scene.yaml"), plan.raw, 0o644); err != nil {
		return record{}, err
	}
	picture, profile, err := gpu.SampledAccelerated(world, localEye, lens, time.Duration(plan.Time*float64(time.Second)),
		plan.Film.Width, plan.Film.Height, plan.Film.Rays)
	if err != nil {
		return record{}, err
	}
	if err := ctx.Err(); err != nil {
		return record{}, err
	}
	for index, pixel := range picture.Pix {
		for _, value := range []float32{pixel.R, pixel.G, pixel.B, pixel.Cover} {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return record{}, fmt.Errorf("island: non-finite rendered sample at pixel %d", index)
			}
		}
		if pixel.Cover < 0 || pixel.Cover > 1 || math.IsNaN(float64(pixel.Depth)) || pixel.Depth < 0 {
			return record{}, fmt.Errorf("island: invalid coverage or depth at pixel %d", index)
		}
	}
	imageHash, err := file.SequencePNGHash(filepath.Join(destination, "island.png"),
		colour.Develop(picture, plan.Exposure))
	if err != nil {
		return record{}, err
	}
	receipt := record{Started: started.UTC(), Elapsed: time.Since(started), SettingsSHA256: plan.hash,
		BinarySHA256: session.binaryHash, ImageSHA256: hex.EncodeToString(imageHash[:]), Build: session.build,
		Sources: plan.Sources, HeightModel: plan.Model, RenderOrigin: origin.Origin(),
		Patches: prepared.Patches, Samples: prepared.Samples, Limited: prepared.Limited, Profile: profile, Site: site}
	receipt.NormalStencils, receipt.NormalHeightQueries = prepared.NormalStencils, prepared.NormalHeightQueries
	receipt.RayReference = trace.Spread(world, time.Duration(plan.Time*float64(time.Second)))
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return record{}, err
	}
	if err := os.WriteFile(filepath.Join(destination, "render.json"), append(raw, '\n'), 0o644); err != nil {
		return record{}, err
	}
	return receipt, nil
}

// openSources verifies exact local bytes before opening any decoding state.
// All caches share one fixed retained-payload allowance rather than each
// receiving the default full cache. Metadata and temporary tiles are separate.
func (plan plan) openSources(ctx context.Context) (*island.Layers, []*terrain.Raster, error) {
	var layers []island.Heights
	var opened []*terrain.Raster
	for _, source := range plan.Sources {
		actual, err := fingerprint(ctx, source.File)
		if err != nil {
			return nil, opened, err
		}
		if actual != source.SHA256 {
			return nil, opened, fmt.Errorf("island: source hash changed: %s", source.File)
		}
		raster, err := terrain.OpenWithBudget(source.File, terrain.RasterBudget{
			CacheBytes: plan.RasterBudget.CacheBytes / int64(len(plan.Sources)), CacheTiles: plan.RasterBudget.CacheTiles,
			TileBytes: plan.RasterBudget.TileBytes, MetadataBytes: plan.RasterBudget.MetadataBytes,
		})
		if err != nil {
			return nil, opened, err
		}
		opened = append(opened, raster)
		heights, err := island.NewRasterHeights(raster, source.CRS, source.Datum, measure.Metres(source.Offset))
		if err != nil {
			return nil, opened, err
		}
		if index := plan.RasterBudget.Index; index != nil {
			heights, err = heights.Indexed(ctx, index.Block, index.Bytes/int64(len(plan.Sources)))
			if err != nil {
				return nil, opened, err
			}
		}
		layers = append(layers, heights)
	}
	layered, err := island.NewLayers(layers...)
	return layered, opened, err
}

// world places prepared ground, the smooth ocean proxy and authored lighting
// under the same render origin.
func (plan plan) world(sphere frame.Sphere, origin frame.Relative, ground island.Prepared) (*thing.World, error) {
	world := thing.NewWorld()
	if plan.RayReference != nil {
		reference, err := thing.NewRayOriginReference(measure.Metres(*plan.RayReference))
		if err != nil {
			return nil, err
		}
		world.RayOriginReference = reference
	}
	// Uniform authored background avoids a painted flat horizon at orbital
	// altitude. A spherical atmosphere and spectral wave water remain separate.
	background := rgb(plan.Sky)
	world.Sky = sky.Gradient{Horizon: background, Zenith: background, Ground: background}
	material := &thing.Material{Name: "authored island ground", Colour: rgb(plan.Ground)}
	if len(plan.Landcover) > 0 || (plan.Site != nil && plan.Site.Forest != nil) {
		material.Texture = &altitudePaint{frame: sphere, origin: origin.Origin(), bands: append([]heightBand(nil), plan.Landcover...), base: rgb(plan.Ground), transition: plan.LandcoverTransition}
	}
	for _, surface := range ground.Surfaces {
		world.Add(thing.Thing{Shape: surface, Material: material})
	}
	ocean, err := globe.New(plan.Radius + plan.Ocean.Height)
	if err != nil {
		return nil, err
	}
	centre, err := origin.Point(sphere.Centre())
	if err != nil {
		return nil, err
	}
	world.Add(thing.Thing{Shape: ocean, Material: &thing.Material{Name: "authored smooth ocean proxy",
		Colour: rgb(plan.Ocean.Colour), Gloss: plan.Ocean.Gloss, Index: plan.Ocean.Index},
		Motion: thing.Still(maths.NewSubspace(centre, maths.Orientation{}, 1))})
	azimuth, altitude := plan.Sun.Azimuth*maths.Degree, plan.Sun.Altitude*maths.Degree
	toward := maths.V(math.Sin(azimuth)*math.Cos(altitude), math.Sin(altitude), -math.Cos(azimuth)*math.Cos(altitude))
	lamp, err := emit.NewDistant(toward, rgb(plan.Sun.Strength))
	if err != nil {
		return nil, err
	}
	world.Lamp(lamp)
	return world, nil
}

// fingerprint checks cancellation between bounded reads; no source-sized
// byte slice is retained while verifying a large terrain file.
func fingerprint(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 32<<10)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		count, err := file.Read(buffer)
		if count > 0 {
			_, _ = hash.Write(buffer[:count])
		}
		if err == io.EOF {
			return hex.EncodeToString(hash.Sum(nil)), nil
		}
		if err != nil {
			return "", err
		}
	}
}
