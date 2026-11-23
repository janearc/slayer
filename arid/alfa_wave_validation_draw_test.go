//go:build draw

package arid_test

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// TestAlfaWaveGeometryValidationDraw traces the catalogued Alfa over one
// bounded synthetic wave field. It is an opaque geometry/contact study, not a
// water-optics or coastal scene. Set SLAYER_ALFA_WAVE_STUDY_OUT to a new PNG
// path to retain its CPU image; without it the trace and assertions still run.
func TestAlfaWaveGeometryValidationDraw(t *testing.T) {
	const (
		width, height = 256, 144
		seconds       = 18.0
		seed          = 41
		extent        = 128.0
		grid          = 257
	)
	sea := oceantex.Sea{Wind: 12, Fetch: 100000, Toward: 0.2, Depth: 200}
	field, err := arid.NewHeightField(arid.HeightFieldConfig{
		Sea: sea, Extent: extent, Samples: grid, Seconds: seconds, Seed: seed,
	})
	if err != nil {
		t.Fatal(err)
	}

	kind, err := manufacture.KindNamed("alfa")
	if err != nil {
		t.Fatal(err)
	}
	boat, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(seed), "quick")
	if err != nil {
		t.Fatal(err)
	}
	afloat, floats := manufacture.Floating("alfa")
	waterline := afloat.Waterline.InMetres()
	if !floats || math.IsNaN(waterline) || math.IsInf(waterline, 0) ||
		waterline <= 0 {
		t.Fatalf("Alfa waterline unavailable: floats=%t line=%v", floats, afloat.Waterline)
	}
	place := maths.NewSubspace(maths.V(0, -waterline, 0), maths.Identity, 1)
	if placed := place.Out(maths.V(0, waterline, 0)); math.Abs(placed.Y) > 1e-12 {
		t.Fatalf("named surfaced waterline lands at y=%g, want mean surface y=0", placed.Y)
	}
	local := boat.Bounds()
	if local.Min.X < -extent/2 || local.Max.X > extent/2 ||
		local.Min.Z < -extent/2 || local.Max.Z > extent/2 {
		t.Fatalf("bounded study field does not cover the Alfa: boat=%+v sea=±%g",
			local, extent/2)
	}

	world := thing.NewWorld()
	world.Sky = sky.Gradient{
		Horizon: maths.RGB(0.48, 0.62, 0.69),
		Zenith:  maths.RGB(0.12, 0.24, 0.34),
		Ground:  maths.RGB(0.035, 0.09, 0.12),
	}
	seaLook := &thing.Material{Name: "synthetic opaque wave-study surface",
		Colour: maths.RGB(0.035, 0.19, 0.25)}
	world.Add(thing.Thing{Shape: field, Material: seaLook})

	boatThings, err := boat.Things(place)
	if err != nil {
		t.Fatal(err)
	}
	if len(boatThings) == 0 {
		t.Fatal("the catalogued Alfa made no scene geometry")
	}
	for index := range boatThings {
		boatThings[index].Material = opaqueDiagnosticLook(boatThings[index].Material)
		if !opaque(boatThings[index].Material) {
			t.Fatalf("boat component %q kept optical material behavior",
				boatThings[index].Material.Name)
		}
		world.Add(boatThings[index])
	}
	if !opaque(seaLook) {
		t.Fatal("wave-study surface is not opaque")
	}
	world.Lamp(thing.Sun(maths.V(-0.45, 0.62, 0.64),
		maths.RGB(1.7, 1.65, 1.55)))

	eyeAt := maths.V(70, 40, 91)
	eye := maths.NewSubspace(eyeAt,
		maths.LookAt(eyeAt, maths.V(0, 0, 0)), 1)
	frame, err := trace.Sampled(world, eye, film.Pinhole{Wide: 49 * maths.Degree},
		0, width, height, 1)
	if err != nil {
		t.Fatal(err)
	}
	lit := false
	for _, pixel := range frame.Pix {
		if pixel.R+pixel.G+pixel.B > 0 {
			lit = true
			break
		}
	}
	if !lit {
		t.Fatal("Alfa/wave geometry frame is entirely black")
	}

	t.Logf("geometry study: KindNamed(alfa), quick, seed=%d; named surfaced waterline %.3f m aligned to mean y=0; synthetic bounded wave field %+v at t=%.1fs, extent=%.1fm, %dx%d samples; %d opaque diagnostic boat components; %dx%d one-ray CPU trace",
		seed, waterline, sea, seconds, extent, grid, grid, len(boatThings), width, height)

	path := os.Getenv("SLAYER_ALFA_WAVE_STUDY_OUT")
	if path == "" {
		t.Log("traced CPU contact study; SLAYER_ALFA_WAVE_STUDY_OUT is unset")
		return
	}
	if filepath.Ext(path) != ".png" {
		t.Fatalf("SLAYER_ALFA_WAVE_STUDY_OUT must end in .png: %q", path)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file,
		colour.Tone{Exposure: 1.15, Knee: 0.8, Top: 2.2}.Develop(frame)); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote CPU geometry study %s", path)
}

// opaqueDiagnosticLook keeps geometry while replacing the catalogue's
// sourced optical finish with an explicitly synthetic, diffuse display color.
func opaqueDiagnosticLook(source *thing.Material) *thing.Material {
	made := *source
	name := made.Name
	color := maths.RGB(0.52, 0.57, 0.59)
	if name == "sail" {
		color = maths.RGB(0.10, 0.13, 0.15)
	}
	made.Name = "synthetic opaque Alfa diagnostic " + name
	made.Colour = color
	made.Texture, made.Glow, made.Shine, made.Radiant, made.Rough = nil,
		maths.Light{}, nil, nil, nil
	made.Gloss, made.Index, made.Rank, made.Through = 0, 0, 0, 0
	made.Metal, made.Film, made.Crinkle = nil, nil, nil
	made.Satin = 0
	return &made
}

func opaque(material *thing.Material) bool {
	return material != nil && material.Through == 0 && material.Gloss == 0 &&
		material.Metal == nil && material.Film == nil && material.Radiant == nil
}
