package main

import (
	"fmt"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"

	"github.com/janearc/slayer/improv/quasar/diagram"
	"github.com/janearc/slayer/improv/quasar/rings"
)

// The smoke's own geometry, as rings takes it: defined by smoke 36
// (scene/smoke/smoke.go, thirtySix) and the quasar template.
const (
	smokeEye   = 50            // the eye, in masses from the centre
	smokeLean  = 10 * degree   // the eye above the disc's plane
	smokeInner = 6             // three schwarzschild radii, in masses
	smokeOuter = 60            // thirty, in masses
	orders     = 3             // the direct image, the second, the third
	diagramOut = 12            // masses from the centre to the edge
	halfway    = 0.5           // defined: a share of a whole
	opaque     = math.MaxUint8 // defined: a colour all there
	closeInPx  = 12            // the third image's width in the close-in: picked
	aimShare   = halfway       // where in the third image the close-in aims
	oneRay     = 1             // defined
	tried      = 200           // rays tried across a band: picked
	landAt     = 8             // masses out the diagram's rays land: picked
)

// The rays' colours in the diagram: the quasar slice, 2026-11-12.
var rayColours = []color.RGBA{
	{R: 120, G: 220, B: 255, A: opaque},
	{R: 255, G: 230, B: 90, A: opaque},
	{R: 255, G: 110, B: 200, A: opaque},
}

// measured is the smoke's lens and disc, and the bands of its images.
type measured struct {
	lens  rings.Lens
	disc  rings.Disc
	bands []rings.Band
}

// measure finds the bands of smoke 36's first three disc images.
func measure() (measured, error) {
	lens, err := rings.NewLens(smokeEye, smokeLean)
	if err != nil {
		return measured{}, err
	}
	disc, err := rings.NewDisc(smokeInner, smokeOuter)
	if err != nil {
		return measured{}, err
	}
	bands, err := lens.Bands(disc, orders)
	if err != nil {
		return measured{}, err
	}
	return measured{lens: lens, disc: disc, bands: bands}, nil
}

// report says, for a camera at the smoke's eye, where each band lies in
// rows above the picture's middle and how wide it is, and the close-in
// that shows the third image so many pixels wide: its aim and width.
func report(out io.Writer, given asked, wide float64) error {
	found, err := measure()
	if err != nil {
		return err
	}
	camera, err := rings.NewCamera(wide, given.width)
	if err != nil {
		return err
	}
	edge := found.lens.Edge()
	fmt.Fprintf(out, "edge %.9f rad, %.2f px above the middle\n", edge,
		camera.Offset(edge))
	for _, band := range found.bands {
		fmt.Fprintf(out, "order %d: %.9f to %.9f rad, rows %.2f to "+
			"%.2f above the middle, %.3f px wide\n", band.Order(),
			band.Near(), band.Far(), camera.Offset(band.Near()),
			camera.Offset(band.Far()), camera.Pixels(band))
	}
	third := found.bands[orders-oneRay]
	closer, err := camera.CloseIn(third, closeInPx)
	if err != nil {
		return err
	}
	aim := third.Near() + aimShare*third.Width()
	fmt.Fprintf(out, "close-in: aim %.9f rad, wide %.6f degrees\n", aim,
		closer.Wide()/degree)
	return nil
}

// landingNear is the ray in a band whose landing is nearest the
// diagram's chosen radius, of so many tried across the band, so every
// ray's landing is in the picture.
func landingNear(lens rings.Lens, band rings.Band) rings.Path {
	best, bestMiss := rings.Path{}, math.Inf(oneRay)
	for each := range tried {
		share := (float64(each) + halfway) / tried
		path := lens.Follow(band.Near() + share*band.Width())
		crossings := path.Crossings()
		if band.Order() >= len(crossings) {
			continue
		}
		miss := math.Abs(crossings[band.Order()].Radius - landAt)
		if miss < bestMiss {
			best, bestMiss = path, miss
		}
	}
	return best
}

// drawDiagram draws the top-down picture: a ray from the middle of each
// band, each drawn to where it lands.
func drawDiagram(given asked) error {
	found, err := measure()
	if err != nil {
		return err
	}
	view, err := diagram.NewView(given.width, diagramOut)
	if err != nil {
		return err
	}
	rays := make([]diagram.Ray, len(found.bands))
	for each, band := range found.bands {
		rays[each] = diagram.Ray{Path: landingNear(found.lens, band),
			Order: band.Order(), Colour: rayColours[each]}
	}
	file, err := os.Create(given.out)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := png.Encode(file, view.Draw(found.disc, rays)); err != nil {
		return err
	}
	return file.Close()
}
