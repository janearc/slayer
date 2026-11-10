package oceantex

import (
	"math"
	"math/cmplx"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

func near(t *testing.T, name string, got, want, within float64) {
	t.Helper()
	if math.IsNaN(got) || math.IsInf(got, 0) || math.IsNaN(want) || math.IsInf(want, 0) || math.Abs(got-want) > within {
		t.Errorf("%s is %v, not %v (within %v)", name, got, want, within)
	}
}

var north = Sea{Wind: 10, Fetch: 100e3, Toward: 0.4}

// The fast transform agrees with the slow one, term by term.
func TestFFTIsTheSlowTransform(t *testing.T) {
	const n = 16
	source := rand.New(rand.NewPCG(1, 2))
	in := make([]complex128, n)
	for i := range in {
		in[i] = complex(source.NormFloat64(), source.NormFloat64())
	}
	fast := append([]complex128(nil), in...)
	fft(fast, false)
	for k := 0; k < n; k++ {
		var slow complex128
		for j := 0; j < n; j++ {
			slow += in[j] * cmplx.Rect(1, -2*math.Pi*float64(j*k)/n)
		}
		if cmplx.Abs(slow-fast[k]) > 1e-9 {
			t.Fatalf("term %d: fast %v, slow %v", k, fast[k], slow)
		}
	}
	fft(fast, true)
	for i := range in {
		if cmplx.Abs(fast[i]/n-in[i]) > 1e-9 {
			t.Fatalf("there and back changed term %d", i)
		}
	}
}

// The spreading shares out exactly all of the energy, at any frequency.
func TestSpreadSumsToOne(t *testing.T) {
	for _, ratio := range []float64{0.3, 0.7, 0.95, 1, 1.3, 1.6, 2, 5, 20} {
		const steps = 20000
		var sum float64
		for i := 0; i < steps; i++ {
			theta := -math.Pi + (float64(i)+0.5)*2*math.Pi/steps
			sum += Spread(ratio, 1, theta) * 2 * math.Pi / steps
		}
		near(t, "the spread's sum", sum, 1, 1e-6)
	}
}

// The spectrum peaks where Peak says it does.
func TestSpectrumPeaksAtItsPeak(t *testing.T) {
	peak := north.Peak()
	best, at := 0.0, 0.0
	for omega := 0.2 * peak; omega < 4*peak; omega += peak / 2000 {
		if s := north.Frequency(omega); s > best {
			best, at = s, omega
		}
	}
	near(t, "the peak", at/peak, 1, 0.01)
}

// The energy on the wavenumber grid is the energy in the frequency
// spectrum over the same band: the change of variables loses nothing.
func TestWavenumberKeepsTheEnergy(t *testing.T) {
	const n, extent = 512, 4000.0
	cell := 2 * math.Pi / extent
	var grid float64
	for iy := 0; iy < n; iy++ {
		for ix := 0; ix < n; ix++ {
			grid += north.Wavenumber(float64(wave(ix, n))*cell,
				float64(wave(iy, n))*cell) * cell * cell
		}
	}
	// the grid holds a square of wave vectors; a disc that fits inside
	// it and one that holds it bracket the answer
	band := func(top float64) float64 {
		lo, hi := math.Sqrt(Gravity*cell/2), math.Sqrt(Gravity*top)
		const steps = 200000
		var sum float64
		for i := 0; i < steps; i++ {
			omega := lo + (float64(i)+0.5)*(hi-lo)/steps
			sum += north.Frequency(omega) * (hi - lo) / steps
		}
		return sum
	}
	inside, outside := band(cell*n/2), band(cell*n/2*math.Sqrt2)
	if grid < 0.97*inside || grid > 1.03*outside {
		t.Errorf("grid energy %v, outside %v to %v", grid, inside, outside)
	}
}

// A drawn sea has the height variance and slope variance its spectrum
// promises, and its slopes are the slopes of its heights.
func TestPatchIsItsSpectrum(t *testing.T) {
	const n, extent = 256, 1000.0
	patch, err := NewPatch(north, n, extent, 7)
	if err != nil {
		t.Fatal(err)
	}
	cell := 2 * math.Pi / extent
	var promised float64
	for iy := 0; iy < n; iy++ {
		for ix := 0; ix < n; ix++ {
			if wave(ix, n) == -n/2 || wave(iy, n) == -n/2 {
				continue
			}
			promised += north.Wavenumber(float64(wave(ix, n))*cell,
				float64(wave(iy, n))*cell) * cell * cell
		}
	}
	along, across := patch.Resolved()
	var height, slope float64
	const seas = 12
	for seed := uint64(0); seed < seas; seed++ {
		p, _ := NewPatch(north, n, extent, seed)
		f := p.At(3.5)
		for i := range f.Height {
			height += f.Height[i] * f.Height[i]
			slope += f.SlopeX[i]*f.SlopeX[i] + f.SlopeY[i]*f.SlopeY[i]
		}
	}
	height /= seas * n * n
	slope /= seas * n * n
	near(t, "height variance", height/promised, 1, 0.1)
	near(t, "slope variance", slope/(along+across), 1, 0.05)

	f := patch.At(0)
	dx := extent / n
	var worst float64
	for iy := 0; iy < n; iy++ {
		for ix := 1; ix < n-1; ix++ {
			central := (f.Height[iy*n+ix+1] - f.Height[iy*n+ix-1]) / (2 * dx)
			worst = math.Max(worst, math.Abs(central-f.SlopeX[iy*n+ix]))
		}
	}
	// a central difference misses the shortest waves; it should still
	// agree to within the slope's own spread
	if worst > 3*math.Sqrt(along+across) {
		t.Errorf("slopes are not the heights' slopes: off by %v", worst)
	}
}

// Waves move downwind: the surface a moment later matches the surface
// now, shifted along the wind, better than shifted against it.
func TestWavesTravelDownwind(t *testing.T) {
	sea := Sea{Wind: 10, Fetch: 100e3, Toward: 0}
	const n, extent = 128, 1000.0
	p, _ := NewPatch(sea, n, extent, 3)
	now, later := p.At(0), p.At(1.5)
	match := func(shift int) float64 {
		var sum float64
		for iy := 0; iy < n; iy++ {
			for ix := 0; ix < n; ix++ {
				from := (ix - shift + n) % n
				sum += later.Height[iy*n+ix] * now.Height[iy*n+from]
			}
		}
		return sum
	}
	if match(2) <= match(-2) {
		t.Errorf("waves are travelling upwind")
	}
}

func TestFresnel(t *testing.T) {
	const index = 1.34
	head := (index - 1) / (index + 1)
	near(t, "head on", Fresnel(1, index), head*head, 1e-12)
	near(t, "grazing", Fresnel(0, index), 1, 1e-12)
	// at Brewster's angle one polarisation is not reflected at all, so
	// the pair reflects half of what the other one does
	brewster := math.Atan(index)
	other := math.Cos(2 * brewster)
	near(t, "at Brewster's angle", Fresnel(math.Cos(brewster), index),
		other*other/2, 1e-12)
}

func TestIndexIsSeawater(t *testing.T) {
	sea := Index(35, 20, 550)
	if sea < 1.335 || sea > 1.345 {
		t.Errorf("seawater's index is %v", sea)
	}
	if Index(35, 20, 400) <= Index(35, 20, 700) {
		t.Errorf("blue should bend more than red")
	}
	if Index(35, 20, 550) <= Index(0, 20, 550) {
		t.Errorf("salt should raise the index")
	}
}

// The published numbers come back out as published.
func TestPublishedValues(t *testing.T) {
	along, across := Roughness(10)
	near(t, "slope variance along the wind", along, 0.0316, 1e-12)
	near(t, "slope variance across it", across, 0.0222, 1e-12)
	// [CM54] fitted the total separately, as 0.003 + 0.00512 U; the two
	// parts were fitted on their own and sum to within a percent of it
	near(t, "total slope variance", along+across, 0.003+0.00512*10, 1e-3)
	near(t, "whitecaps at 10 m/s", Whitecaps(10), 0.00987, 1e-5)
	near(t, "absorption at 440", Absorption(440), 0.00635, 1e-12)
	near(t, "absorption at 700", Absorption(700), 0.624, 1e-12)
	near(t, "absorption at 445", Absorption(445), (0.00635+0.00922)/2, 1e-12)
	near(t, "scattering at 500", Scattering(500), 0.00288, 1e-12)
}

// The slope density is a density.
func TestDensitySumsToOne(t *testing.T) {
	a, c := Roughness(8)
	const reach, steps = 2.0, 800
	var sum float64
	for i := 0; i < steps; i++ {
		for j := 0; j < steps; j++ {
			x := -reach + (float64(i)+0.5)*2*reach/steps
			y := -reach + (float64(j)+0.5)*2*reach/steps
			sum += density(x, y, a, c) * (2 * reach / steps) * (2 * reach / steps)
		}
	}
	near(t, "the density's sum", sum, 1, 1e-6)
}

// Clear deep water is blue, and far darker than what lights it. With
// nothing in it but water and salt it is a violet blue: the reflectance
// climbs all the way down to 400 nm, so a screen needs a trace more red
// than green to show it.
func TestWaterIsBlue(t *testing.T) {
	if !(Reflectance(450, 0) > Reflectance(550, 0) &&
		Reflectance(550, 0) > Reflectance(650, 0)) {
		t.Errorf("reflectance does not fall toward red")
	}
	r, g, b := srgb.Light(Water(swatch.D65(), 0))
	if !(b > 10*g && b > 10*r && g > 0 && r > 0) {
		t.Errorf("water is r %v g %v b %v", r, g, b)
	}
	_, y, _ := Water(swatch.D65(), 0).XYZ()
	if y <= 0 || y > 0.05 {
		t.Errorf("water's luminance is %v", y)
	}
}

// libtheme's own white comes back through lit.
func TestLampIsLibthemesWhite(t *testing.T) {
	x, y, z := Lamp(swatch.D65()).XYZ()
	wx, wy, wz := swatch.WhitePoint().XYZ()
	near(t, "X", x, wx, 1e-9)
	near(t, "Y", y, wy, 1e-9)
	near(t, "Z", z, wz, 1e-9)
}

// The glint returns about as much light as a flat sea would reflect: the
// rough surface moves the sun's reflection around, it does not make any.
// Summed over every direction of view, with the sun high, the reflected
// share is Fresnel's.
func TestGlintConservesEnergy(t *testing.T) {
	const index = 1.34
	varA, varC := Roughness(5)
	zenith := 30 * math.Pi / 180
	sun := Vec{math.Sin(zenith), 0, math.Cos(zenith)}
	var sent float64
	const steps = 600
	for i := 0; i < steps; i++ {
		for j := 0; j < steps; j++ {
			// equal-area steps over the upper hemisphere
			cosine := (float64(i) + 0.5) / steps
			turn := (float64(j) + 0.5) * 2 * math.Pi / steps
			sine := math.Sqrt(1 - cosine*cosine)
			eye := Vec{sine * math.Cos(turn), sine * math.Sin(turn), cosine}
			half := Vec{sun.X + eye.X, sun.Y + eye.Y, sun.Z + eye.Z}.unit()
			chance := density(-half.X/half.Z, -half.Y/half.Z, varA, varC)
			radiance := Fresnel(sun.dot(half), index) * chance /
				(4 * eye.Z * math.Pow(half.Z, 4))
			sent += radiance * eye.Z * (2 * math.Pi / (steps * steps))
		}
	}
	received := sun.Z
	share := sent / received
	flat := Fresnel(sun.Z, index)
	if share < 0.9*flat || share > 1.25*flat {
		t.Errorf("the glint returns %v of the light; a flat sea %v",
			share, flat)
	}
}

// From orbit the grid carries almost none of the slope; near the surface
// it carries a large part. That is the whole level-of-detail scheme.
func TestGridCarriesLessFromHigher(t *testing.T) {
	sea := Sea{Wind: 10, Fetch: 100e3}
	total := func() float64 { a, c := Roughness(sea.Wind); return a + c }()
	carried := func(extent float64) float64 {
		p, _ := NewPatch(sea, 256, extent, 1)
		a, c := p.Resolved()
		return (a + c) / total
	}
	low, high := carried(60), carried(300e3)
	if !(low > 0.2 && low < 1) {
		t.Errorf("a 60 m tile carries %v of the slope variance", low)
	}
	if high > 0.01 {
		t.Errorf("a 300 km tile carries %v of the slope variance", high)
	}
}

func TestRenderIsRepeatableAndFinite(t *testing.T) {
	scene := Scene{Sea: north, Salinity: 35, Celsius: 15,
		Sun: Vec{0.3, 0.2, 1}}
	tile := Tile{Size: 64, Extent: 100, Altitude: 50, Seed: 5}
	a, _, err := Render(scene, tile)
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := Render(scene, tile)
	for i := range a.Light {
		if a.Light[i] != b.Light[i] {
			t.Fatalf("the same seed gave a different sea at texel %d", i)
		}
		for _, v := range a.Light[i] {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("texel %d is %v", i, a.Light[i])
			}
		}
	}
	if _, _, err := Render(scene, Tile{Size: 100, Extent: 1, Altitude: 1}); err == nil {
		t.Errorf("a size that is not a power of two was accepted")
	}
}

