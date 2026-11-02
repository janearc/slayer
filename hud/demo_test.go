package hud_test

import (
	"bytes"
	"flag"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/hud"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/shape"
	skies "github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// pictures, given a directory, has the demonstration write its picture
// with the hud drawn on it, and the terminal's cells with the hud over
// them as the escape codes a terminal draws, for an eye to look at.
var pictures = flag.String("pictures", "",
	"write hud.png and hud.ans, the hud over a traced scene, here")

// stage is a clay ball on a chalk floor under a sun, and a second ball
// behind the first from the eye, seen from in front and above.
func stage() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.6)}})
	clay := &thing.Material{Colour: maths.RGB(0.7, 0.35, 0.2)}
	for _, at := range []maths.Vec{maths.V(0, 1, 0), maths.V(0, 1, -6)} {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Material: clay,
			Motion:   thing.Still(maths.Subspace{Origin: at})})
	}
	world.Lamp(thing.Sun(maths.V(-0.4, 0.8, 0.5).Unit(),
		maths.Grey(math.Pi)))
	world.Sky = skies.Gradient{Horizon: maths.Grey(0.5),
		Zenith: maths.RGB(0.15, 0.25, 0.5)}
	from := maths.V(0, 2.5, 7)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0.8, 0))}
}

// display is the demonstration's hud: a frame round the screen, a title,
// a bar filling over four seconds with its count, a label pinned to the
// near ball, and one pinned to the far ball that asks to be hidden
// behind the near one, and is.
func display() hud.Hud {
	filling := hud.Ramp{From: 0, To: 1, Over: 4 * time.Second}
	return hud.Hud{Elements: []hud.Element{
		hud.Frame{Box: hud.Box{0, 0, 1, 1}, Ink: "border"},
		hud.Text{Words: hud.Words(" slayer "), At: hud.Point{0.5, 0},
			Align: hud.Centre, Ink: "ink", Ground: "ground"},
		hud.Bar{Box: hud.Box{0.1, 0.86, 0.7, 0.91}, Value: filling,
			Fill: "ok", Empty: "surface-1"},
		hud.Text{Words: hud.Count{Of: hud.Ramp{To: 100,
			Over: 4 * time.Second}, Format: "%3.0f%%"},
			At: hud.Point{0.72, 0.88}, Ink: "ink",
			Ground: "ground"},
		hud.Label{Point: maths.V(1, 2, 0), Words: hud.Words("< near"),
			Ink: "ink", Ground: "ground", Occluded: true},
		hud.Label{Point: maths.V(0, 1, -6), Words: hud.Words("far"),
			Ink: "warn", Occluded: true},
	}}
}

// the demonstration draws: the near ball's label is laid out, the far
// ball's, hidden behind the near one, is not, and at two seconds the
// bar is half full; with -pictures it writes what it drew.
func TestTheDemonstrationDraws(t *testing.T) {
	world, eye := stage()
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	const cols, rows = 60, 20
	cells, view := traced(t, world, eye, lens, cols, rows)
	cover, err := display().Cover(image.Rect(0, 0, cols, rows),
		2*time.Second, theme(), view)
	if err != nil {
		t.Fatal(err)
	}
	words := text(cover, cols, rows)
	if !strings.Contains(words, "< near") ||
		strings.Contains(words, "far") ||
		!strings.Contains(words, " 50%") {
		t.Errorf("the demonstration says:\n%s", words)
	}
	if *pictures != "" {
		write(t, world, eye, cover, cells)
	}
}

// traced is the stage traced at sextant size for a terminal so many
// cells across and down, chopped into cells, and the view it was seen
// through.
func traced(t *testing.T, world *thing.World, eye maths.Subspace,
	lens film.Pinhole, cols, rows int) (*terminal.Cells, hud.View) {
	t.Helper()
	drawn, err := trace.Render(world, eye, lens, 0, cols*2, rows*3)
	if err != nil {
		t.Fatal(err)
	}
	view := hud.View{Eye: eye, Sensor: lens, Width: cols * 2,
		Height: rows * 3, Blocker: hud.Seen(world, 0)}
	return terminal.Chop(colour.Develop(drawn, 1)), view
}

// text is a cover's characters, row by row, a space where it covers
// nothing.
func text(cover hud.Cover, cols, rows int) string {
	var lines strings.Builder
	for y := range rows {
		for x := range cols {
			cell, ok := cover.At(x, y)
			if !ok {
				cell.Rune = ' '
			}
			lines.WriteRune(cell.Rune)
		}
		lines.WriteByte('\n')
	}
	return lines.String()
}

// write writes the hud over the terminal's cells as escape codes, and
// drawn on the stage traced at 960 by 640 as a png.
func write(t *testing.T, world *thing.World, eye maths.Subspace,
	cover hud.Cover, cells *terminal.Cells) {
	t.Helper()
	over, err := cover.Over(cells)
	if err != nil {
		t.Fatal(err)
	}
	var codes bytes.Buffer
	if err := terminal.Draw(&codes, over); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(*pictures, "hud.ans"), codes.Bytes(), 0o644)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	drawn, err := trace.Render(world, eye, lens, 0, 960, 640)
	if err != nil {
		t.Fatal(err)
	}
	img := colour.Develop(drawn, 1)
	view := hud.View{Eye: eye, Sensor: lens, Width: 960, Height: 640,
		Blocker: hud.Seen(world, 0)}
	if err := display().Picture(img, 24, 2*time.Second, theme(),
		view); err != nil {
		t.Fatal(err)
	}
	if err := file.PNG(filepath.Join(*pictures, "hud.png"), img,
		1); err != nil {
		t.Fatal(err)
	}
}
