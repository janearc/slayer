package sky

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// A star's light and the size it is drawn, for these tests.
var (
	starlight = maths.RGB(2, 3, 4)
	faint     = maths.Grey(0.01)
)

const (
	starSize  = 0.2 * maths.Degree // picked: a star's radius here
	skyNear   = 1e-12              // picked: sums of a few lights, to rounding
	aDegree   = maths.Degree       // defined: a degree, in radians
	manyStars = 400                // picked: stars in the random skies
	onAnEdge  = 1e-12              // picked: a cosine this near a disc's is on its edge, where rounding decides
)

// way is the unit vector so many degrees round and so many up.
func way(round, up float64) maths.Vec {
	r, u := round*aDegree, up*aDegree
	return maths.V(math.Cos(u)*math.Cos(r), math.Cos(u)*math.Sin(r), math.Sin(u))
}

func sameLight(one, other maths.Light) bool {
	return math.Abs(one.R-other.R) < skyNear && math.Abs(one.G-other.G) < skyNear &&
		math.Abs(one.B-other.B) < skyNear
}

// A star is seen where it is, with the dark behind it, and a degree away
// there is only the dark. Two stars in one place are both seen.
func TestACatalogueShowsAStarWhereItIs(t *testing.T) {
	made, err := NewCatalogue([]Star{{way(40, 10), starlight},
		{way(200, -60), starlight}, {way(200, -60), starlight}}, starSize, faint)
	if err != nil {
		t.Fatal(err)
	}
	if got := made.Light(way(40, 10)); !sameLight(got, faint.Add(starlight)) {
		t.Errorf("toward the star the sky is %+v", got)
	}
	if got := made.Light(way(40.1, 10.1)); !sameLight(got, faint.Add(starlight)) {
		t.Errorf("a tenth of a degree off, inside its disc, the sky is %+v", got)
	}
	if got := made.Light(way(41, 10)); !sameLight(got, faint) {
		t.Errorf("a degree from the star the sky is %+v, not the dark", got)
	}
	if got := made.Light(way(200, -60)); !sameLight(got, faint.Add(starlight).Add(starlight)) {
		t.Errorf("toward two stars in one place the sky is %+v", got)
	}
	if got := made.Light(way(40, 10).Scale(7)); !sameLight(got, faint.Add(starlight)) {
		t.Errorf("a long way toward the star is %+v: its length mattered", got)
	}
	if got := made.Light(maths.Vec{}); !sameLight(got, faint) {
		t.Errorf("a look no way at all sees %+v, not the dark", got)
	}
}

// The index misses nothing at its seams: a star at a pole is seen from
// just beside the pole on its far side, and a star on the line where the
// sectors wrap is seen from just across it.
func TestACatalogueFindsStarsAcrossItsSeams(t *testing.T) {
	made, err := NewCatalogue([]Star{{way(0, 90), starlight},
		{way(10, -89.95), starlight}, {way(179.99, 20), starlight},
		{way(-179, 88), starlight}, {way(179, -88), starlight}}, starSize, faint)
	if err != nil {
		t.Fatal(err)
	}
	looks := map[string]maths.Vec{
		// and one in a polar band's last sector is seen from its first
		"in the other polar band, from across its wrap": way(-179.5, -88),
		// near a pole the whole band is searched: a star in its first
		// sector is seen once, from its last, and not missed or doubled
		"in a polar band, from across its wrap": way(179.5, 88),
		"the north pole, from beside it":        way(137, 89.9),
		"near the south pole, from past it":     way(190, -89.95),
		"across the line the sectors wrap at":   way(-179.95, 20),
	}
	for name, toward := range looks {
		if got := made.Light(toward); !sameLight(got, faint.Add(starlight)) {
			t.Errorf("%s the sky is %+v, want the star", name, got)
		}
	}
}