// Every number the code asks for is in sources.yaml, every number there
// is asked for, and every one says where and when it was read.
func TestEveryNumberIsSourcedAndUsed(t *testing.T) {
	asked := map[string]bool{}
	call := regexp.MustCompile(`nums?\("([^"]+)"\)`)
	files, _ := filepath.Glob("*.go")
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		text, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range call.FindAllStringSubmatch(string(text), -1) {
			asked[match[1]] = true
			if _, found := sources[match[1]]; !found {
				t.Errorf("%s asks for %q, which sources.yaml lacks",
					file, match[1])
			}
		}
	}
	for _, source := range Sources() {
		if !asked[source.Name] {
			t.Errorf("sources.yaml has %q, which no code uses", source.Name)
		}
		if _, err := time.Parse("2006-01-02", source.Fetched); err != nil {
			t.Errorf("%s: fetched %q is not a date", source.Name, source.Fetched)
		}
		switch source.Kind {
		case "sourced", "definition":
			if !strings.HasPrefix(source.URL, "https://") ||
				(source.Read != "fetch" && source.Read != "search" &&
					source.Read != "clone") {
				t.Errorf("%s is %s but has no url read by fetch, search or clone",
					source.Name, source.Kind)
			}
		case "glue":
		default:
			t.Errorf("%s: no kind called %q", source.Name, source.Kind)
		}
	}
}

