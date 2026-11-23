//go:build draw

package session

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// burningHill is the command's arguments for a small hillside at night
// with one spruce on fire and nothing else lit but the moon, exposed for
// the fire: the request burningHillAsked says as fields.
func burningHill() []string {
	return append([]string{"-w", "40", "-h", "30", "-n", "1", "-night",
		"-alight", "1", "-exposure", "100000"},
		only(map[string]string{"spruce": "1", "boulder": "1"})...)
}

// smallSetAsked is smallAsked for the set file at path, with the size and
// the rays said, so that they stand over the set's film as they do when
// the command line gives them beside -set.
func smallSetAsked(path string) asked {
	given := smallAsked()
	given.set = path
	given.said.width, given.said.height, given.said.samples = true, true,
		true
	return given
}

// light is all the light of a picture together.
func light(drawn *frame.Frame) float64 {
	sum := 0.0
	for _, pixel := range drawn.Pix {
		sum += float64(pixel.R + pixel.G + pixel.B)
	}
	return sum
}

// a fire lights what is near it: the burning hillside drawn as made,
// and again with the fire's lamps taken out and only the moon's two
// left. the flame glows alike in both, so what the first has more of is
// the fire's light on what is near it.
//
// a behaviour, not a law: the picture's light, the flame's own glow
// in it, comes to more with the fire's light, and the test asks for a
// tenth more. measured 464 against 395 once a matte surface took its
// colour over pi of what lands on it. the bar was a quarter, picked
// from 618 against 400 measured before that, by a tracer pi too
// bright: the fire's added light fell from 218 to 69, and 218 over pi
// is 69.4. a moved bar, listed to be judged.
func TestAFireLightsTheGround(t *testing.T) {
	given := mustParse(t, burningHill()...)
	world, made, err := hillside(given)
	if err != nil {
		t.Fatal(err)
	}
	const moons = 2
	if made.lamps <= moons || len(world.Lamps) != made.lamps {
		t.Fatalf("%d lamps counted, %d in the world", made.lamps,
			len(world.Lamps))
	}
	lit, err := draw(world, given, made.across)
	if err != nil {
		t.Fatal(err)
	}
	world.Lamps = world.Lamps[:moons]
	unlit, err := draw(world, given, made.across)
	if err != nil {
		t.Fatal(err)
	}
	if !(light(lit) > 1.1*light(unlit)) {
		t.Errorf("the hillside is %v with the fire's light and %v "+
			"without it", light(lit), light(unlit))
	}
}

// fermat proves a burning hillside: a pixel through the fire, named, and more
// picked, each made again bit for bit, the named one's page telling the steps
// through the flame and how hot it was at each.
func TestABurningHillsideIsProved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "burning.pfm")
	drawing := append(burningHill(), "-out", filepath.Join(dir, "b.png"),
		"-raw", path)
	var wrong bytes.Buffer
	if code := run(drawing, io.Discard, &wrong); code != 0 {
		t.Fatalf("drawing gave %d: %s", code, wrong.String())
	}
	var said bytes.Buffer
	proving := append(burningHill(), "-prove", path, "-pixel", "20,12",
		"-many", "8")
	if code := run(proving, &said, &wrong); code != 0 ||
		!strings.Contains(said.String(), "holds: 9 of 9") {
		t.Fatalf("gave %d: %s %s", code, said.String(), wrong.String())
	}
	page, err := os.ReadFile(filepath.Join(dir, "burning-proof",
		"pixel-20-12.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "a step through the flame") ||
		!strings.Contains(string(page), "kelvin=") {
		t.Errorf("the page through the fire tells no flame:\n%s", page)
	}
}