// Whatever the stars and wherever the look, the catalogue gives what
// adding up every star by hand gives: its index only saves the work.
func TestACatalogueIsTheSumOfItsStars(t *testing.T) {
	rapid.Check(t, func(draw *rapid.T) {
		stars := make([]Star, manyStars)
		for at := range stars {
			stars[at] = Star{way(rapid.Float64Range(-180, 180).Draw(draw, "round"),
				rapid.Float64Range(-90, 90).Draw(draw, "up")), starlight}
		}
		// any size a catalogue takes, up to the widest: the index is
		// held to finding a star as wide as a degree from its far edge
		size := rapid.Float64Range(0.05, 1).Draw(draw, "size") * aDegree
		made, err := NewCatalogue(stars, size, faint)
		if err != nil {
			draw.Fatal(err)
		}
		// look near a star, as far off as its disc is wide and a little
		// further, where a miss would show
		near := stars[rapid.IntRange(0, manyStars-1).Draw(draw, "which")].Toward
		toward := near.Add(maths.V(rapid.Float64Range(-0.02, 0.02).Draw(draw, "dx"),
			rapid.Float64Range(-0.02, 0.02).Draw(draw, "dy"),
			rapid.Float64Range(-0.02, 0.02).Draw(draw, "dz"))).Unit()
		// every star well inside its disc is seen and none well outside
		// is; one within rounding of its disc's edge may be either, the
		// catalogue having made its way a unit long and this sum not
		var inside, edge int
		for _, each := range stars {
			switch cosine := each.Toward.Dot(toward) - math.Cos(size); {
			case cosine >= onAnEdge:
				inside++
			case cosine > -onAnEdge:
				edge++
			}
		}
		want := faint
		for range inside {
			want = want.Add(starlight)
		}
		got := made.Light(toward)
		same := sameLight(got, want)
		for range edge {
			want = want.Add(starlight)
			same = same || sameLight(got, want)
		}
		if !same {
			draw.Fatalf("toward %+v the catalogue gives %+v; %d stars are "+
				"in their discs and %d on an edge", toward, got, inside, edge)
		}
	})
}

// A catalogue that cannot be drawn is refused.
func TestACatalogueThatCannotBeDrawnIsRefused(t *testing.T) {
	one := []Star{{way(0, 0), starlight}}
	refused := map[string]struct {
		stars []Star
		size  float64
	}{
		"no stars":                 {nil, starSize},
		"a size of nought":         {one, 0},
		"a size below nought":      {one, -starSize},
		"a size over a degree":     {one, 1.5 * aDegree},
		"a size that is no number": {one, math.NaN()},
		"a star with no way":       {[]Star{{maths.Vec{}, starlight}}, starSize},
		"a star of no number":      {[]Star{{way(0, 0), maths.RGB(math.NaN(), 0, 0)}}, starSize},
	}
	for name, each := range refused {
		if _, err := NewCatalogue(each.stars, each.size, faint); err == nil {
			t.Errorf("%s: made", name)
		}
	}
	if made, err := NewCatalogue(one, starSize, faint); err != nil || made.Count() != 1 {
		t.Errorf("one sound star: %v, %v", made, err)
	}
}

// brightLine is a line of the bright star catalogue with a place and a
// magnitude in their columns. Every other column holds a seven, as the
// real file's hold other numbers, so a field read a column too wide or
// from a column too far left takes in a seven and is seen to.
func brightLine(hours, minutes int, seconds float64, sign string,
	degrees, arcmin, arcsec int, visual string) string {
	return strings.Repeat("7", 75) +
		fmt.Sprintf("%02d%02d%04.1f%s%02d%02d%02d", hours, minutes, seconds,
			sign, degrees, arcmin, arcsec) +
		strings.Repeat("7", 12) + fmt.Sprintf("%5s", visual) + "777"
}

