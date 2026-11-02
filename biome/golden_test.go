package biome_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/maths"
)

// update rewrites the golden worlds from what is made now, by hand, by a
// change meant to move them.
var update = flag.Bool("update", false,
	"rewrite testdata/golden/world.txt from the worlds made now")

const (
	// filed is how the golden file is opened to write, defined: read and
	// written by its owner, read by others
	filed = 0o644
	// folder is how the golden folder is made, defined: entered, read
	// and written by its owner, entered and read by others
	folder = 0o755
)

// drawn is a light as the golden file writes it.
func drawn(light maths.Light) string {
	return fmt.Sprintf("%.9g %.9g %.9g", light.R, light.G, light.B)
}

// golden is a world the golden file holds, by a name for its head.
type golden struct {
	name  string
	world *biome.World
}

// goldens is each world the golden file holds.
func goldens(test testing.TB) []golden {
	test.Helper()
	flatWorld, _ := made(test, nought)
	flatWorld.Lights = lights(test)
	return []golden{
		{"the flat world, its cities lit in three lights", flatWorld},
		{"the ring, its cities sprawling in three lights",
			sprawled(test, lights(test))},
		{"the ring, its cities sprawling in sodium", sprawled(test, nil)},
	}
}

// pictured is the golden worlds, each under its name, then a line a
// tile: where it is, its paint, and what it gives off.
func pictured(test testing.TB) string {
	test.Helper()
	var lines strings.Builder
	for _, golden := range goldens(test) {
		world := golden.world
		fmt.Fprintf(&lines, "# %s: %d rivers, %d lakes, %d volcanoes, "+
			"%d cities\n", golden.name, world.Rivers, world.Lakes,
			world.Volcanoes, world.Cities)
		shine := world.Shine()
		for _, tile := range tiles(world) {
			paint, light := world.Paint(tile), shine.Paint(tile)
			fmt.Fprintf(&lines, "%d %d %d | %s | %s\n", tile.Face, tile.I,
				tile.J, drawn(paint), drawn(light))
		}
	}
	return lines.String()
}

// the golden worlds are made as they were when the golden file was
// written: every tile's paint and light, and the counts, so a change to
// any of biome's tuned numbers shows here.
func TestTheWorldIsAsItWasDrawn(t *testing.T) {
	got := pictured(t)
	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenFile), folder); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, []byte(got), filed); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("%v; draw it once with -update", err)
	}
	gotLines, wantLines := strings.Split(got, "\n"),
		strings.Split(string(want), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("the worlds have %d lines, the golden %d", len(gotLines),
			len(wantLines))
	}
	for place := range gotLines {
		if gotLines[place] != wantLines[place] {
			t.Errorf("line %d moved: %q, not %q", place, gotLines[place],
				wantLines[place])
			return
		}
	}
}

// drawnWorld is one world of the golden file: its paints and how many
// of its tiles give off light.
type drawnWorld struct {
	paints map[string]bool
	lit    int
}

// readGolden is the golden file's worlds, in order.
func readGolden(test testing.TB) []drawnWorld {
	test.Helper()
	want, err := os.ReadFile(goldenFile)
	if err != nil {
		test.Fatal(err)
	}
	unlit := drawn(dark())
	var worlds []drawnWorld
	for _, line := range strings.Split(string(want), "\n") {
		parts := strings.Split(line, " | ")
		switch {
		case strings.HasPrefix(line, "#"):
			worlds = append(worlds, drawnWorld{paints: map[string]bool{}})
		case len(parts) == len([]string{"tile", "paint", "light"}) &&
			len(worlds) > nought:
			last := &worlds[len(worlds)-unity]
			last.paints[parts[unity]] = true
			if parts[len(parts)-unity] != unlit {
				last.lit++
			}
		}
	}
	return worlds
}

// anti-tests: the golden file is no blank: it holds every golden world,
// and each holds lit tiles and tiles of more than one paint.
func TestTheGoldenWorldIsNoBlank(t *testing.T) {
	worlds, made := readGolden(t), goldens(t)
	if len(worlds) != len(made) {
		t.Errorf("the golden file holds %d worlds, not %d", len(worlds),
			len(made))
	}
	for place, world := range worlds {
		if len(world.paints) <= unity || world.lit == nought {
			t.Errorf("golden world %d has %d paints and %d lit tiles",
				place, len(world.paints), world.lit)
		}
	}
}
