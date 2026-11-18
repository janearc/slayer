//go:build draw

package trace

import (
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// TestClearAttenuationDistanceStudyDraw compares a glowing target at known
// camera-path distances with an air reference and a same-medium composition.
// Set SLAYER_CLEAR_ATTENUATION_STUDY_OUT to a new directory to keep all PNGs.
// The medium coefficients are synthetic RGB extinction in inverse metres.
func TestClearAttenuationDistanceStudyDraw(test *testing.T) {
	const (
		width, height = 512, 192
		maxDistance   = 4.5
		compositeSize = 0.05
	)
	extinction := maths.RGB(0.08, 0.25, 0.50) // picked, synthetic per metre
	clear, err := thing.NewClearMedium("synthetic-rgb-distance-study",
		extinction, 1)
	if err != nil {
		test.Fatal(err)
	}
	media := []InitialMedium{{Index: 1, ClearMedium: clear}}
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	eyeAt := maths.V(0, 0, 0)
	eye := maths.NewSubspace(eyeAt, maths.LookAt(eyeAt, maths.Up), 1)

	air, err := clearStudyPanel(eye, lens, maxDistance, nil,
		width, height)
	if err != nil {
		test.Fatal(err)
	}
	assertClearStudyTransmission(test, "air", air, extinction, 0)
	writeClearStudyImage(test, "clear-air-4.50m.png", air)

	for _, distance := range []float64{0.25, 0.5, 1, 2, maxDistance} {
		frame, err := clearStudyPanel(eye, lens, distance,
			media, width, height)
		if err != nil {
			test.Fatalf("distance %.2f m: %v", distance, err)
		}
		assertClearStudyTransmission(test, fmt.Sprintf("%.2f m", distance),
			frame, extinction, distance)
		writeClearStudyImage(test,
			fmt.Sprintf("synthetic-medium-%.2fm.png", distance), frame)
	}

	composition, err := clearStudyComposition(eye, lens, clear,
		compositeSize, width, height)
	if err != nil {
		test.Fatal(err)
	}
	visible := 0
	for _, pixel := range composition.Pix {
		if pixel.R > 0 || pixel.G > 0 || pixel.B > 0 {
			visible++
		}
	}
	if visible < 25 {
		test.Fatalf("composite shows only %d lit target pixels", visible)
	}
	writeClearStudyImage(test, "same-medium-five-targets.png", composition)
	test.Logf("five identical glowing targets: centre-path distances [0.75, 1.50, 2.50, 3.50, 4.50] m; extinction R/G/B=%.2f/%.2f/%.2f m^-1; all in the same explicitly supplied synthetic medium",
		extinction.R, extinction.G, extinction.B)
}

// clearStudyPanel creates the identical infinite glowing reference plane at
// one known axial distance. The central ray is axial, so expected path length
// is exactly the plane offset; no direct lamp path or refracting boundary is
// present.
func clearStudyPanel(eye maths.Subspace, lens film.Sensor, distance float64,
	media []InitialMedium, width, height int,
) (*frame.Frame, error) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "synthetic glowing reference target",
			Glow: maths.Grey(1)},
		Motion: thing.Still(maths.NewSubspace(maths.V(0, distance, 0),
			maths.Identity, 1))})
	return RenderWithInitialMedia(world, eye, lens, 0, width, height, media)
}

func assertClearStudyTransmission(test *testing.T, label string, made *frame.Frame,
	extinction maths.Light, pathMetres float64,
) {
	test.Helper()
	pixel := made.At(made.W/2, made.H/2)
	for _, channel := range []struct {
		name string
		got  float64
		ext  float64
	}{{"red", float64(pixel.R), extinction.R},
		{"green", float64(pixel.G), extinction.G},
		{"blue", float64(pixel.B), extinction.B}} {
		want := math.Exp(-channel.ext * pathMetres)
		if math.Abs(channel.got-want) > 2e-5 {
			test.Errorf("%s %s transmission %.8f at %.2f m, want Beer-Lambert %.8f",
				label, channel.name, channel.got, pathMetres, want)
		}
	}
	test.Logf("%s centre transmission R/G/B = %.4f / %.4f / %.4f",
		label, pixel.R, pixel.G, pixel.B)
}

func clearStudyComposition(eye maths.Subspace, lens film.Sensor,
	medium *thing.ClearMedium, radius float64, width, height int,
) (*frame.Frame, error) {
	world := thing.NewWorld()
	glow := &thing.Material{Name: "identical synthetic glowing composition target",
		Glow: maths.Grey(1)}
	distances := []float64{0.75, 1.5, 2.5, 3.5, 4.5}
	angles := []float64{-20, -10, 0, 10, 20}
	for index, distance := range distances {
		angle := angles[index] * maths.Degree
		center := maths.V(math.Sin(angle)*distance,
			math.Cos(angle)*distance, 0)
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: radius},
			Material: glow,
			Motion:   thing.Still(maths.NewSubspace(center, maths.Identity, 1))})
	}
	return RenderWithInitialMedia(world, eye, lens, 0, width, height,
		[]InitialMedium{{Index: 1, ClearMedium: medium}})
}

func writeClearStudyImage(test *testing.T, name string, frame *frame.Frame) {
	test.Helper()
	directory := os.Getenv("SLAYER_CLEAR_ATTENUATION_STUDY_OUT")
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	path := filepath.Join(directory, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		test.Fatal(err)
	}
	if err := png.Encode(file,
		colour.Tone{Exposure: 1, Knee: 0.8, Top: 2.2}.Develop(frame)); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		test.Fatal(err)
	}
	if err := file.Close(); err != nil {
		test.Fatal(err)
	}
	test.Logf("wrote %s", path)
}
