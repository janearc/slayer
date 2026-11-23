package session

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/trace"
)

func authoredOceanText(unit string) string {
	text := `scene:
  name: authored-ocean
  film: {width: 1, height: 1, rays: 1}
  cameras:
    front:
      title: front
      location: {at: [0, 2, 0]}
      orientation: {look_at: [0, 0, 0]}
  world:
    unit: ` + unit + `
    ocean:
      material: water
      middle: [10, 20, 30]
      extent: {value: 8, unit: m}
      samples: 9
      sea:
        wind: {value: 0, unit: m/s}
        fetch: {value: 0, unit: m}
        depth: {value: 0, unit: m}
        toward: {value: 0, unit: rad}
      seed: 7
      time: {value: 0, unit: s}
      source: eye
    materials:
      water:
        clear: {through: 1, index: 1.33, gloss: 1, rank: 0}
        source: eye
`
	// The reader's set-world coordinates use [east, north, up], so put the
	// sea centre at the origin for the camera-boundary regression.
	text = strings.Replace(text, "middle: [10, 20, 30]", "middle: [0, 0, 0]", 1)
	return text
}

func authoredOceanSet(t *testing.T, unit string) set.Set {
	t.Helper()
	text := authoredOceanText(unit)
	read, err := set.Read("authored-ocean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

func TestAuthoredOceanStagesFiniteClearFieldWithUnitAndCentreTransform(t *testing.T) {
	read := authoredOceanSet(t, "10")
	drawn := stagedWorld(read, asked{set: "authored-ocean.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	if drawn.world.OceanSurface.WaveMotion != nil {
		t.Fatal("legacy snapshot ocean acquired scene-clock metadata")
	}
	if len(drawn.world.Things) != 1 {
		t.Fatalf("staged %d things, want just the ocean", len(drawn.world.Things))
	}
	ocean := drawn.world.Things[0]
	field, ok := ocean.Shape.(*shape.HeightField)
	if !ok {
		t.Fatalf("staged shape is %T, want *shape.HeightField", ocean.Shape)
	}
	heights, n, extent, bottom, top, _ := field.Pack()
	if n != 9 || extent != 8 || len(heights) != 81 || bottom != 0 || top != 0 {
		t.Fatalf("calm field n=%d extent=%g samples=%d range=[%g,%g]",
			n, extent, len(heights), bottom, top)
	}
	place := ocean.At(0)
	if place.Origin != maths.V(0, 0, 0) || place.Scale != .1 {
		t.Fatalf("field transform = %+v, want centred origin and scale .1", place)
	}
	if ocean.Material == nil || ocean.Material.Through != 1 || ocean.Material.Index != 1.33 {
		t.Fatalf("staged material = %+v", ocean.Material)
	}
}

func TestAuthoredOceanSmoothNormalsPreserveItsExactHeightFieldGeometry(t *testing.T) {
	text := strings.Replace(authoredOceanText("1"), "      source: eye\n",
		"      source: eye\n      shading-normal: "+
			arid.SmoothNormalBilinearVertexSlopes+"\n", 1)
	read, err := set.Read("authored-ocean-smooth-normal.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	got := stagedWorld(read, asked{set: "authored-ocean-smooth-normal.yaml"})
	if got.err != nil {
		t.Fatal(got.err)
	}
	plain := stagedWorld(authoredOceanSet(t, "1"), asked{set: "authored-ocean.yaml"})
	if plain.err != nil {
		t.Fatal(plain.err)
	}
	field, ok := got.world.Things[0].Shape.(*shape.HeightField)
	if !ok {
		t.Fatalf("staged shape is %T, want *shape.HeightField", got.world.Things[0].Shape)
	}
	plainField := plain.world.Things[0].Shape.(*shape.HeightField)
	if !got.world.OceanSurface.SmoothNormalApproximation {
		t.Fatal("staged ocean lost its explicit smooth-normal approximation marker")
	}
	if field.Bounds() != plainField.Bounds() {
		t.Fatalf("smooth-normal bounds = %v, plain bounds = %v", field.Bounds(), plainField.Bounds())
	}
	for _, point := range []maths.Vec{maths.V(0, 2, 0), maths.V(2.1, -1, -3)} {
		if got, want := field.Distance(point), plainField.Distance(point); got != want {
			t.Errorf("smooth-normal distance at %v = %.17g, plain = %.17g", point, got, want)
		}
	}
}

func TestAuthoredOceanSpectralNormalsPreserveItsExactHeightFieldGeometry(t *testing.T) {
	text := strings.Replace(authoredOceanText("1"), "      source: eye\n",
		"      source: eye\n      shading-normal: "+
			arid.SpectralSlopeNormals+"\n", 1)
	read, err := set.Read("authored-ocean-spectral-normal.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	got := stagedWorld(read, asked{set: "authored-ocean-spectral-normal.yaml"})
	if got.err != nil {
		t.Fatal(got.err)
	}
	plain := stagedWorld(authoredOceanSet(t, "1"), asked{set: "authored-ocean.yaml"})
	if plain.err != nil {
		t.Fatal(plain.err)
	}
	field, ok := got.world.Things[0].Shape.(*shape.HeightField)
	if !ok {
		t.Fatalf("staged shape is %T, want *shape.HeightField", got.world.Things[0].Shape)
	}
	plainField := plain.world.Things[0].Shape.(*shape.HeightField)
	if got.world.OceanSurface.ShadingNormalApproximation != arid.SpectralSlopeNormals ||
		field.ApproximateShadingNormalMode() != arid.SpectralSlopeNormals {
		t.Fatalf("spectral normal metadata/shape mode = %q/%q",
			got.world.OceanSurface.ShadingNormalApproximation,
			field.ApproximateShadingNormalMode())
	}
	plainHeights, n, extent, bottom, top, _ := plainField.Pack()
	spectralHeights, spectralN, spectralExtent, spectralBottom, spectralTop, _ := field.Pack()
	if !reflect.DeepEqual(plainHeights, spectralHeights) || n != spectralN ||
		extent != spectralExtent || bottom != spectralBottom || top != spectralTop ||
		field.Bounds() != plainField.Bounds() {
		t.Fatal("spectral shading mode changed staged height-field geometry")
	}
	for _, point := range []maths.Vec{maths.V(0, 2, 0), maths.V(2.1, -1, -3)} {
		if got, want := field.Distance(point), plainField.Distance(point); got != want {
			t.Errorf("spectral shading distance at %v = %.17g, plain = %.17g", point, got, want)
		}
	}
}

func TestAuthoredOceanStagesSceneClockRecipeOnItsBoundary(t *testing.T) {
	text := `"$schema": "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"` + "\n" +
		strings.Replace(authoredOceanText("1"), "      time: {value: 0, unit: s}\n",
			"      time: {value: 0, unit: s}\n      motion: scene-clock\n", 1)
	text = strings.Replace(text, "wind: {value: 0, unit: m/s}", "wind: {value: 8, unit: m/s}", 1)
	text = strings.Replace(text, "fetch: {value: 0, unit: m}", "fetch: {value: 1500, unit: m}", 1)
	read, err := set.Read("scene-clock-ocean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	drawn := stagedWorld(read, asked{set: "scene-clock-ocean.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	if len(drawn.world.Things) != 1 || drawn.world.OceanSurface == nil ||
		drawn.world.OceanSurface.Boundary != drawn.world.Things[0] {
		t.Fatal("scene-clock ocean did not retain exactly its authored boundary")
	}
	if drawn.world.OceanSurface.WaveMotion == nil || !drawn.world.OceanSurface.WaveMotion.Config().SceneClock {
		t.Fatalf("staged ocean motion metadata = %+v", drawn.world.OceanSurface.WaveMotion)
	}
}

func TestSceneClockOceanRetainsInfiniteMeanPlaneOption(t *testing.T) {
	text := `"$schema": "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"` + "\n" +
		strings.Replace(authoredOceanText("1"),
			"      time: {value: 0, unit: s}\n",
			"      time: {value: 0, unit: s}\n      motion: scene-clock\n", 1)
	text = strings.Replace(text, "      source: eye\n",
		"      source: eye\n"+
			"      flat-mean:\n"+
			"        mean-level: {value: 0, unit: m}\n"+
			"        collar-width: {value: 2, unit: m}\n"+
			"        extends-to-infinity: true\n", 1)
	read, err := set.Read("scene-clock-infinite-mean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	drawn := stagedWorld(read, asked{set: "scene-clock-infinite-mean.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	if !drawn.world.OceanSurface.InfiniteMeanPlane {
		t.Fatal("initial ocean staging lost infinite mean-plane metadata")
	}
	framed, err := sceneClockOceanWorldAt(drawn.world, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := framed.OceanSurface.Boundary.Shape.(*arid.InfiniteMeanContinuation); !ok {
		t.Fatalf("scene-clock boundary is %T, want infinite mean continuation",
			framed.OceanSurface.Boundary.Shape)
	}
	if !framed.OceanSurface.InfiniteMeanPlane {
		t.Fatal("scene-clock snapshot lost infinite mean-plane metadata")
	}
}

// sceneClockFrameShadingMode stages a scene-clock ocean with the named
// shading-normal mode, or none when empty, and returns the mode its field
// carries at one second of scene time.
func sceneClockFrameShadingMode(t *testing.T, mode string) string {
	t.Helper()
	text := `"$schema": "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"` + "\n" +
		strings.Replace(authoredOceanText("1"), "      time: {value: 0, unit: s}\n",
			"      time: {value: 0, unit: s}\n      motion: scene-clock\n", 1)
	text = strings.Replace(text, "wind: {value: 0, unit: m/s}", "wind: {value: 8, unit: m/s}", 1)
	text = strings.Replace(text, "fetch: {value: 0, unit: m}", "fetch: {value: 1500, unit: m}", 1)
	if mode != "" {
		text = strings.Replace(text, "      source: eye\n",
			"      source: eye\n      shading-normal: "+mode+"\n", 1)
	}
	read, err := set.Read("scene-clock-shaded-ocean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	drawn := stagedWorld(read, asked{set: "scene-clock-shaded-ocean.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	framed, err := sceneClockOceanWorldAt(drawn.world, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	field, ok := framed.OceanSurface.Boundary.Shape.(*shape.HeightField)
	if !ok {
		t.Fatalf("framed shape is %T, want *shape.HeightField",
			framed.OceanSurface.Boundary.Shape)
	}
	return field.ApproximateShadingNormalMode()
}

func TestSceneClockFramesKeepTheStagedShadingNormal(t *testing.T) {
	for _, mode := range []string{arid.SpectralSlopeNormals,
		arid.SmoothNormalBilinearVertexSlopes} {
		if got := sceneClockFrameShadingMode(t, mode); got != mode {
			t.Errorf("scene-clock frame shading mode = %q, want %q", got, mode)
		}
	}
}

func TestSceneClockFramesWithoutShadingNormalKeepTheCellNormal(t *testing.T) {
	if got := sceneClockFrameShadingMode(t, ""); got != "" {
		t.Errorf("unshaded scene-clock frame shading mode = %q, want none", got)
	}
}

func TestAuthoredOceanRefusesUnrepresentableWorldScale(t *testing.T) {
	read := authoredOceanSet(t, "1e-323")
	drawn := stagedWorld(read, asked{set: "authored-ocean.yaml"})
	if drawn.err == nil || !strings.Contains(drawn.err.Error(), "invalid metre scale") {
		t.Fatalf("staging error = %v, want invalid-scale refusal", drawn.err)
	}
}

func TestAuthoredFlatMeanContinuationCrossesOnceInCoreAndCollar(t *testing.T) {
	text := strings.Replace(authoredOceanText("1"), "      source: eye\n",
		"      source: eye\n"+
			"      flat-mean:\n"+
			"        mean-level: {value: 0, unit: m}\n"+
			"        collar-width: {value: 2, unit: m}\n", 1)
	read, err := set.Read("authored-ocean-flat-mean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	drawn := stagedWorld(read, asked{set: "authored-ocean-flat-mean.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	if _, ok := drawn.world.Things[0].Shape.(*arid.FlatMeanContinuation); !ok {
		t.Fatalf("staged shape is %T, want *arid.FlatMeanContinuation", drawn.world.Things[0].Shape)
	}
	world := drawn.world
	planeAt := thing.Still(maths.Subspace{Origin: maths.V(0, -2, 0)})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Glow: maths.Grey(1)},
		Motion: planeAt})
	want := 1 - maths.Fresnel(1, 1.33)
	for _, x := range []float64{0, 5} { // core and the collar beyond its 4 m edge
		eye := maths.Subspace{Origin: maths.V(x, 1, 0),
			Orientation: maths.LookAt(maths.V(x, 1, 0), maths.V(x, -2, 0))}
		image, err := trace.Render(world, eye, film.Pinhole{}, 0, 1, 1)
		if err != nil {
			t.Fatalf("x=%g: %v", x, err)
		}
		if got := float64(image.At(0, 0).R); math.Abs(got-want) > 2e-5 {
			t.Errorf("x=%g transmitted %g, want one interface transmission %g", x, got, want)
		}
	}
	// Beyond the finite outer edge, the exact same ray reaches the target
	// without crossing an interface.
	x := 7.0
	eye := maths.Subspace{Origin: maths.V(x, 1, 0),
		Orientation: maths.LookAt(maths.V(x, 1, 0), maths.V(x, -2, 0))}
	image, err := trace.Render(world, eye, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := float64(image.At(0, 0).R); got != 1 {
		t.Fatalf("outside finite collar transmitted %g, want untouched target", got)
	}
}

func TestAuthoredOceanWorldHalfUsesTerrainRebase(t *testing.T) {
	text := strings.Replace(authoredOceanText("1"), "  film:",
		"  terrain:\n"+
			"    from: fixture-dem\n"+
			"    file: fixture.tif\n"+
			"    west: 100\n"+
			"    north: 200\n"+
			"    across: 8\n"+
			"    down: 8\n"+
			"    step: 1\n"+
			"  film:", 1)
	read, err := set.Read("terrain-ocean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := read.World()
	world := thing.NewWorld()
	making := &staging{world: world, west: 100, north: 200, lowest: 30}
	if err := making.halved(read); err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 1 {
		t.Fatalf("staged %d things, want one ocean", len(world.Things))
	}
	// The authored [east,north,up] centre [0,0,0] is (0,0,0) in the
	// world frame, then terrain placement moves it by (-west,-lowest,north).
	if got, want := world.Things[0].At(0).Origin, maths.V(-100, -30, 200); got != want {
		t.Fatalf("terrain-rebased ocean centre = %v, want %v", got, want)
	}
	if said.Unit() != 1 || world.Things[0].At(0).Scale != 1 {
		t.Fatalf("terrain world unit=%g, field scale=%g, want metre scale 1", said.Unit(), world.Things[0].At(0).Scale)
	}
}

func TestTerrainOceanRefusesWorldUnitWithoutCommonScale(t *testing.T) {
	text := strings.Replace(authoredOceanText("10"), "  film:",
		"  terrain:\n"+
			"    from: fixture-dem\n"+
			"    file: fixture.tif\n"+
			"    west: 100\n"+
			"    north: 200\n"+
			"    across: 8\n"+
			"    down: 8\n"+
			"    step: 1\n"+
			"  film:", 1)
	read, err := set.Read("terrain-ocean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	making := &staging{given: asked{set: "authored-ocean.yaml"}, world: world}
	if err := making.halved(read); err == nil || !strings.Contains(err.Error(), "requires world unit 1") {
		t.Fatalf("terrain ocean error = %v, want explicit unit refusal", err)
	}
	if len(world.Things) != 0 {
		t.Fatalf("unit refusal staged %d things before failing", len(world.Things))
	}
}

func TestAuthoredOceanClearSheetHandlesAirWaterAndFiniteEdgeRays(t *testing.T) {
	read := authoredOceanSet(t, "1")
	drawn := stagedWorld(read, asked{set: "authored-ocean.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	world := drawn.world
	// A glowing plane on the far side makes transmission measurable.
	glow := &thing.Material{Glow: maths.Grey(1)}
	planeAt := thing.Still(maths.Subspace{Origin: maths.V(0, -2, 0)})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: glow,
		Motion: planeAt})
	airEye := maths.Subspace{Origin: maths.V(0, 1, 0),
		Orientation: maths.LookAt(maths.V(0, 1, 0), maths.V(0, -2, 0))}
	air, err := trace.Render(world, airEye, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantTransmission := 1 - maths.Fresnel(1, 1.33)
	if delta := float64(air.At(0, 0).R) - wantTransmission; delta < -2e-5 || delta > 2e-5 {
		t.Fatalf("air-to-water ray returned %g, want Fresnel transmission %g", air.At(0, 0).R, wantTransmission)
	}

	// Reverse the setup: the explicitly declared water camera exits through
	// the same finite sheet toward a glowing plane in air.
	above := thing.NewWorld()
	for _, each := range world.Things[:1] {
		above.Add(*each)
	}
	roofAt := thing.Still(maths.Subspace{Origin: maths.V(0, 2, 0)})
	above.Add(thing.Thing{Shape: shape.Plane{}, Material: glow,
		Motion: roofAt})
	underEye := maths.Subspace{Origin: maths.V(0, -1, 0),
		Orientation: maths.LookAt(maths.V(0, -1, 0), maths.V(0, 2, 0))}
	under, err := trace.RenderWithInitialMedia(above, underEye,
		film.Pinhole{}, 0, 1, 1,
		[]trace.InitialMedium{{Index: 1.33, Rank: 0}})
	if err != nil {
		t.Fatal(err)
	}
	if delta := float64(under.At(0, 0).R) - wantTransmission; delta < -2e-5 || delta > 2e-5 {
		t.Fatalf("water-to-air ray returned %g, want Fresnel transmission %g", under.At(0, 0).R, wantTransmission)
	}

	// A downward ray outside the finite field reaches the lower light without
	// crossing a clear boundary.
	edgeEye := maths.Subspace{Origin: maths.V(5, 1, 0),
		Orientation: maths.LookAt(maths.V(5, 1, 0), maths.V(5, -2, 0))}
	edge, err := trace.Render(world, edgeEye, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := float64(edge.At(0, 0).R); got != 1 {
		t.Fatalf("finite-edge miss transmitted %g light, want full light", got)
	}
}
