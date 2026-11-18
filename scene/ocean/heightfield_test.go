package ocean

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
)

func testConfig() HeightFieldConfig {
	return HeightFieldConfig{
		Sea:    oceantex.Sea{Wind: 8, Fetch: 1500, Toward: .3},
		Extent: 128, Samples: 9, Seed: 41, Seconds: 2.25,
	}
}

func TestHeightFieldUsesPeriodicEastNorthUpCascadesInEastUpSouth(t *testing.T) {
	config := testConfig()
	got, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	actual, n, extent, bottom, top, _ := got.Pack()
	if n != config.Samples || extent != config.Extent || !(bottom < top) {
		t.Fatalf("packed field n=%d extent=%g range=[%g,%g]", n, extent, bottom, top)
	}

	cascades, err := oceantex.WaveCascades(config.Sea, config.Seconds, config.Seed)
	if err != nil {
		t.Fatal(err)
	}
	step := config.Extent / float64(n-1)
	for _, point := range []struct{ row, column int }{{0, 0}, {0, 8}, {4, 4}, {8, 0}, {8, 8}} {
		east := -config.Extent/2 + float64(point.column)*step
		south := -config.Extent/2 + float64(point.row)*step
		var want float64
		for _, cascade := range cascades {
			want += exactCascadeNode(t, cascade.Field, east, south)
		}
		if actual[point.row*n+point.column] != want {
			t.Fatalf("sample at east=%g m, south=%g m is %g, want %g",
				east, south, actual[point.row*n+point.column], want)
		}
	}
	// Make sure the chosen asymmetric interior point can detect either axis
	// sign being reversed, rather than merely agreeing with a symmetric sample.
	east, south := -32.0, -48.0 // column 2, row 1 at the test's 16 m step
	var correct, eastReversed, northReversed float64
	for _, cascade := range cascades {
		correct += exactCascadeNode(t, cascade.Field, east, south)
		eastReversed += exactCascadeNode(t, cascade.Field, -east, south)
		northReversed += exactCascadeNode(t, cascade.Field, east, -south)
	}
	if correct == eastReversed || correct == northReversed {
		t.Fatal("oracle point does not distinguish east or north/south sign")
	}
	if actual[1*n+2] != correct {
		t.Fatalf("signed-axis sample=%g want=%g", actual[1*n+2], correct)
	}
	if got.Bounds() != (maths.Box{Min: maths.V(-64, bottom, -64), Max: maths.V(64, top, 64)}) {
		t.Fatalf("finite bounds: %+v", got.Bounds())
	}
	// The finite field has an explicit exact query for opt-in callers, but
	// ordinary scene tracing keeps its historical marching dispatch.
	centerHeight := actual[4*n+4]
	if distance, ok := got.IntersectExact(maths.Ray{At: maths.V(0, centerHeight+1, 0),
		Dir: maths.V(0, -1, 0)}, 0, 2); !ok || math.Abs(distance-1) > 1e-12 {
		t.Fatalf("exact centre hit distance=%g hit=%t, want 1 m", distance, ok)
	}
	if _, ok := got.IntersectExact(maths.Ray{At: maths.V(0, centerHeight+1, 0),
		Dir: maths.V(0, 1, 0)}, 0, 2); ok {
		t.Fatal("upward ray unexpectedly hit the field")
	}
	if _, ok := any(got).(shape.Spanner); ok {
		t.Fatal("height field unexpectedly claims a ray span")
	}
}

func TestSurfaceWakeIsBoundedAndMovesAlongAuthoredHeading(t *testing.T) {
	wake := SurfaceWake{OriginEastMetres: 0, OriginNorthMetres: 0,
		HeadingRadians: math.Pi / 2, SpeedMetresPerSecond: 2,
		AmplitudeMetres: .2, HalfWidthMetres: 5, LengthMetres: 40,
		WavelengthMetres: 12}
	// Heading east at 2 m/s moves the origin to x=2 at t=1. The aft
	// envelope exists west of that point, never ahead or outside its width.
	if got, _, _ := wake.sample(2, 0, 1); math.Abs(got) > 1e-12 {
		t.Fatalf("wake anchor displacement = %g, want 0", got)
	}
	inside, _, _ := wake.sample(-8, 0, 1)
	ahead, _, _ := wake.sample(4, 0, 1)
	farSide, _, _ := wake.sample(-8, 100, 1)
	if inside == 0 || ahead != 0 || math.Abs(farSide) > 1e-12 {
		t.Fatalf("wake samples inside=%g ahead=%g far-side=%g", inside, ahead, farSide)
	}
	// Rotating toward north changes the aft direction rather than leaving
	// the previous east-west trail in place.
	wake.HeadingRadians = 0
	wake.SpeedMetresPerSecond = 0
	northTrail, _, _ := wake.sample(0, -8, 1)
	eastTrail, _, _ := wake.sample(-8, 0, 1)
	if northTrail == 0 || math.Abs(eastTrail) > 1e-12 {
		t.Fatalf("heading changed to north but trail remained east-west: north=%g east=%g", northTrail, eastTrail)
	}
}

