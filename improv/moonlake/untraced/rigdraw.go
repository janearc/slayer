package main

import (
	"image"
	"image/color"
	"math"

	"github.com/janearc/slayer/improv/moonlake/rig"
	"github.com/janearc/slayer/improv/walk"
)

// view is a drawn pane: a camera's still, the part of it the pane shows
// (left, top, width and height in the still's pixels), and the camera,
// so a place of the show lands on the pane where the camera sees it.
type view struct {
	camera walk.Camera
	still  *image.RGBA
	part   image.Rectangle
	grade  func(color.RGBA) color.RGBA
	// moves are the window's travels over the song
	moves []move
}

// eye is the show's eye on the map: the west rise, 20 m up, 71.7 m over
// the lake. From sets/moonlake.yaml and places (see the README).
var eye = [3]float64{263104.5, 2148423.5, 1147.7}

// placeOf is a rig point, metres from the show's eye, as a place.
func placeOf(at rig.Point) walk.Place {
	place, _ := walk.NewPlace(eye[0]+at[0], eye[1]+at[1], eye[2]+at[2])
	return place
}

// onPane is where a rig point lands on a pane of a size, and whether it
// is in front of the camera.
func (pane view) onPane(at rig.Point, size image.Point) (float64, float64,
	bool) {
	seen := pane.camera.Project(placeOf(at))
	scale := float64(size.X) / float64(pane.part.Dx())
	return (seen.X()+0.5-float64(pane.part.Min.X))*scale - 0.5,
		(seen.Y()+0.5-float64(pane.part.Min.Y))*scale - 0.5, seen.Ahead()
}

// picture is the pane's part of its still at a size.
func (pane view) picture(size image.Point) *image.RGBA {
	made := image.NewRGBA(image.Rectangle{Max: size})
	scale := float64(pane.part.Dx()) / float64(size.X)
	for y := range size.Y {
		for x := range size.X {
			sx := float64(pane.part.Min.X) + (float64(x)+0.5)*scale
			sy := float64(pane.part.Min.Y) + (float64(y)+0.5)*scale
			made.SetRGBA(x, y, pane.grade(pane.still.RGBAAt(int(sx),
				int(sy))))
		}
	}
	return made
}

// glow adds light to a pixel, the colour's channels times a share.
func glow(picture *image.RGBA, x, y int, colour [3]float64, share float64) {
	if !(image.Point{x, y}).In(picture.Rect) {
		return
	}
	was := picture.RGBAAt(x, y)
	add := func(channel uint8, light float64) uint8 {
		return uint8(math.Min(math.MaxUint8, float64(channel)+
			math.MaxUint8*light*share))
	}
	picture.SetRGBA(x, y, color.RGBA{add(was.R, colour[0]),
		add(was.G, colour[1]), add(was.B, colour[2]), math.MaxUint8})
}

// spot adds a soft disc of light, brightest at its middle.
func spot(picture *image.RGBA, x, y, radius float64, colour [3]float64,
	share float64) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			fall := 1 - math.Hypot(dx, dy)/radius
			if fall > 0 {
				glow(picture, int(x+dx), int(y+dy), colour, share*fall)
			}
		}
	}
}

// stroke adds a line of light between two points of a picture.
func stroke(picture *image.RGBA, x0, y0, x1, y1 float64,
	colour [3]float64, share float64) {
	steps := int(math.Ceil(math.Hypot(x1-x0, y1-y0)))
	for k := 0; k <= steps; k++ {
		f := float64(k) / float64(max(steps, 1))
		glow(picture, int(x0+(x1-x0)*f), int(y0+(y1-y0)*f), colour, share)
	}
}

// ray draws the show's light from one rig point to another on a pane,
// as a line through its points in front of the camera.
func (pane view) ray(picture *image.RGBA, from, to rig.Point,
	colour [3]float64, share float64) {
	size := picture.Rect.Size()
	lastX, lastY, lastAhead := pane.onPane(from, size)
	for k := 1; k <= raySteps; k++ {
		at := from.Add(to.Add(from.Scale(-1)).Scale(float64(k) / raySteps))
		x, y, ahead := pane.onPane(at, size)
		if ahead && lastAhead {
			stroke(picture, lastX, lastY, x, y, colour, share)
		}
		lastX, lastY, lastAhead = x, y, ahead
	}
}

// shape is how a kind of fixture is marked: so many corners (a circle
// for none), turned so many degrees.
type shape struct {
	corners int
	turn    float64
}

// mark outlines a fixture's place in its kind's shape and colour.
func (pane view) mark(picture *image.RGBA, at rig.Point, outline shape,
	colour [3]float64) {
	x, y, ahead := pane.onPane(at, picture.Rect.Size())
	if !ahead {
		return
	}
	corners := outline.corners
	if corners == 0 {
		corners = markSteps
	}
	for k := range corners {
		one := (outline.turn + 360*float64(k)/float64(corners)) * math.Pi /
			180
		other := (outline.turn + 360*float64(k+1)/float64(corners)) *
			math.Pi / 180
		stroke(picture, x+markRadius*math.Cos(one), y+markRadius*
			math.Sin(one), x+markRadius*math.Cos(other),
			y+markRadius*math.Sin(other), colour, 1)
	}
}

// blueprint is a still graded down hard toward a dark blue-grey, so
// drawn light reads on it: its brightness to a share of a tint over a
// dark ground; with peaked, the middle greys brightest and the
// brightest (a white sky) dark again.
func blueprint(peaked bool) func(color.RGBA) color.RGBA {
	return func(was color.RGBA) color.RGBA {
		light := (0.2126*float64(was.R) + 0.7152*float64(was.G) +
			0.0722*float64(was.B)) / math.MaxUint8
		share := light * plainLift
		if peaked {
			share = 4 * light * (1 - light)
		}
		share = math.Min(1, share)
		tone := func(base, tint float64) uint8 {
			return uint8(math.Min(math.MaxUint8, base+share*tint))
		}
		return color.RGBA{tone(printBase[0], printTint[0]),
			tone(printBase[1], printTint[1]),
			tone(printBase[2], printTint[2]), math.MaxUint8}
	}
}

// The blueprint grade: a dark ground and the tint the land's light is
// laid over it in, as 0 to 255, and how far a plain still's brightness
// is lifted first. Picked by the untraced film's author, 2026-11-12.
var (
	printBase = [3]float64{8, 12, 26}
	printTint = [3]float64{62, 78, 104}
)

// plainLift is how far a plain still's brightness is lifted.
const plainLift = 1.3
