//go:build draw && darwin

package hillside

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/planet"
)

// the camera of Melancholia's giant alone: 142,000 km from its middle as the
// set's ground stands from it, through the lens the set's rig has, 55
// millimetres on a frame 70.41 across, which sees 64.9 degrees.
const (
	melancholiaRange = 1.42e8
	melancholiaView  = 64.9 * maths.Degree
)

// aloneHeader is the scene of a set file of the giant alone, before its planet
// block: the sun, the film, and the camera that stands off in space.
const aloneHeader = `scene:
  name: melancholia's giant alone
  light: sun
  sun:
    azimuth: 135
    altitude: 20
  film:
    width: 240
    rays: 1
    frame:
      width: 70.41
      height: 51.61
    lens:
      focal: 55
      stop: 8
      focus: the giant
    shutter: 0.004
    glimpses: 1
  cameras:
    afar:
      title: "the giant, alone"
      orientation:
        point_to: the giant
        alpha: 0
      location:
        loc_relative:
          object: the giant
          standoff_x: 0
          standoff_y: -1.42e8
          standoff_z: 0
`

// planetBlock is the lines of the planet block of the set the text is, less
// its sky, and with the pole tipped to open its rings 20 degrees.
func planetBlock(t *testing.T, text []byte) []string {
	t.Helper()
	lines := strings.Split(string(text), "\n")
	first, last := -1, len(lines)
	for at, line := range lines {
		switch {
		case line == "  planet:":
			first = at
		case line == "  set:" && first >= 0:
			last = at
		}
	}
	if first < 0 {
		t.Fatal("sets/melancholia.yaml has no planet block")
	}
	var block []string
	for _, line := range lines[first:last] {
		switch {
		case strings.HasPrefix(line, "    sky:"):
		case strings.HasPrefix(line, "    pole:"):
			block = append(block, "    pole: {azimuth: 180, altitude: 70}")
		default:
			block = append(block, line)
		}
	}
	return block
}

// melancholiaAlone is a set file of the giant of sets/melancholia.yaml with
// nothing else: its planet block as the set has it, less its place in the
// sky, with the pole tipped to open its rings 20 degrees to a camera that
// stands off in space, as the set says they open to the one on the ground.
// it is written in the test's own directory beside a copy of the sources
// file, and its path returned.
func melancholiaAlone(t *testing.T) string {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("..", "sets",
		"melancholia.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	block := planetBlock(t, text)
	directory := t.TempDir()
	sources, err := os.ReadFile(filepath.Join("..", "sources.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "sources.yaml"), sources,
		0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "melancholia-alone.yaml")
	body := aloneHeader + strings.Join(block, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Melancholia's giant alone, in space, drawn at 160 by 90 on the tracer and
// on the gpu: its seven rings with ringlets, grains in four patterns,
// moonlets and a glow. The test records whether the gpu draws it, or the
// name of what it refuses, and how far it is from the tracer and how long
// each takes; it fails only if the gpu draws a pixel that is not finite.
func TestTheGPUDrawsMelancholiasPlanetAlone(t *testing.T) {
	given := mustParse(t)
	given.set = melancholiaAlone(t)
	world, _, given, err := staged(given)
	if err != nil {
		t.Skipf("the set cannot be staged here: %v", err)
	}
	if given.eye == nil {
		t.Fatal("the set has no camera")
	}
	rings := 0
	for _, each := range world.Volumes {
		if _, ok := each.Medium.(planet.RingSlab); ok {
			rings++
		}
	}
	eye := *given.eye
	found := ringsBoth(t, world, eye, film.Pinhole{Wide: melancholiaView})
	t.Logf("things %d, volumes %d, ring slabs %d, lamps %d",
		len(world.Things), len(world.Volumes), rings, len(world.Lamps))
	if found.refused != nil {
		t.Logf("the gpu refuses it: %v; the tracer took %v", found.refused,
			found.cpu)
		return
	}
	t.Logf("drawn: mean %g, largest %g, not finite %d", found.apart.mean,
		found.apart.largest, found.apart.notFinite)
	t.Logf("cpu %v, gpu %v, gpu again %v", found.cpu, found.gpu,
		found.gpuAgain)
	if found.apart.notFinite > 0 {
		t.Errorf("%d pixels of the gpu's picture are not finite",
			found.apart.notFinite)
	}
}