// A star with minutes and seconds in every field is read to where it
// is. The place is Sirius's, 06 45 08.9 and -16 42 58, and the way to it
// is worked by hand: right ascension 101.28708 degrees, declination
// -16.71611, so cos d is 0.957742, and the way is (-0.187454, 0.939218,
// -0.287630). A field read from the wrong column moves it by more than
// the test allows: the smallest slip, the seconds of arc read as 8 for
// 58, moves it 0.00024.
func TestABrightStarsMinutesAndSecondsAreRead(t *testing.T) {
	file := brightLine(6, 45, 8.9, "-", 16, 42, 58, "-1.46")
	stars, err := BrightStars(strings.NewReader(file), maths.Grey(1))
	if err != nil || len(stars) != 1 {
		t.Fatalf("read %d stars, %v", len(stars), err)
	}
	want := maths.V(-0.187454, 0.939218, -0.287630)
	if off := stars[0].Toward.Sub(want).Len(); off > 1e-4 {
		t.Errorf("the star is toward %+v, %v from %+v", stars[0].Toward, off, want)
	}
	// 10 to the 0.584 is 3.837
	if math.Abs(stars[0].Light.G-3.837) > 0.001 {
		t.Errorf("a star of magnitude -1.46 is %v bright, want 3.837", stars[0].Light.G)
	}
}

// The index's bands run from the south pole to the north and its sectors
// once round from the far side: the south pole is in the first band, the
// north in the last and the equator in the middle one; and round the
// equator a way just past -180 degrees is in the first sector, one at
// nought in the middle and one just short of 180 in the last. An index
// that put every way in one band or one sector would still give the
// right light, by searching everything; this is what holds it to being
// an index.
func TestACellIsTheBandAndTheSectorAWayLiesIn(t *testing.T) {
	bands := map[string]struct {
		toward maths.Vec
		want   int
	}{
		"the south pole": {way(0, -90), 0},
		"the equator":    {way(0, 0), catalogueBands / 2},
		"the north pole": {way(0, 90), catalogueBands - 1},
	}
	for name, each := range bands {
		if band, _ := cell(each.toward); band != each.want {
			t.Errorf("%s is in band %d, want %d", name, band, each.want)
		}
	}
	sectors := map[string]struct {
		toward maths.Vec
		want   int
	}{
		"just past -180 degrees": {way(-179.9, 0), 0},
		"nought":                 {way(0, 0), catalogueSectors / 2},
		"just short of 180":      {way(179.9, 0), catalogueSectors - 1},
	}
	for name, each := range sectors {
		if _, sector := cell(each.toward); sector != each.want {
			t.Errorf("%s is in sector %d, want %d", name, sector, each.want)
		}
	}
}

// Near a pole a sector is narrower than a star may be wide, which is why
// the bands there are searched whole. At 69.5 degrees up, in the last
// band below each cap's edge, a sector is 0.985 degrees of arc: a star a
// degree in radius just short of a sector's edge is in the disc of a
// look 2.827 degrees further round, 0.99 degrees of arc away and two
// sectors over. The anti-test is a look a little further round still,
// outside the disc, which sees the dark.
func TestAWideStarNearAPoleIsFoundTwoSectorsOver(t *testing.T) {
	const (
		high    = 69.5  // worked: inside the band next to each cap's last
		before  = -0.01 // picked: just short of the sector edge at nought
		within  = 2.817 // worked: 0.99 degrees of arc from the star
		outside = 2.917 // worked: 1.025 degrees of arc from the star
	)
	for name, up := range map[string]float64{"north": high, "south": -high} {
		made, err := NewCatalogue([]Star{{way(before, up), starlight}},
			aDegree, faint)
		if err != nil {
			t.Fatal(err)
		}
		_, star := cell(way(before, up))
		if _, look := cell(way(within, up)); look-star != 2 {
			t.Fatalf("%s: the look is %d sectors from the star, want 2: "+
				"the test holds nothing", name, look-star)
		}
		if got := made.Light(way(within, up)); !sameLight(got, faint.Add(starlight)) {
			t.Errorf("%s: a look in the star's disc two sectors over "+
				"sees %v, want the star", name, got)
		}
		if got := made.Light(way(outside, up)); !sameLight(got, faint) {
			t.Errorf("%s: a look outside the star's disc sees %v, want "+
				"the dark", name, got)
		}
	}
}