func TestSurfaceWakeRequiresSpectralSlopeOptIn(t *testing.T) {
	config := testConfig()
	config.SpectralShadingNormals = true
	config.SurfaceWake = &SurfaceWake{HeadingRadians: math.Pi,
		AmplitudeMetres: .1, HalfWidthMetres: 5, LengthMetres: 30,
		WavelengthMetres: 12}
	if _, err := NewHeightField(config); err == nil {
		t.Fatal("spectral wake silently omitted its surface slope")
	}
	config.SurfaceWake.AddToSpectralShading = true
	if _, err := NewHeightField(config); err != nil {
		t.Fatalf("explicit spectral wake slopes refused: %v", err)
	}
}

func TestSurfaceWakeIsOptInAndTracksItsAuthoredPoseOverTime(t *testing.T) {
	config := testConfig()
	config.Samples = 33
	withoutWake, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	config.SurfaceWake = nil
	stillOff, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	base, _, _, _, _, _ := withoutWake.Pack()
	off, _, _, _, _, _ := stillOff.Pack()
	if !reflect.DeepEqual(base, off) {
		t.Fatal("wake-off ocean changed when the option is absent")
	}
	wake := &SurfaceWake{HeadingRadians: math.Pi / 2,
		SpeedMetresPerSecond: 3, AmplitudeMetres: .2,
		HalfWidthMetres: 12, LengthMetres: 48, WavelengthMetres: 16}
	config.SurfaceWake = wake
	config.Seconds = 0
	eastbound, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	config.Seconds = 1
	movedEastbound, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	wake.HeadingRadians = 0
	northbound, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	first, _, _, _, _, _ := eastbound.Pack()
	second, _, _, _, _, _ := movedEastbound.Pack()
	third, _, _, _, _, _ := northbound.Pack()
	if reflect.DeepEqual(base, first) || reflect.DeepEqual(first, second) ||
		reflect.DeepEqual(second, third) {
		t.Fatal("wake did not alter the sea for its authored position/time/heading")
	}
}

// exactCascadeNode is an independent grid-node oracle. The chosen 128 m test
// extent and 16 m output step align every output point to the 256-sample grids
// of all three cascades, so expected heights can be read by integer indexing
// without using production's periodic interpolation helper.
func exactCascadeNode(t *testing.T, field *oceantex.Field, east, south float64) float64 {
	t.Helper()
	cell := field.Extent / float64(field.N)
	eastOffset := east / cell
	southOffset := south / cell
	if math.Abs(eastOffset-math.Round(eastOffset)) > 1e-9 ||
		math.Abs(southOffset-math.Round(southOffset)) > 1e-9 {
		t.Fatalf("test point east=%g south=%g is not on cascade grid %g", east, south, cell)
	}
	wrap := func(index int) int { return ((index % field.N) + field.N) % field.N }
	column := wrap(field.N/2 + int(math.Round(eastOffset)))
	row := wrap(field.N/2 - int(math.Round(southOffset)))
	return field.Height[row*field.N+column]
}

func TestPeriodicSampleWrapsNegativeAndPastPeriodCoordinates(t *testing.T) {
	field := &oceantex.Field{N: 2, Extent: 2, Height: []float64{0, 10, 20, 30}}
	if got, want := periodicSample(field, field.Height, -.5, -.5), 15.0; got != want {
		t.Fatalf("negative wrapped sample=%g want %g", got, want)
	}
	if got, want := periodicSample(field, field.Height, 2.5, .5),
		periodicSample(field, field.Height, .5, .5); got != want {
		t.Fatalf("sample past one period=%g want %g", got, want)
	}
}

func TestHeightFieldIsDeterministicAtAnExplicitTime(t *testing.T) {
	config := testConfig()
	first, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	a, _, _, _, _, _ := first.Pack()
	b, _, _, _, _, _ := second.Pack()
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same sea, seed and time produced different height samples")
	}
	config.Seconds += 1.5
	third, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	c, _, _, _, _, _ := third.Pack()
	if reflect.DeepEqual(a, c) {
		t.Fatal("changing explicit time left all wave heights unchanged")
	}
}

