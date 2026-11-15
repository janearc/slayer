package smoke

import (
	"compress/gzip"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
)

const (
	edgeOfRegard = 135.0 // published: the far edge of where Webb may point, degrees from the sun (STScI)
	momentWithin = 0.5   // picked: the moment is said to the minute and the sun's formula is good to an arcminute; the equator's turn since J2000, left out, is 0.4 degree
	axesNear     = 1e-12 // picked: unit vectors and right angles, to rounding
)

// At its moment the sun stands at the edge of where Webb may point: 135
// degrees from its star. This is the check that the date the smoke gives
// is the date the geometry happens; until it, nothing in the tree held
// the two together. The anti-test is a month earlier, when the star is
// well inside the edge.
func TestAtWebbsMomentTheSunIsAtTheEdge(t *testing.T) {
	_, _, _, fromSun := webbAxes(webbMoment)
	if math.Abs(fromSun-edgeOfRegard) > momentWithin {
		t.Errorf("at the moment the sun is %v degrees from the star, want %v",
			fromSun, edgeOfRegard)
	}
	_, _, _, earlier := webbAxes(webbMoment.AddDate(0, -1, 0))
	if math.Abs(earlier-edgeOfRegard) < 10*momentWithin {
		t.Errorf("a month earlier the sun is %v degrees from the star: the "+
			"check cannot tell one date from another", earlier)
	}
}

// Webb's axes are three unit vectors square to each other, right-handed,
// with the star along -z and the sun on the -y side with nothing of it
// along x: the sun is where the smoke's own lamp is.
func TestWebbsAxesPutTheStarAheadAndTheSunUnderTheShield(t *testing.T) {
	x, y, z, fromSun := webbAxes(webbMoment)
	for name, each := range map[string]maths.Vec{"x": x, "y": y, "z": z} {
		if math.Abs(each.Len()-1) > axesNear {
			t.Errorf("%s is %v long", name, each.Len())
		}
	}
	if math.Abs(x.Dot(y)) > axesNear || math.Abs(y.Dot(z)) > axesNear ||
		x.Cross(y).Sub(z).Len() > axesNear {
		t.Error("the axes are not square to each other and right-handed")
	}
	star := among(noahRound*maths.Degree, noahUp*maths.Degree)
	if got := maths.V(star.Dot(x), star.Dot(y), star.Dot(z)); got.Sub(maths.V(0, 0, -1)).Len() > axesNear {
		t.Errorf("in Webb's frame the star is toward %+v, want -z", got)
	}
	edge := fromSun * maths.Degree
	want := maths.V(0, -math.Sin(edge), -math.Cos(edge))
	round, up, _ := oceantex.SunAmongStars(webbMoment)
	sun := among(round, up)
	if got := maths.V(sun.Dot(x), sun.Dot(y), sun.Dot(z)); got.Sub(want).Len() > 1e-9 {
		t.Errorf("in Webb's frame the sun is toward %+v, want %+v", got, want)
	}
}

// With SLAYER_STARS set, a catalogue that is not there, is not gzip, or
// holds no star stops the picture, saying the file; and one that is
// sound gives the smoke its sky.
func TestAStarFolderThatCannotBeReadStopsThePicture(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(starsFolder, dir)
	path := filepath.Join(dir, starsFile)
	stopped := func(why string) {
		t.Helper()
		err := refusal(func() { webbSky() })
		if err == nil || !strings.Contains(err.Error(), path) {
			t.Errorf("%s: %v, want the picture stopped, naming %s", why, err, path)
		}
	}
	stopped("no file in the folder")
	if err := os.WriteFile(path, []byte("not gzip at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	stopped("a file that is not gzip")
	packed, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	squeezed := gzip.NewWriter(packed)
	if _, err := squeezed.Write([]byte("no star here\n")); err != nil {
		t.Fatal(err)
	}
	if squeezed.Close() != nil || packed.Close() != nil {
		t.Fatal("the test's own file could not be written")
	}
	stopped("a catalogue with no star in it")
}

// With SLAYER_STARS set, a catalogue with one star at Webb's own star's
// place puts that star straight ahead of the mirror, along -z. The
// anti-tests: it is nowhere else; and unset, the smoke has no stars.
func TestTheStarsBehindWebbAreTurnedToItsFrame(t *testing.T) {
	t.Setenv(starsFolder, "")
	if _, some := webbSky(); some {
		t.Fatal("there are stars with no folder named")
	}
	// 287.50086 degrees is 19 h 10 m 00.2 s; -22.23481 is -22 14 05
	line := strings.Repeat(" ", 75) + "191000.2-221405" +
		strings.Repeat(" ", 12) + fmt.Sprintf("%5s", "1.00")
	dir := t.TempDir()
	packed, err := os.Create(filepath.Join(dir, starsFile))
	if err != nil {
		t.Fatal(err)
	}
	squeezed := gzip.NewWriter(packed)
	if _, err := squeezed.Write([]byte(line + "\n")); err != nil {
		t.Fatal(err)
	}
	if err := squeezed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := packed.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv(starsFolder, dir)
	stars, some := webbSky()
	if !some {
		t.Fatal("no stars with the folder named")
	}
	if ahead := stars.Light(maths.V(0, 0, -1)); !(ahead.G > 0) {
		t.Errorf("straight ahead of the mirror the sky is %+v, want the star", ahead)
	}
	// and the smoke itself has that sky, where unset it has none
	if stage, _ := fifty(); stage.Sky == nil || !(stage.Sky.Light(maths.V(0, 0, -1)).G > 0) {
		t.Error("with the folder named, smoke 50's sky does not hold the star")
	}
	for name, toward := range map[string]maths.Vec{"behind": maths.V(0, 0, 1),
		"above": maths.V(0, 1, 0), "aside": maths.V(1, 0, 0),
		"a degree off": maths.V(math.Sin(maths.Degree), 0, -math.Cos(maths.Degree))} {
		if got := stars.Light(toward); got.G != 0 {
			t.Errorf("%s the sky is %+v, want the dark", name, got)
		}
	}
}