func TestSourcesRefuseBadFiles(t *testing.T) {
	for _, bad := range []string{
		"a:\n  value: 1\n",     // nothing about where from
		"  value: 1\n",         // a field with no name
		"a:\n  value: one\n",   // not a number
		"a:\n  colour: blue\n", // no such field
		"a:\n  values: 1, 2\n", // no brackets
		"a\n",                  // no colon
	} {
		if _, err := parseSources(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

// More chlorophyll makes the sea greener and none leaves it as it was.
func TestChlorophyllGreensTheSea(t *testing.T) {
	green := func(chl float64) float64 {
		return Reflectance(550, chl) / Reflectance(450, chl)
	}
	last := green(0)
	for _, chl := range []float64{0.03, 0.1, 0.3, 1, 3, 10} {
		if now := green(chl); now <= last {
			t.Errorf("at %v mg/m^3 the sea is no greener: %v then %v",
				chl, last, now)
		} else {
			last = now
		}
	}
	if green(10) < 1 {
		t.Errorf("at 10 mg/m^3 the sea is still bluer than green")
	}
	if green(0.03) > 1 {
		t.Errorf("at 0.03 mg/m^3 the sea is already greener than blue")
	}
}

func TestGridAndColourAt(t *testing.T) {
	rows := "lat,lon,month,chl\n35,-125,6,0.2\n35,-125,1,0.6\n-45,5,6,0.1\n"
	grid, err := ReadGrid(strings.NewReader(rows), 10)
	if err != nil {
		t.Fatal(err)
	}
	if got, found := grid.At(37.5, -122.5, time.June); !found || got != 0.2 {
		t.Errorf("June off California is %v, %v", got, found)
	}
	if got, _ := grid.At(37.5, 237.5, time.January); got != 0.6 {
		t.Errorf("longitude did not wrap: %v", got)
	}
	if _, found := grid.At(0, 0, time.June); found {
		t.Errorf("found a cell that was never given")
	}
	june := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	colour, err := ColourAt(37.5, -122.5, june, grid)
	if err != nil {
		t.Fatal(err)
	}
	want := Water(swatch.D65(), 0.2)
	if colour != want {
		t.Errorf("colour is %v, not %v", colour, want)
	}
	if _, err := ColourAt(0, 0, june, grid); err == nil {
		t.Errorf("a place with no chlorophyll got a colour")
	}
	if _, err := ReadGrid(strings.NewReader("1,2,13,0.1\n"), 10); err == nil {
		t.Errorf("month 13 was accepted")
	}
}

// The sun is where the calendar says: overhead on the tropic at the June
// solstice, on the equator at the equinoxes, due south at noon.
func TestSunFollowsTheCalendar(t *testing.T) {
	declination := func(when time.Time) float64 {
		toward, _ := Sun(90, 0, when) // at the pole, height is declination
		return math.Asin(toward.Z) / degrees
	}
	at := func(month time.Month, day int) time.Time {
		return time.Date(2026, month, day, 12, 0, 0, 0, time.UTC)
	}
	near(t, "June solstice", declination(at(time.June, 21)), 23.44, 0.05)
	near(t, "December solstice", declination(at(time.December, 21)), -23.44, 0.05)
	near(t, "March equinox", declination(at(time.March, 20)), 0, 0.3)
	near(t, "September equinox", declination(at(time.September, 23)), 0, 0.3)

	noon, _ := Sun(50, 0, at(time.March, 20))
	if math.Abs(noon.X) > 0.05 || noon.Y >= 0 {
		t.Errorf("at Greenwich noon the sun is at %v, not due south", noon)
	}
	near(t, "noon height at 50 north", math.Asin(noon.Z)/degrees, 40, 0.5)
	night, _ := Sun(50, 0, time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))
	if night.Z > 0 {
		t.Errorf("the sun is up at midnight")
	}
	_, january := Sun(0, 0, at(time.January, 3))
	_, july := Sun(0, 0, at(time.July, 4))
	if !(january < 0.984 && july > 1.016) {
		t.Errorf("the sun is %v au off in January and %v in July",
			january, july)
	}
}

// A low sun crosses more air, and comes out dimmer and redder.
func TestLowSunIsDimAndRed(t *testing.T) {
	near(t, "air overhead", AirMass(1), 1, 0.001)
	near(t, "air at 60 degrees", AirMass(0.5), 2, 0.01)
	if horizon := AirMass(0); horizon < 30 || horizon > 45 {
		t.Errorf("air mass at the horizon is %v", horizon)
	}
	if haze := Haze(550); haze < 0.09 || haze > 0.105 {
		t.Errorf("clean air at 550 nm has optical depth %v", haze)
	}
	above, high := Sunlight(1, 1)
	_, low := Sunlight(0.05, 1)
	_, highY, _ := lit(high, above, func(int) float64 { return 1 }).XYZ()
	_, lowY, _ := lit(low, above, func(int) float64 { return 1 }).XYZ()
	if !(lowY < highY && highY < 1 && highY > 0.8) {
		t.Errorf("luminance through the air: high sun %v, low sun %v",
			highY, lowY)
	}
	red := func(lamp map[int]float64) float64 { return lamp[650] / lamp[450] }
	if red(low) <= red(high) {
		t.Errorf("the low sun is no redder")
	}
}

func TestSeaStates(t *testing.T) {
	// a height exactly on a bound belongs to the lower code: the table's
	// own example is that four metres is coded 5
	for height, want := range map[float64]SeaState{
		0: 0, 0.05: 1, 0.1: 1, 0.3: 2, 1: 3, 2: 4, 2.5: 4, 4: 5, 4.1: 6,
		8: 7, 9: 7, 14: 8, 14.1: 9, 30: 9,
	} {
		if got := StateOf(height); got != want {
			t.Errorf("%v m is state %d, not %d", height, got, want)
		}
	}
	if SeaState(5).String() != "rough" {
		t.Errorf("state 5 is %q", SeaState(5))
	}
	for code := SeaState(1); code <= 7; code++ {
		sea, err := code.Raise(500e3, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := StateOf(sea.Height()); got != code {
			t.Errorf("a sea raised to state %d is %v m, state %d, in a "+
				"%v m/s wind", code, sea.Height(), got, sea.Wind)
		}
	}
	if _, err := SeaState(8).Raise(1000, 0, 0); err == nil {
		t.Errorf("a very high sea was raised over one kilometre of fetch")
	}
	if _, err := SeaState(10).Raise(1e6, 0, 0); err == nil {
		t.Errorf("state 10 was accepted")
	}
	// the Icelandic Met Office's worked example: 9.5 is 24.5 m/s
	near(t, "Beaufort 9.5", Beaufort(9.5), 24.5, 0.05)
}

// Shallow water slows long waves to sqrt(g h) and takes energy from them.
func TestShallowWater(t *testing.T) {
	deep := Sea{Wind: 15, Fetch: 200e3}
	shoal := deep
	shoal.Depth = 5
	if !(shoal.Height() < deep.Height()) {
		t.Errorf("5 m of water did not lower the waves: %v against %v",
			shoal.Height(), deep.Height())
	}
	k := 0.001 // a wave six kilometres long
	omega, _ := shoal.turning(k)
	near(t, "long-wave speed", omega/k, math.Sqrt(Gravity*5), 0.01)
	k = 10 // a wave too short to feel the bottom
	omega, _ = shoal.turning(k)
	near(t, "short-wave frequency", omega, math.Sqrt(Gravity*k), 1e-9)
	// the group speed is the slope of the dispersion curve
	for _, k := range []float64{0.01, 0.1, 0.5} {
		lo, _ := shoal.turning(k * 0.999)
		hi, _ := shoal.turning(k * 1.001)
		_, slope := shoal.turning(k)
		near(t, "group speed", slope, (hi-lo)/(k*0.002), 1e-4*slope)
	}
}

func TestTide(t *testing.T) {
	tide := Tide{{Name: "M2", Amplitude: 1.5, Phase: 90}}
	level, err := tide.Level(0)
	if err != nil {
		t.Fatal(err)
	}
	near(t, "a quarter cycle before high water", level, 0, 1e-12)
	period := 360 / tideSpeeds["M2"]
	if period < 12.41 || period > 12.43 {
		t.Errorf("the moon's tide comes every %v hours", period)
	}
	high, _ := tide.Level(period / 4)
	near(t, "high water", high, 1.5, 1e-9)
	if _, err := (Tide{{Name: "X9"}}).Level(0); err == nil {
		t.Errorf("an unknown constituent was accepted")
	}
	sea, err := (Sea{Wind: 8, Fetch: 50e3}).Flood(3, tide, period/4)
	if err != nil || math.Abs(sea.Depth-4.5) > 1e-9 {
		t.Errorf("flood depth is %v, %v", sea.Depth, err)
	}
	if _, err := (Sea{}).Flood(1, tide, period*3/4); err == nil {
		t.Errorf("a sea with the tide out below its bed was accepted")
	}
}

// Choppy waves gather water at the crests: where the sea is high the
// push squeezes it, and where it is low the push stretches it.
func TestChopSharpensCrests(t *testing.T) {
	p, _ := NewPatch(north, 128, 500, 4)
	f := p.At(1)
	var both float64
	for i, h := range f.Height {
		both += h * (f.StretchXX[i] + f.StretchYY[i])
	}
	if both >= 0 {
		t.Errorf("the push stretches the crests")
	}
}

// Looking level out to sea: sky above the horizon, sea below it, and the
// same picture every time.
func TestViewFromTheShore(t *testing.T) {
	scene := Scene{Sea: Sea{Wind: 8, Fetch: 100e3, Toward: 1},
		Salinity: 35, Celsius: 15, Chlorophyll: 0.3, Sun: Vec{0.5, 0.2, 0.6}}
	eye := Vec{Z: 2}
	camera := Camera{Eye: Subspace{eye, LookAt(eye, Vec{Y: 100, Z: 2})},
		Lens: NewLens(35, Gauge{}, 0), Width: 48, Height: 32, Seed: 2}
	a, err := View(scene, camera)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := View(scene, camera)
	for i := range a.Light {
		if a.Light[i] != b.Light[i] {
			t.Fatalf("the same view differs at pixel %d", i)
		}
		for _, v := range a.Light[i] {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < -1e-9 {
				t.Fatalf("pixel %d is %v", i, a.Light[i])
			}
		}
	}
	top := a.Light[(a.Height-1)*a.Width+a.Width/2]
	bottom := a.Light[a.Width/2]
	if top != ([3]float64{}) {
		t.Errorf("the sky, with no sky given, is %v", top)
	}
	if bottom[2] <= 0 {
		t.Errorf("the sea at the camera's feet is dark: %v", bottom)
	}
	for _, bad := range []Camera{
		{Eye: Subspace{Origin: Vec{Z: -1}}, Width: 4, Height: 4},
		{Eye: Subspace{Origin: Vec{Z: 2}}, Width: 0, Height: 4},
	} {
		if _, err := View(scene, bad); err == nil {
			t.Errorf("accepted camera %+v", bad)
		}
	}
}

// Straight down from high up, a view is all sea and, with every wave
// smaller than a pixel, perfectly smooth.
func TestViewFromOrbitIsSmooth(t *testing.T) {
	scene := Scene{Sea: north, Salinity: 35, Celsius: 15, Sun: Vec{0.2, 0, 1}}
	picture, err := View(scene, Camera{Eye: Subspace{Origin: Vec{Z: 400e3}},
		Lens: Swath(200, 400), Width: 16, Height: 16})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(picture.Light); i++ {
		if i%16 == 0 {
			continue
		}
		step := math.Abs(picture.Light[i][2] - picture.Light[i-1][2])
		if step > 0.5*picture.Light[i][2] {
			t.Fatalf("a jump of %v between neighbouring pixels", step)
		}
	}
}

// A sea stops growing: past the fetch that fully develops it, more fetch
// changes nothing, and its peak sits where Pierson and Moskowitz put it.
func TestSeaStopsGrowing(t *testing.T) {
	near2 := Sea{Wind: 10, Fetch: 2000e3}
	far := Sea{Wind: 10, Fetch: 20000e3}
	short := Sea{Wind: 10, Fetch: 20e3}
	if near2.Height() != far.Height() {
		t.Errorf("a fully developed sea kept growing")
	}
	if !(short.Height() < near2.Height()) {
		t.Errorf("a short fetch raised as much sea as a long one")
	}
	near(t, "fully developed peak", far.Peak(), 0.877*Gravity/10, 1e-9)
	// (from memory, not sources.yaml: a loose check only) Pierson and
	// Moskowitz's own sea, with no peak enhancement, is
	// 0.21 U^2/g high; JONSWAP's sharper peak makes this one somewhat
	// higher, and it should not be wildly so
	own := 0.21 * 100 / Gravity
	if got := far.Height(); got < own || got > 1.6*own {
		t.Errorf("a fully developed sea in 10 m/s is %v m; "+
			"Pierson-Moskowitz gives %v", got, own)
	}
}

// The sample points load, and answer only for the cells they sit in.
func TestSamplePointsLoad(t *testing.T) {
	file, err := os.Open("data/chlorophyll-points.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	grid, err := ReadGrid(file, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got, found := grid.At(37.5, -122.5, time.June); !found || got != 1.270 {
		t.Errorf("June off California is %v, %v", got, found)
	}
	if _, found := grid.At(60, -30, time.January); found {
		t.Errorf("a month with no data has a value")
	}
	if _, found := grid.At(10, -100, time.June); found {
		t.Errorf("a cell with no point in it has a value")
	}
	if len(grid.cells) != 251 {
		t.Errorf("%d values loaded", len(grid.cells))
	}
}

// The camera is slayer's: the same numbers out for the same numbers in.
// The expected values below were worked by hand from slayer's formulas,
// not copied from a run of slayer, which needs a newer Go than is here.
func TestCameraIsPatternedOnSlayers(t *testing.T) {
	// a 50 on Super 35 is the zero lens, and sees 2 atan(24.89/100)
	near(t, "the zero lens", Lens{}.Wide(), 2*math.Atan(24.89/100), 1e-12)
	full, err := GaugeNamed("full frame")
	if err != nil {
		t.Fatal(err)
	}
	near(t, "full frame's aspect", full.Aspect(), 1.5, 1e-12)
	near(t, "a 50 on full frame", NewLens(50, full, 0).Wide(),
		2*math.Atan(36.0/100), 1e-12)
	near(t, "Focal undoes Wide", Focal(NewLens(85, full, 0).Wide(), full),
		85, 1e-9)
	// slayer's own test of Swath frames forty kilometres from four hundred
	near(t, "a swath", Swath(40, 400).Wide(), 2*math.Atan(40.0/800), 1e-12)
	near(t, "no swath", Swath(0, 400).Wide(), Lens{}.Wide(), 1e-12)
	near(t, "a pinhole of no width", Pinhole{}.Wide(), math.Pi/3, 1e-12)
	near(t, "a pinhole too wide", NewPinhole(4, 0).Wide(), math.Pi/3, 1e-12)
	if _, err := GaugeNamed("vistavision"); err == nil {
		t.Errorf("a gauge not in the table was found")
	}
	if _, err := NewGauge("", 1, 1); err == nil {
		t.Errorf("a gauge with no name was made")
	}
	if len(Gauges()) != len(gaugeWidths) || len(Gauges()) != len(gaugeHeights) {
		t.Errorf("the gauge table's columns disagree")
	}

	// the middle of the picture is straight ahead, down minus z; the top
	// row looks up, the right column right, and the edges are Wide apart
	pin := NewPinhole(1, 0)
	middle := pin.Ray(0, 0, 1, 1).Dir
	if middle != (Vec{Z: -1}) {
		t.Errorf("the middle of a picture looks along %v", middle)
	}
	top := pin.Ray(50, 0, 101, 101).Dir
	right := pin.Ray(100, 50, 101, 101).Dir
	if !(top.Y > 0 && math.Abs(top.X) < 1e-12 && right.X > 0 &&
		math.Abs(right.Y) < 1e-12) {
		t.Errorf("top looks %v and right looks %v", top, right)
	}
	left := pin.Ray(0, 50, 101, 101).Dir
	edge := math.Acos(left.dot(right)) // between the outer pixels' middles
	near(t, "edge to edge", 2*math.Atan(math.Tan(edge/2)*101/100), 1, 1e-9)
	if got := pin.Ray(3, 3, 0, 0).Dir; got != (Vec{Z: -1}) {
		t.Errorf("a picture of no pixels looks along %v", got)
	}

	// an eye with no turn looks straight down with north up the picture;
	// LookAt faces what it is given with the world's up at the top
	down := Look(Subspace{Origin: Vec{Z: 10}}, nil, 0, 0, 1, 1)
	if down.At != (Vec{Z: 10}) || down.Dir != (Vec{Z: -1}) {
		t.Errorf("the empty eye looks from %v along %v", down.At, down.Dir)
	}
	if LookAt(Vec{Z: 10}, Vec{}) != Identity {
		t.Errorf("looking straight down is %v", LookAt(Vec{Z: 10}, Vec{}))
	}
	east := LookAt(Vec{Z: 2}, Vec{X: 50, Z: 2})
	ahead := east.Out(Vec{Z: -1})
	above := east.Out(Vec{Y: 1})
	if math.Abs(ahead.X-1) > 1e-12 || math.Abs(above.Z-1) > 1e-12 {
		t.Errorf("facing east: ahead %v, top of picture %v", ahead, above)
	}
	if LookAt(Vec{Z: 2}, Vec{Z: 2}) != Identity {
		t.Errorf("looking at where you stand is a turn")
	}
}
