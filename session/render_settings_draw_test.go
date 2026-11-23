//go:build draw

package session

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// number is a size or a count as the flags give it, as a number.
func number(t *testing.T, given string) int {
	t.Helper()
	made, err := strconv.Atoi(given)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// exit numbers hillside gives, as its documentation of run says them.
const (
	drawnNumber   = 0 // defined: a picture was written
	refusedNumber = 2 // defined: flags it could not take
)

// the sizes and rays the tests ask for on the command line, as the flags
// are given: strings, as a person types them.
const (
	askedWide  = "64"
	askedTall  = "36"
	askedRays  = "2"
	filmWide   = "40"
	filmTall   = "24"
	filmRays   = "1"
	oneRay     = "1 rays a pixel"
	fourRays   = "4 rays a pixel"
	sizedFilm  = "    width: " + filmWide + "\n    height: " + filmTall + "\n    rays: " + filmRays + "\n"
	widowWidth = "    wide: 40\n"
)

// sizedWidow is the widow set with a film that says its size, written to a
// fresh directory; and its path.
func sizedWidow(t *testing.T) string {
	t.Helper()
	return widowChanged(t, widowWidth, sizedFilm+widowWidth)
}

// drawnAs is the size of the picture a command line draws, the line it
// says of it, and the number it exits with.
func drawnAs(t *testing.T, arguments ...string) (image.Point, int, string,
	string) {
	t.Helper()
	var none image.Point
	out := filepath.Join(t.TempDir(), "picture.png")
	var told, complained bytes.Buffer
	code := run(append(arguments, "-out", out), &told, &complained)
	if code != drawnNumber {
		return none, code, told.String(), complained.String()
	}
	written, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer written.Close()
	picture, err := png.Decode(written)
	if err != nil {
		t.Fatal(err)
	}
	return picture.Bounds().Size(), code, told.String(), complained.String()
}

// with a set, -w, -h and -n say the size and the rays of the picture and
// the set's film says them where they are not given: the film's own when
// none is, and each one that is given in place of the film's alone.
func TestSizeAndRaysAreSettingsOfASet(t *testing.T) {
	path := sizedWidow(t)
	for _, x := range []struct {
		flags      []string
		wide, tall string
		rays       string
	}{
		{nil, filmWide, filmTall, oneRay},
		{[]string{"-w", askedWide, "-h", askedTall, "-n", askedRays},
			askedWide, askedTall, fourRays},
		{[]string{"-w", askedWide}, askedWide, filmTall, oneRay},
		{[]string{"-h", askedTall}, filmWide, askedTall, oneRay},
		{[]string{"-n", askedRays}, filmWide, filmTall, fourRays},
	} {
		arguments := append([]string{"-set", path}, x.flags...)
		size, code, said, wrong := drawnAs(t, arguments...)
		if code != drawnNumber || size.X != number(t, x.wide) ||
			size.Y != number(t, x.tall) || !strings.Contains(said, x.rays) {
			t.Errorf("%v: exit %d, %v, said %q, wrong %q; wanted %sx%s "+
				"at %s", x.flags, code, size, said, wrong, x.wide, x.tall,
				x.rays)
		}
	}
}

// the flags that say what a set says are still refused with a set, each
// by name, whether or not a size is given beside them.
func TestEveryOtherFlagIsStillRefusedWithASet(t *testing.T) {
	path := sizedWidow(t)
	for _, x := range []struct {
		flags []string
		name  string
	}{
		{[]string{"-seed", "3"}, "seed"}, {[]string{"-night"}, "night"},
		{[]string{"-w", askedWide, "-spruce", "2"}, "spruce"},
		{[]string{"-exposure", "5"}, "exposure"},
		{[]string{"-n", askedRays, "-h", askedTall, "-fern", "1"}, "fern"},
	} {
		var told, wrong bytes.Buffer
		code := run(append([]string{"-set", path}, x.flags...), &told, &wrong)
		want := "hillside: asked for a hillside that cannot be drawn: " +
			"-set says the world and the film, and -" + x.name +
			" says them too\n"
		if code != refusedNumber || wrong.String() != want {
			t.Errorf("%v: exit %d and %q, not %d and %q", x.flags, code,
				wrong.String(), refusedNumber, want)
		}
	}
}

// a set whose film is a rig has its height from the rig's frame and the
// width, so -h beside it is refused, naming what says the height; -w
// still sets the width, and the height follows it.
func TestAHeightIsRefusedWithARigAndAWidthIsNot(t *testing.T) {
	given := mustParse(t)
	given.set = onFixtureRigged(t, "0.5", false)
	given.said.height = true
	if _, _, _, err := staged(given); err == nil ||
		!strings.Contains(err.Error(), "-h") ||
		!strings.Contains(err.Error(), "frame") {
		t.Errorf("a height with a rig: %v", err)
	}
	given.said.height = false
	given.said.width = true
	_, _, got, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if got.width != given.width || got.height != got.frame.Rows(got.width) {
		t.Errorf("drawn %dx%d, not %d wide and the frame's rows", got.width,
			got.height, given.width)
	}
}

// a world-only set that gives no film size is drawn 320 pixels wide, 200
// tall (the 8 to 5 aspect of the usual picture), at 2 rays a pixel along
// each side: that is the default size.
func TestAWorldOnlySetWithNoSizeIsDrawnAtTheDefault(t *testing.T) {
	const (
		defaultWide = 320
		defaultTall = 200
		defaultRays = 2
	)
	asked, err := parse([]string{"-set", "../sets/widow.yaml"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	_, _, got, err := staged(asked)
	if err != nil {
		t.Fatal(err)
	}
	if got.width != defaultWide || got.height != defaultTall ||
		got.samples != defaultRays {
		t.Errorf("staged %dx%d at %d rays, not %dx%d at %d", got.width,
			got.height, got.samples, defaultWide, defaultTall, defaultRays)
	}
}