func TestHeightFieldNormalsAreFiniteAndFaceUp(t *testing.T) {
	field, err := NewHeightField(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	heights, n, extent, _, _, _ := field.Pack()
	step := extent / float64(n-1)
	for _, rowColumn := range [][2]int{{1, 1}, {4, 4}, {7, 6}} {
		row, column := rowColumn[0], rowColumn[1]
		point := maths.V(-extent/2+float64(column)*step,
			heights[row*n+column], -extent/2+float64(row)*step)
		normal := field.Normal(point)
		if !finite(normal.X) || !finite(normal.Y) || !finite(normal.Z) ||
			math.Abs(normal.Len()-1) > 1e-12 || normal.Y <= 0 {
			t.Fatalf("invalid upward unit normal at %v: %v", point, normal)
		}
	}
}

func TestSpectralShadingNormalsKeepGeometryAndUseEastNorthSlopes(t *testing.T) {
	config := testConfig()
	plain, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	config.SpectralShadingNormals = true
	spectral, err := NewHeightField(config)
	if err != nil {
		t.Fatal(err)
	}
	plainHeights, n, extent, bottom, top, _ := plain.Pack()
	gotHeights, gotN, gotExtent, gotBottom, gotTop, _ := spectral.Pack()
	if n != gotN || extent != gotExtent || bottom != gotBottom || top != gotTop ||
		!reflect.DeepEqual(plainHeights, gotHeights) {
		t.Fatal("spectral shading mode changed the sampled height field geometry")
	}
	if spectral.ApproximateShadingNormalMode() != SpectralSlopeNormals {
		t.Fatalf("normal mode = %q", spectral.ApproximateShadingNormalMode())
	}
	if plain.ApproximateShadingNormalMode() != "" {
		t.Fatalf("default normal mode = %q, want exact cell normals",
			plain.ApproximateShadingNormalMode())
	}
	for _, point := range []maths.Vec{maths.V(-32, .25, 16), maths.V(0, -1, -16)} {
		if got, want := spectral.Bounds(), plain.Bounds(); got != want {
			t.Fatalf("spectral bounds = %v, plain = %v", got, want)
		}
		if got, want := spectral.Distance(point), plain.Distance(point); got != want {
			t.Errorf("spectral distance at %v = %.17g, plain = %.17g", point, got, want)
		}
	}

	cascades, err := oceantex.WaveCascades(config.Sea, config.Seconds, config.Seed)
	if err != nil {
		t.Fatal(err)
	}
	row, column := 3, 5
	step := extent / float64(n-1)
	east := -extent/2 + float64(column)*step
	south := -extent/2 + float64(row)*step
	var slopeEast, slopeNorth float64
	for _, cascade := range cascades {
		eastSlope, northSlope := exactCascadeSlopesAt(t,
			cascade.Field, east, south)
		slopeEast += eastSlope
		slopeNorth += northSlope
	}
	point := maths.V(east, gotHeights[row*n+column], south)
	want := maths.V(-slopeEast, 1, slopeNorth).Unit()
	if got := spectral.Normal(point); got != want {
		t.Fatalf("spectral normal = %v, want east/north-up normal %v", got, want)
	}
	if got := spectral.Normal(point); got == plain.Normal(point) {
		t.Fatal("spectral shading normal unexpectedly equals the bilinear geometry normal")
	}
}

// exactCascadeSlopesAt reads the expected cascade slopes at aligned grid
// nodes, converting Slayer south-positive z to oceantex north-positive y.
func exactCascadeSlopesAt(t *testing.T, field *oceantex.Field,
	east, south float64) (slopeEast, slopeNorth float64) {
	t.Helper()
	cell := field.Extent / float64(field.N)
	eastOffset, southOffset := east/cell, south/cell
	if math.Abs(eastOffset-math.Round(eastOffset)) > 1e-9 ||
		math.Abs(southOffset-math.Round(southOffset)) > 1e-9 {
		t.Fatalf("test point east=%g south=%g is not on cascade grid %g", east, south, cell)
	}
	wrap := func(index int) int { return ((index % field.N) + field.N) % field.N }
	column := wrap(field.N/2 + int(math.Round(eastOffset)))
	row := wrap(field.N/2 - int(math.Round(southOffset)))
	return field.SlopeX[row*field.N+column],
		field.SlopeY[row*field.N+column]
}

func TestHeightFieldRejectsUnsupportedBoundsAndTimes(t *testing.T) {
	valid := testConfig()
	for name, change := range map[string]func(*HeightFieldConfig){
		"zero extent":      func(c *HeightFieldConfig) { c.Extent = 0 },
		"nonfinite extent": func(c *HeightFieldConfig) { c.Extent = math.Inf(1) },
		"sample spacing underflow": func(c *HeightFieldConfig) {
			c.Extent = math.SmallestNonzeroFloat64
			c.Samples = 3
		},
		"too many samples":   func(c *HeightFieldConfig) { c.Samples = MaxHeightFieldSamples + 1 },
		"too few samples":    func(c *HeightFieldConfig) { c.Samples = 1 },
		"negative time":      func(c *HeightFieldConfig) { c.Seconds = -1 },
		"nonfinite time":     func(c *HeightFieldConfig) { c.Seconds = math.NaN() },
		"extent past period": func(c *HeightFieldConfig) { c.Extent = 4097 },
	} {
		t.Run(name, func(t *testing.T) {
			config := valid
			change(&config)
			if _, err := NewHeightField(config); err == nil {
				t.Fatal("unsupported configuration was accepted")
			}
		})
	}
}
