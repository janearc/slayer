package hillside

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	seaocean "github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestSessionInspectSetSnapshotAndLifecycle(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	setPath, err := filepath.Abs("../sets/widow.yaml")
	if err != nil {
		t.Fatal(err)
	}
	session, err := Open(context.Background(), jobFrom(t, "scene: {set: "+setPath+"}\n"), Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if got, want := session.Times(), []time.Duration{0}; !reflect.DeepEqual(got, want) {
		t.Fatalf("still sample times %v, want %v", got, want)
	}
	first, err := session.Inspect(0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := session.Inspect(0)
	if err != nil {
		t.Fatal(err)
	}
	world := session.source.(stillFramer).world
	want := thing.Storage(valuesOf(world.Things))
	if !reflect.DeepEqual(first.Geometry, want) || !reflect.DeepEqual(first, second) {
		t.Fatalf("snapshot differs or changed on repeat: first=%+v second=%+v want=%+v", first, second, want)
	}
	if first.Volumes != len(world.Volumes) || first.Fields != len(world.Fields) || first.Lamps != len(world.Lamps) {
		t.Fatalf("top-level resources differ: %+v", first)
	}
	if _, err := session.Inspect(1); err == nil {
		t.Fatal("inspection accepted a frame outside the still")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Inspect(0); err == nil {
		t.Fatal("closed session allowed inspection")
	}
}

func TestSessionInspectNamedFilmUsesExactTimesWithoutDrawing(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	wanted := jobFrom(t, "scene: {named: orbital, with: {far: true}}\nrenderer: cpu\n"+
		"film: {width: 24, height: 14, rays: 1}\n"+
		"fps: 2\nstart: {value: 0.5, unit: s}\n"+
		"duration: {value: 1, unit: s}\n")
	session, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	indices := []int{0, 1}
	times := session.Times()
	if len(times) != len(indices) || times[0] != 500*time.Millisecond || times[1] != time.Second {
		t.Fatalf("job sample times = %v", times)
	}
	source := session.source.(*namedFramer)
	if source.frames != nil {
		t.Fatal("opening the report allocated a GPU drawer")
	}
	if !source.built.Timed() {
		t.Fatal("test scene no longer exercises time-dependent construction")
	}
	for _, index := range indices {
		got, err := session.Inspect(index)
		if err != nil {
			t.Fatalf("inspect frame %d: %v", index, err)
		}
		want := thing.Storage(valuesOf(source.built.World().Things))
		if got.At != times[index] || !reflect.DeepEqual(got.Geometry, want) {
			t.Fatalf("frame %d snapshot %+v differs from world storage %+v", index, got, want)
		}
		if source.frames != nil {
			t.Fatal("inspection opened a GPU drawer")
		}
	}
}

type inspectTestSource struct{ world *thing.World }

func (source inspectTestSource) inspect(time.Duration) (*thing.World, error) {
	return source.world, nil
}

func (inspectTestSource) light(time.Duration, asked) (drawnLight, error) {
	return drawnLight{}, nil
}

func TestSessionInspectRefusesNilWorldEntry(t *testing.T) {
	session := &Session{source: inspectTestSource{world: &thing.World{
		Things: []*thing.Thing{nil},
	}}, times: []time.Duration{0}}
	if _, err := session.Inspect(0); err == nil {
		t.Fatal("inspection silently omitted a nil thing")
	}
}

func valuesOf(things []*thing.Thing) []thing.Thing {
	values := make([]thing.Thing, 0, len(things))
	for _, placed := range things {
		if placed != nil {
			values = append(values, *placed)
		}
	}
	return values
}

func TestSceneClockOceanSnapshotsFollowAbsoluteFrameTimesAndInspect(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	directory := t.TempDir()
	setPath := filepath.Join(directory, "ocean.yaml")
	text := `"$schema": "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"` + "\n" +
		strings.Replace(authoredOceanText("1"), "      time: {value: 0, unit: s}\n",
			"      time: {value: 0, unit: s}\n      motion: scene-clock\n"+
				"      surface-wake:\n"+
				"        origin: [0, 0, 0]\n"+
				"        heading: {value: 0, unit: rad}\n"+
				"        speed: {value: 2, unit: m/s}\n"+
				"        amplitude: {value: 0.12, unit: m}\n"+
				"        half-width: {value: 2, unit: m}\n"+
				"        length: {value: 6, unit: m}\n"+
				"        wavelength: {value: 3, unit: m}\n"+
				"        add-to-spectral-shading: false\n", 1)
	text = strings.Replace(text, "wind: {value: 0, unit: m/s}", "wind: {value: 8, unit: m/s}", 1)
	text = strings.Replace(text, "fetch: {value: 0, unit: m}", "fetch: {value: 1500, unit: m}", 1)
	if err := os.WriteFile(setPath, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	sources, err := os.ReadFile("../sources.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "sources.yaml"), sources, 0o600); err != nil {
		t.Fatal(err)
	}
	wanted := jobFrom(t, "scene: {set: "+setPath+"}\nrenderer: cpu\n"+
		"film: {width: 1, height: 1, rays: 1}\nfps: 1\n"+
		"duration: {value: 2, unit: s}\n")
	session, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	source := session.source.(stillFramer)
	if len(session.Times()) != 2 || session.Times()[0] != 0 || session.Times()[1] != time.Second {
		t.Fatalf("scene times = %v", session.Times())
	}
	first, err := source.worldAt(session.Times()[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := source.worldAt(session.Times()[1])
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := source.worldAt(session.Times()[0])
	if err != nil {
		t.Fatal(err)
	}
	firstField := first.OceanSurface.Boundary.Shape.(*shape.HeightField)
	secondField := second.OceanSurface.Boundary.Shape.(*shape.HeightField)
	repeatField := repeated.OceanSurface.Boundary.Shape.(*shape.HeightField)
	one, n, _, _, _, _ := firstField.Pack()
	two, _, _, _, _, _ := secondField.Pack()
	again, _, _, _, _, _ := repeatField.Pack()
	if !reflect.DeepEqual(one, again) {
		t.Fatal("same scene time produced different wave samples")
	}
	if reflect.DeepEqual(one, two) {
		t.Fatal("distinct frame times produced the same wave samples")
	}
	if n != 9 || len(first.Things) != 1 || first.OceanSurface.Boundary != first.Things[0] ||
		second.OceanSurface.Boundary != second.Things[0] {
		t.Fatal("frame snapshot does not have one shared ocean boundary")
	}
	wake := first.OceanSurface.WaveMotion.Config().SurfaceWake
	if wake == nil || wake.SpeedMetresPerSecond != 2 ||
		wake.HeadingRadians != 0 || wake.AmplitudeMetres != .12 {
		t.Fatalf("scene-clock wake recipe was not preserved: %+v", wake)
	}
	if firstField == secondField || first.OceanSurface.Boundary == second.OceanSurface.Boundary {
		t.Fatal("distinct times reused mutable ocean snapshot objects")
	}
	if firstField.Normal(maths.V(0, 0, 0)).Y <= 0 {
		t.Fatal("regenerated ocean normal does not face up")
	}
	for index, at := range session.Times() {
		inspection, err := session.Inspect(index)
		if err != nil {
			t.Fatal(err)
		}
		inspected, err := source.inspect(at)
		if err != nil {
			t.Fatal(err)
		}
		field := inspected.OceanSurface.Boundary.Shape.(*shape.HeightField)
		got, _, _, _, _, _ := field.Pack()
		want := one
		if index == 1 {
			want = two
		}
		if inspection.At != at || !reflect.DeepEqual(got, want) {
			t.Fatalf("inspection at %s did not use the frame's scene clock", at)
		}
	}
}

func TestSceneClockOceanFilmRefusesGPUShutterAndCameraPath(t *testing.T) {
	directory := t.TempDir()
	setPath := filepath.Join(directory, "ocean.yaml")
	text := `"$schema": "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"` + "\n" +
		strings.Replace(authoredOceanText("1"), "      time: {value: 0, unit: s}\n",
			"      time: {value: 0, unit: s}\n      motion: scene-clock\n", 1)
	if err := os.WriteFile(setPath, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	base := "scene: {set: " + setPath + "}\nfilm: {width: 1, height: 1, rays: 1}\n" +
		"fps: 1\nduration: {value: 1, unit: s}\n"
	for _, each := range []struct {
		name, extra, want string
	}{
		{"gpu", "renderer: gpu\n", "CPU"},
		{"shutter", "shutter: {angle: {value: 180, unit: deg}, glimpses: 2}\n", "shutter"},
		{"camera path", "camera:\n  unit: m\n  look-at: {x: 0, y: 0, z: 0}\n" +
			"  moves:\n    - from: {x: 0, y: 2, z: 4}\n" +
			"      to: {x: 1, y: 2, z: 4}\n      duration: {value: 1, unit: s}\n", "camera"},
		{"camera zoom", "camera:\n  name: view\n  zoom:\n    from: {value: 20, unit: deg}\n" +
			"    to: {value: 30, unit: deg}\n    duration: {value: 1, unit: s}\n", "camera"},
	} {
		t.Run(each.name, func(t *testing.T) {
			_, err := Open(context.Background(), jobFrom(t, base+each.extra), Stamp{})
			if err == nil || !strings.Contains(err.Error(), each.want) {
				t.Fatalf("Open error = %v, want explicit %s refusal", err, each.want)
			}
		})
	}
}

func TestRetainedCPUFilmUsesSceneClockOceanSnapshots(t *testing.T) {
	config := thing.OceanWaveConfig{WindMetresPerSecond: 8, FetchMetres: 1500,
		DepthMetres: 30, ExtentMetres: 8, Samples: 9, Seed: 7, SceneClock: true}
	field, err := seaocean.NewHeightField(seaocean.HeightFieldConfig{
		Sea: oceantex.Sea{Wind: config.WindMetresPerSecond, Fetch: config.FetchMetres,
			Depth: config.DepthMetres}, Extent: config.ExtentMetres,
		Samples: config.Samples, Seed: config.Seed, Seconds: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Shape: field}
	world := &thing.World{Things: []*thing.Thing{boundary}, OceanSurface: &thing.OceanSurface{
		Boundary: boundary, WaveMotion: thing.NewOceanWaveMotion(config),
	}}
	live := &liveScene{base: *world}
	timed := timedFramer{given: asked{live: live}}
	first, err := timed.inspect(0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := timed.inspect(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := timed.inspect(0)
	if err != nil {
		t.Fatal(err)
	}
	firstSamples, _, _, _, _, _ := first.OceanSurface.Boundary.Shape.(*shape.HeightField).Pack()
	secondSamples, _, _, _, _, _ := second.OceanSurface.Boundary.Shape.(*shape.HeightField).Pack()
	repeatSamples, _, _, _, _, _ := repeat.OceanSurface.Boundary.Shape.(*shape.HeightField).Pack()
	if reflect.DeepEqual(firstSamples, secondSamples) {
		t.Fatal("retained CPU ocean stayed at one authored time")
	}
	if !reflect.DeepEqual(firstSamples, repeatSamples) {
		t.Fatal("same absolute time produced a different retained CPU ocean")
	}
	if first.OceanSurface.Boundary != first.Things[0] || second.OceanSurface.Boundary != second.Things[0] ||
		first.OceanSurface.Boundary == second.OceanSurface.Boundary {
		t.Fatal("per-frame ocean metadata did not point to its unique boundary")
	}
	if live.base.OceanSurface.Boundary != boundary || live.base.Things[0] != boundary || live.base.Things[0].Shape != field {
		t.Fatal("frame inspection mutated the retained ocean source")
	}
}