// A way exactly on the edge of a star's disc is in the disc. The star is
// along x, and the way is the cosine of the disc's size along x and what
// is left along y: where rounding leaves that way exactly a unit long,
// its cosine with the star is exactly the disc's. Not every size rounds
// so, so several are tried and each that does is held to it; the
// anti-test is a way a hundredth of the size further out, which sees
// the dark.
func TestAWayOnADiscsEdgeIsInTheDisc(t *testing.T) {
	var held int
	for tenths := 1; tenths <= 10; tenths++ {
		size := float64(tenths) / 10 * aDegree
		edge := math.Cos(size)
		on := maths.V(edge, math.Sqrt(1-edge*edge), 0)
		if on.Len() != 1 {
			continue
		}
		held++
		made, err := NewCatalogue([]Star{{maths.V(1, 0, 0), starlight}}, size, faint)
		if err != nil {
			t.Fatal(err)
		}
		if got := made.Light(on); !sameLight(got, faint.Add(starlight)) {
			t.Errorf("a way on the edge of a disc %d tenths of a degree "+
				"wide sees %v, want the star", tenths, got)
		}
		beyond := size * 1.01
		off := maths.V(math.Cos(beyond), math.Sin(beyond), 0)
		if got := made.Light(off); !sameLight(got, faint) {
			t.Errorf("a way past the edge of a disc %d tenths of a degree "+
				"wide sees %v, want the dark", tenths, got)
		}
	}
	if held == 0 {
		t.Fatal("no size tried puts a way exactly on its disc's edge: " +
			"the test holds nothing")
	}
}

// A line may end with its magnitude: the catalogue's lines run on past
// it, but a line cut off just after it still holds a star. And seconds
// of time may be ten or more: 45.0 seconds of right ascension on the
// equator is 0.1875 degrees round, whose sine is 0.00327248.
func TestABrightStarsLineMayEndAtItsMagnitudeAndItsSecondsBeTens(t *testing.T) {
	whole := brightLine(0, 0, 45.0, "+", 0, 0, 0, " 0.00")
	cut := whole[:len(whole)-3]
	stars, err := BrightStars(strings.NewReader(cut), maths.Grey(1))
	if err != nil || len(stars) != 1 {
		t.Fatalf("a line that ends at its magnitude read as %d stars, %v",
			len(stars), err)
	}
	if got := stars[0].Toward.Y; math.Abs(got-0.00327248) > 1e-7 {
		t.Errorf("45 seconds of time round is %v along y, want 0.00327248", got)
	}
	// the anti-test: a line a column shorter has lost its magnitude's
	// last figure with it, and is still read, as a brighter star, by
	// what is left; one cut before its magnitude holds no star at all
	if _, err := BrightStars(strings.NewReader(whole[:100]), maths.Grey(1)); err == nil {
		t.Error("a line cut off before its magnitude was read as a star")
	}
}

// A star whose way is not a unit long is seen where it is all the same,
// and a star whose light is an infinity below nought is refused as one
// above is.
func TestACatalogueTakesAnyLengthOfWayAndNoInfinity(t *testing.T) {
	made, err := NewCatalogue([]Star{{way(40, 10).Scale(5), starlight}}, starSize, faint)
	if err != nil {
		t.Fatal(err)
	}
	if got := made.Light(way(40, 10)); !sameLight(got, faint.Add(starlight)) {
		t.Errorf("toward a star given five long the sky is %+v", got)
	}
	if got := made.Light(way(41, 10)); !sameLight(got, faint) {
		t.Errorf("a degree from it the sky is %+v", got)
	}
	for name, light := range map[string]maths.Light{
		"an infinity above": maths.RGB(math.Inf(1), 0, 0),
		"an infinity below": maths.RGB(0, 0, math.Inf(-1))} {
		if _, err := NewCatalogue([]Star{{way(0, 0), light}}, starSize, faint); err == nil {
			t.Errorf("a star of %s was taken", name)
		}
	}
}

