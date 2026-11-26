package session

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/render/carry"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/trace"
)

func cameraMediumSetText(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../set/testdata/built.yaml")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(raw),
		"      grass:\n"+
			"        colour: {srgb_linear: [0.05, 0.06, 0.04]}\n"+
			"        source: eye\n",
		"      grass:\n"+
			"        clear: {through: 1, index: 1.33, gloss: 1, rank: 2, "+
			"extinction-per-metre: [0.1, 0.2, 0.3]}\n"+
			"        source: synthetic validation coefficients; not measured water\n", 1)
	source = strings.Replace(source, "      title: \"the front\"\n",
		"      title: \"the front\"\n      initial-media: [grass]\n", 1)
	return source
}

func readCameraMediumSet(t *testing.T) set.Set {
	t.Helper()
	source := cameraMediumSetText(t)
	made, err := set.Read("camera-medium.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func fixedCameraTransportSet(t *testing.T, initial bool) set.Set {
	t.Helper()
	initialMedia := ""
	if initial {
		initialMedia = "      initial-media: [water]\n"
	}
	document := `scene:
  name: fixed-camera-water
  film: {width: 1, height: 1, rays: 1, wide: 1}
  cameras:
    underwater:
      title: underwater
      location: {at: [0, 0, 0]}
      orientation: {look_at: [0, -2, 0]}
` + initialMedia + `  world:
    unit: 2
    materials:
      water:
        clear: {through: 1, index: 1.33, gloss: 1, rank: 0, extinction-per-metre: [0.08, 0.15, 0.30]}
        source: "synthetic validation; not measured water"
      target:
        glow: {temperature: 2700, luminance: 1}
        source: "synthetic validation light"
    spheres:
      water-boundary:
        material: water
        middle: [0, 0, 0]
        radius: 5
        source: "synthetic validation boundary"
      target:
        material: target
        middle: [0, -7, 0]
        radius: 0.5
        source: "synthetic validation target"
`
	read, err := set.Read("fixed-camera-water.yaml", []byte(document))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

func TestStagingCarriesSelectedSetCameraMediaToCPU(t *testing.T) {
	read := readCameraMediumSet(t)
	stage := staging{given: asked{set: "camera-medium.yaml"}}
	if _, err := stage.eye(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.given.initialMedia) != 1 {
		t.Fatalf("staged camera media = %+v, want one medium",
			stage.given.initialMedia)
	}
	medium := stage.given.initialMedia[0].ClearMedium
	if stage.given.initialMedia[0].Index != 1.33 ||
		stage.given.initialMedia[0].Rank != 2 || medium == nil ||
		medium.Name() != "grass" ||
		medium.Extinction() != maths.RGB(0.1, 0.2, 0.3) ||
		medium.MetresPerUnit() != 1 {
		t.Fatalf("staged initial optical medium = %+v, want grass RGB extinction and 1 m/world unit", stage.given.initialMedia[0])
	}
}

func TestStagingCarriesParticipatingCameraMediumWithoutChangingExtinction(t *testing.T) {
	source := strings.Replace(cameraMediumSetText(t),
		"extinction-per-metre: [0.1, 0.2, 0.3]}",
		"extinction-per-metre: [0.1, 0.2, 0.3], scatter: {coefficient-per-metre: [0.02, 0.03, 0.04], phase-g: 0.7, steps: 12, max-distance: {value: 8, unit: m}}}", 1)
	read, err := set.Read("camera-participating-medium.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	stage := staging{given: asked{set: "camera-participating-medium.yaml"}}
	if _, err := stage.eye(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.given.initialMedia) != 1 {
		t.Fatalf("staged camera media = %+v, want one medium", stage.given.initialMedia)
	}
	medium := stage.given.initialMedia[0].ClearMedium
	if medium == nil || !medium.HasScattering() ||
		medium.Extinction() != maths.RGB(.1, .2, .3) ||
		medium.Scattering() != maths.RGB(.02, .03, .04) ||
		medium.Absorption() != maths.RGB(.08, .17, .26) ||
		medium.InscatterSteps() != 12 || medium.MaxInscatterDistanceMetres() != 8 {
		t.Fatalf("staged participating camera medium = %+v", medium)
	}
}

func TestOceanDiffuseSkyResolvesCameraStackToBoundaryMediumCopy(t *testing.T) {
	base, err := thing.NewClearMediumWithScattering("water",
		maths.Grey(.1), maths.Grey(.02), .25, 1, 8, 12)
	if err != nil {
		t.Fatal(err)
	}
	configured, err := base.WithDiffuseSkyInscatter(24)
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Material: &thing.Material{Through: 1,
		Index: 1.333, Rank: 2, ClearMedium: configured}}
	world := thing.NewWorld()
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary}
	given := asked{initialMedia: []trace.InitialMedium{{Index: 1.333,
		Rank: 2, ClearMedium: base}}}
	if err := given.resolveOceanInitialMedium(world); err != nil {
		t.Fatal(err)
	}
	if given.initialMedia[0].ClearMedium != configured ||
		given.initialMedia[0].ClearMedium.DiffuseSkyInscatterSamples() != 24 {
		t.Fatalf("camera medium stack did not adopt the configured ocean value: %+v",
			given.initialMedia[0])
	}
	wrong, err := thing.NewClearMediumWithScattering("water",
		maths.Grey(.11), maths.Grey(.02), .25, 1, 8, 12)
	if err != nil {
		t.Fatal(err)
	}
	bad := asked{initialMedia: []trace.InitialMedium{{Index: 1.333,
		Rank: 2, ClearMedium: wrong}}}
	if err := bad.resolveOceanInitialMedium(world); err == nil {
		t.Fatal("camera medium with mismatched coefficients was silently replaced")
	}
}

func TestNamedSetCameraKeepsItsOwnInitialMediaSelection(t *testing.T) {
	source := strings.Replace(cameraMediumSetText(t), "  world:\n",
		"    dry:\n"+
			"      title: dry\n"+
			"      location: {at: [10, -20, 2]}\n"+
			"      orientation: {look_at: [0, 0, 3]}\n"+
			"  world:\n", 1)
	read, err := set.Read("named-camera-medium.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	stage := staging{given: asked{set: "named-camera-medium.yaml", camera: "dry"}}
	if _, err := stage.eye(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.given.initialMedia) != 0 {
		t.Fatalf("selected air camera inherited another camera's medium: %+v",
			stage.given.initialMedia)
	}
}

func TestWorldOnlySetCarriesCameraMediaWithoutChangingAirCameraLens(t *testing.T) {
	worldText := strings.Replace(cameraMediumSetText(t), "  world:\n",
		"  world:\n    unit: 2\n", 1)
	withMedia, err := set.Read("world-only-medium.yaml", []byte(worldText))
	if err != nil {
		t.Fatal(err)
	}
	with := stagedWorld(withMedia, asked{set: "world-only-medium.yaml"})
	if with.err != nil {
		t.Fatal(with.err)
	}
	if len(with.given.initialMedia) != 1 {
		t.Fatalf("world-only camera media = %+v", with.given.initialMedia)
	}
	initial := with.given.initialMedia[0]
	if initial.Index != 1.33 || initial.Rank != 2 || initial.ClearMedium == nil ||
		initial.ClearMedium.Name() != "grass" ||
		initial.ClearMedium.Extinction() != maths.RGB(0.1, 0.2, 0.3) ||
		initial.ClearMedium.MetresPerUnit() != 2 {
		t.Fatalf("non-unit world-only initial medium = %+v, want 2 m/world unit", initial)
	}
	grass := false
	for _, each := range with.world.Things {
		if each.Material != nil && each.Material.Name == "grass" {
			grass = each.Material.ClearMedium != nil &&
				each.Material.ClearMedium.Name() == "grass" &&
				each.Material.ClearMedium.MetresPerUnit() == 2
		}
	}
	if !grass {
		t.Fatal("world-only set did not stage its grass interface with the 2 m/unit conversion")
	}

	noMediaText := strings.Replace(cameraMediumSetText(t),
		"      initial-media: [grass]\n", "", 1)
	noMediaText = strings.Replace(noMediaText, "      title: \"the front\"\n",
		"      title: \"the front\"\n      wide: 90\n", 1)
	noMedia, err := set.Read("world-only-air.yaml", []byte(noMediaText))
	if err != nil {
		t.Fatal(err)
	}
	air := stagedWorld(noMedia, asked{set: "world-only-air.yaml"})
	if air.err != nil {
		t.Fatal(air.err)
	}
	shot, err := noMedia.Shot(noMedia.Cameras()[0], nowhere)
	if err != nil {
		t.Fatal(err)
	}
	if air.given.eye == nil || *air.given.eye != shot.Eye() {
		t.Fatalf("air camera eye = %v, want old selected camera eye %v",
			air.given.eye, shot.Eye())
	}
	film, _ := noMedia.Film()
	if air.given.wide != film.Wide() || len(air.given.initialMedia) != 0 {
		t.Fatalf("air camera lens/media changed: wide %g, stack %+v; want film wide %g and air",
			air.given.wide, air.given.initialMedia, film.Wide())
	}
}

func TestTerrainBackedNamedMediumRequiresMetreWorldUnit(t *testing.T) {
	base, err := os.ReadFile("../set/testdata/kilauea-iki.yaml")
	if err != nil {
		t.Fatal(err)
	}
	withWater := func(unit string) set.Set {
		t.Helper()
		source := strings.Replace(string(base), "  set:\n",
			"  world:\n"+
				"    unit: "+unit+"\n"+
				"    materials:\n"+
				"      water:\n"+
				"        clear: {through: 1, index: 1.333, gloss: 1, extinction-per-metre: [0.1, 0.2, 0.3]}\n"+
				"        source: synthetic validation; not measured water\n"+
				"  set:\n", 1)
		source = strings.Replace(source,
			"      title: \"across the crater\"\n",
			"      title: \"across the crater\"\n"+
				"      initial-media: [water]\n", 1)
		read, err := set.Read("../set/testdata/kilauea-iki.yaml", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		return read
	}
	if err := refuseTerrainOpticalUnit(withWater("1")); err != nil {
		t.Fatalf("metre-based terrain rejected optical medium: %v", err)
	}
	if err := refuseTerrainOpticalUnit(withWater("2")); err == nil ||
		!strings.Contains(err.Error(), "require world.unit 1") {
		t.Fatalf("non-unit terrain optical scale error = %v", err)
	}
}

func TestStagingRefusesUnsupportedCameraInitialMedia(t *testing.T) {
	read := readCameraMediumSet(t)
	for _, test := range []struct {
		name  string
		given asked
		want  string
	}{
		{"moving job camera", asked{set: "camera-medium.yaml", cameraPath: true}, "fixed start medium cannot follow camera motion"},
		{"GPU job aperture", asked{set: "camera-medium.yaml", onGPU: true,
			optics: optics{aperture: 1}}, "require a pinhole camera"},
		{"GPU rig lens", asked{set: "camera-medium.yaml", onGPU: true,
			glass: &lens.Thin{}}, "require a pinhole camera"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stage := staging{given: test.given}
			if _, err := stage.eye(read); err == nil ||
				!strings.Contains(err.Error(), test.want) {
				t.Fatalf("staging error = %v, want substring %q", err,
					test.want)
			}
		})
	}
}

func TestGPUInitialMediaApertureRefusesBeforeDraw(t *testing.T) {
	media := []trace.InitialMedium{{Index: 1.33, Rank: 2}}
	for _, test := range []struct {
		name string
		lens asked
	}{
		{"job aperture", asked{optics: optics{aperture: 1}}},
		{"opened job lens", asked{optics: optics{opened: &lens.Thin{}}}},
		{"set rig", asked{glass: &lens.Thin{}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			given := test.lens
			given.onGPU, given.initialMedia = true, media
			_, _, err := drawMeasured(nil, given, 0)
			if err == nil || !strings.Contains(err.Error(),
				"GPU initial media currently require a pinhole camera") {
				t.Fatalf("GPU aperture error = %v", err)
			}
		})
	}
	if err := refuseGPUInitialMediaAperture(asked{initialMedia: media,
		optics: optics{aperture: 1}}); err != nil {
		t.Fatalf("CPU aperture behavior was changed: %v", err)
	}
}

func TestStagingCarriesFixedSetCameraMediaForGPU(t *testing.T) {
	read := readCameraMediumSet(t)
	stage := staging{given: asked{set: "camera-medium.yaml", onGPU: true}}
	eye, err := stage.eye(read)
	if err != nil {
		t.Fatalf("fixed camera media was refused before GPU dispatch: %v", err)
	}
	if len(stage.given.initialMedia) != 1 || stage.given.initialMedia[0].Index != 1.33 ||
		stage.given.initialMedia[0].Rank != 2 ||
		stage.given.initialMedia[0].ClearMedium == nil ||
		stage.given.initialMedia[0].ClearMedium.Name() != "grass" {
		t.Fatalf("GPU camera stack = %+v, want the authored grass medium", stage.given.initialMedia)
	}
	shot, err := read.Shot(read.Cameras()[0], func(string) (maths.Vec, bool) {
		return maths.Vec{}, false
	})
	if err != nil {
		t.Fatal(err)
	}
	if eye != shot.Eye() {
		t.Fatalf("staged eye %v differs from selected camera %v", eye, shot.Eye())
	}
}

func TestFixedCameraTransportFixtureStagesWithWorldScale(t *testing.T) {
	read := fixedCameraTransportSet(t, true)
	drawn := stagedWorld(read, asked{set: "fixed-camera-water.yaml",
		onGPU: true, width: 1, height: 1, samples: 1})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	if len(drawn.world.Things) != 2 {
		t.Fatalf("staged %d things, want water boundary and glow target",
			len(drawn.world.Things))
	}
	if len(drawn.given.initialMedia) != 1 {
		t.Fatalf("staged initial stack = %+v", drawn.given.initialMedia)
	}
	medium := drawn.given.initialMedia[0].ClearMedium
	if medium == nil || medium.Name() != "water" ||
		medium.Extinction() != maths.RGB(0.08, 0.15, 0.30) ||
		medium.MetresPerUnit() != 2 {
		t.Fatalf("staged camera medium = %+v, want water RGB extinction at 2 m/unit",
			medium)
	}
	if drawn.given.eye == nil || drawn.given.eye.Origin != (maths.Vec{}) ||
		drawn.given.eye.Forward() != maths.V(0, 0, 1) {
		t.Fatalf("staged camera eye = %v, want origin looking along +Z",
			drawn.given.eye)
	}
}

func TestGPUCameraMediaReachesDirectSunPreflight(t *testing.T) {
	world := thing.NewWorld()
	world.OceanSurface = &thing.OceanSurface{DirectSunMeanPlane: true}
	media := []trace.InitialMedium{{Index: 1.33}}
	_, _, err := drawMeasured(world, asked{onGPU: true,
		initialMedia: media, width: 1, height: 1, samples: 1}, 0)
	if err == nil || !strings.Contains(err.Error(),
		"CPU-only ocean mean-plane direct-sun transport") {
		t.Fatalf("GPU draw error = %v; want the GPU direct-sun preflight, before device opening", err)
	}
}

func TestJobCameraPathRefusalComesFromAuthoredJobBeforeDrawing(t *testing.T) {
	dir := t.TempDir()
	setPath := filepath.Join(dir, "camera-medium.yaml")
	source := cameraMediumSetText(t)
	if err := os.WriteFile(setPath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	jobPath := filepath.Join(dir, "moving.yaml")
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: camera-medium.yaml}
camera:
  unit: m
  look-at: {x: 0, y: 0, z: 0}
  moves:
    - from: {x: 0, y: 0, z: 6}
      to: {x: 0, y: 0, z: 2}
      speed: {value: 2, unit: m/s}
film: {width: 1, height: 1, rays: 1}
renderer: cpu
outputs: {picture: should-not-exist.png}
`
	if err := os.WriteFile(jobPath, []byte(jobText), 0o600); err != nil {
		t.Fatal(err)
	}
	wanted, err := job.Read(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	given, err := jobAsked(wanted)
	if err != nil {
		t.Fatal(err)
	}
	if !given.cameraPath {
		t.Fatal("job camera path was not carried into session options")
	}
	setSource := cameraMediumSetText(t)
	read, err := set.Read(setPath, []byte(setSource))
	if err != nil {
		t.Fatal(err)
	}
	stage := staging{given: given}
	if _, err := stage.eye(read); err == nil ||
		!strings.Contains(err.Error(), "fixed start medium cannot follow camera motion") {
		t.Fatalf("authored moving camera refusal = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "should-not-exist.png")); !os.IsNotExist(err) {
		t.Fatalf("rejected job created output: stat error %v", err)
	}
}

func TestGPUCameraMeanPlaneRefusalCreatesNoOutput(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("SLAYER_CACHE", cache)
	dir := t.TempDir()
	setPath := filepath.Join(dir, "camera-medium.yaml")
	text := strings.Replace(authoredOceanText("1"),
		"      title: front\n", "      title: front\n      initial-media: [water]\n", 1)
	text = strings.Replace(text, "      material: water\n",
		"      material: water\n      direct-sun: mean-plane\n", 1)
	if err := os.WriteFile(setPath, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "nested", "must-not-exist.png")
	var said, wrong strings.Builder
	code := run([]string{"-set", setPath, "-gpu", "-out", output,
		"-w", "1", "-h", "1", "-n", "1"}, &said, &wrong)
	if code == 0 || !strings.Contains(wrong.String(),
		"mean-plane") {
		t.Fatalf("run code %d, stdout %q, stderr %q", code,
			said.String(), wrong.String())
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("unsupported GPU draw created output: stat error %v", err)
	}
}

func TestInitialMediaProofRefusalCreatesNoProofOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-proof")
	var said, wrong strings.Builder
	code := proving(nil, asked{initialMedia: []trace.InitialMedium{{Index: 1.33}},
		prove: "missing.pfm", proof: path}, 0, nil, &said, &wrong)
	if code == 0 || !strings.Contains(wrong.String(),
		"pixel replay does not support an explicit start stack") {
		t.Fatalf("proof code %d, stdout %q, stderr %q", code,
			said.String(), wrong.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unsupported proof created output: stat error %v", err)
	}
}

func TestDrawMeasuredUsesTheSelectedCameraInitialStack(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Name: "water surface", Through: 1, Gloss: 1, Index: 1.33,
	}})
	from := maths.V(0, -1, 0)
	direction := maths.V(math.Sin(maths.Degree*30),
		math.Cos(maths.Degree*30), 0)
	entryX := direction.X / direction.Y
	sineOut := 1.33 * direction.X
	cosineOut := math.Sqrt(1 - sineOut*sineOut)
	onPath := maths.V(entryX+3*sineOut/cosineOut, 3, 0)
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.25},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: onPath})})
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, from.Add(direction))}
	given := asked{eye: &eye, width: 1, height: 1, samples: 1,
		initialMedia: []trace.InitialMedium{{Index: 1.33, Rank: 2}}}
	got, _, err := drawMeasured(w, given, 0)
	if err != nil {
		t.Fatal(err)
	}
	want, err := trace.SampledWithInitialMedia(w, eye, film.Pinhole{}, 0,
		1, 1, 1, given.initialMedia)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Pix, want.Pix) {
		t.Fatalf("hillside pixel %+v differs from CPU trace %+v",
			got.Pix, want.Pix)
	}
	if got.Pix[0].R == 1 {
		t.Fatal("the camera medium was not used at the flat boundary")
	}
	legacy, err := trace.Sampled(w, eye, film.Pinhole{}, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	air, _, err := drawMeasured(w, asked{eye: &eye, width: 1, height: 1,
		samples: 1}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(air.Pix, legacy.Pix) {
		t.Fatal("the no-medium hillside path differs from the original trace.Sampled path")
	}
}

func TestPhysicalLensRouteUsesTheSelectedCameraInitialStack(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Name: "water surface", Through: 1, Gloss: 1, Index: 1.33,
	}})
	from := maths.V(0, -1, 0)
	direction := maths.V(math.Sin(maths.Degree*30),
		math.Cos(maths.Degree*30), 0)
	entryX := direction.X / direction.Y
	sineOut := 1.33 * direction.X
	cosineOut := math.Sqrt(1 - sineOut*sineOut)
	onPath := maths.V(entryX+3*sineOut/cosineOut, 3, 0)
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.25},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: onPath})})
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, from.Add(direction))}
	sensorFrame, err := lens.FrameOf(36, 24)
	if err != nil {
		t.Fatal(err)
	}
	glass, err := lens.NewThin(sensorFrame, 50, 8, 10)
	if err != nil {
		t.Fatal(err)
	}
	given := asked{eye: &eye, glass: &glass, open: time.Second,
		glimpses: 1, width: 1, height: 1, samples: 1,
		initialMedia: []trace.InitialMedium{{Index: 1.33, Rank: 2}}}
	got, _, err := drawMeasured(w, given, 0)
	if err != nil {
		t.Fatal(err)
	}
	moments, err := instants(given, eye)
	if err != nil {
		t.Fatal(err)
	}
	glimpses := make([]*frame.Frame, 0, len(moments))
	for _, at := range moments {
		drawn, _, err := glimpsed(w, given, eye, at)
		if err != nil {
			t.Fatal(err)
		}
		glimpses = append(glimpses, drawn)
	}
	raw, err := glass.Expose(given.open, carry.NinjaFrames(glimpses))
	if err != nil {
		t.Fatal(err)
	}
	want := carry.Slayer(raw)
	if !slices.Equal(got.Pix, want.Pix) {
		t.Fatalf("physical-lens draw %+v differs from stacked CPU glimpses %+v",
			got.Pix, want.Pix)
	}
}

func TestCompositeRouteRefusesInitialMediaBeforeDeviceDraw(t *testing.T) {
	_, _, err := drawMeasured(nil, asked{onGPU: true, composite: &gpu.Frames{},
		initialMedia: []trace.InitialMedium{{Index: 1.33}}}, 0)
	if err == nil || !strings.Contains(err.Error(), "composite passes") {
		t.Fatalf("composite media error = %v", err)
	}
}

func TestOpenCompositeRefusesSetCameraMediaBeforeDeviceDraw(t *testing.T) {
	dir := t.TempDir()
	setPath := filepath.Join(dir, "camera-medium.yaml")
	source := cameraMediumSetText(t)
	if err := os.WriteFile(setPath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	jobPath := filepath.Join(dir, "composite.yaml")
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: camera-medium.yaml}
film: {width: 1, height: 1, rays: 1}
renderer: gpu
`
	if err := os.WriteFile(jobPath, []byte(jobText), 0o600); err != nil {
		t.Fatal(err)
	}
	wanted, err := job.Read(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	session, err := OpenComposite(context.Background(), wanted, Stamp{})
	if err == nil || session != nil || !strings.Contains(err.Error(),
		"explicit camera initial media") {
		if session != nil {
			_ = session.Close()
		}
		t.Fatalf("OpenComposite returned session %v and error %v; want early initial-media refusal", session, err)
	}
}