// The bright star catalogue's lines are read where their columns are:
// six hours of right ascension on the equator is along y, the pole is
// along z, and a star five magnitudes fainter is a hundredth as bright.
// A line with no magnitude is no star.
func TestBrightStarsAreReadFromTheirColumns(t *testing.T) {
	file := strings.Join([]string{
		brightLine(6, 0, 0, "+", 0, 0, 0, " 0.00"),
		brightLine(0, 0, 0, "+", 90, 0, 0, " 5.00"),
		brightLine(12, 0, 0, "-", 30, 0, 0, "-2.50"),
		brightLine(3, 0, 0, "+", 10, 0, 0, ""),
		"a short line",
	}, "\n")
	stars, err := BrightStars(strings.NewReader(file), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(stars) != 3 {
		t.Fatalf("read %d stars, want the three with a magnitude", len(stars))
	}
	want := []struct {
		toward maths.Vec
		light  float64
	}{
		{maths.V(0, 1, 0), 1},
		{maths.V(0, 0, 1), 0.01},
		{maths.V(-math.Sqrt(3)/2, 0, -0.5), 10},
	}
	for at, each := range want {
		if stars[at].Toward.Sub(each.toward).Len() > 1e-9 {
			t.Errorf("star %d is toward %+v, want %+v", at, stars[at].Toward, each.toward)
		}
		if math.Abs(stars[at].Light.G-each.light) > 1e-9*each.light {
			t.Errorf("star %d is %v bright, want %v", at, stars[at].Light.G, each.light)
		}
	}
}

// A line whose columns do not hold numbers is refused by its number, and
// a file with no star in it is refused.
func TestABrightStarFileThatCannotBeReadIsRefused(t *testing.T) {
	garbled := brightLine(6, 0, 0, "+", 0, 0, 0, " 0.00") + "\n" +
		strings.Replace(brightLine(7, 0, 0, "+", 0, 0, 0, " 1.00"), "07", "xx", 1)
	if _, err := BrightStars(strings.NewReader(garbled), maths.Grey(1)); err == nil ||
		!strings.Contains(err.Error(), "line 2") {
		t.Errorf("a garbled second line: %v, want a refusal that names line 2", err)
	}
	if _, err := BrightStars(strings.NewReader("nothing\nhere\n"), maths.Grey(1)); err == nil {
		t.Error("a file with no star in it was read")
	}
	// each field garbled in turn is refused: the declination and the
	// magnitude as the right ascension is
	sound := brightLine(7, 30, 15.5, "+", 12, 20, 40, " 1.00")
	garbles := map[string]struct{ was, is string }{
		"the declination's degrees": {"+122040", "+x22040"},
		"its minutes of arc":        {"+122040", "+12x040"},
		"the magnitude":             {" 1.00", " 1.x0"},
	}
	for name, each := range garbles {
		if !strings.Contains(sound, each.was) {
			t.Fatalf("%s: the line has no %q to garble", name, each.was)
		}
		bad := strings.Replace(sound, each.was, each.is, 1)
		if _, err := BrightStars(strings.NewReader(bad), maths.Grey(1)); err == nil {
			t.Errorf("%s garbled: read", name)
		}
	}
	// a line too long for the reader to hold is a failure of reading,
	// and is said, not passed over as the end of the file
	long := sound + "\n" + strings.Repeat("7", 100000) + "\n"
	if _, err := BrightStars(strings.NewReader(long), maths.Grey(1)); err == nil {
		t.Error("a file with a line too long to read was read")
	}
}
