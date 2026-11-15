package walk

import (
	"fmt"
	"image"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// Lens is a rig's frame and lens: the frame's width and height and the
// focal length in millimetres, and how many metres off it is focused.
type Lens struct {
	frameWidth, frameHeight, focal, focus float64
	pinhole                               bool
}

// NewLens is a lens, refused unless the frame and focal length are more
// than nothing and the focus is further off than the focal length, as
// ninjafilm/lens.NewThin refuses them.
func NewLens(frameWidth, frameHeight, focal, focus float64) (Lens, error) {
	if !(frameWidth > 0) || !(frameHeight > 0) || !(focal > 0) ||
		!(focus > focal*millimetre) || math.IsInf(focus, 0) ||
		math.IsInf(frameWidth, 0) || math.IsInf(frameHeight, 0) {
		return Lens{}, fmt.Errorf("%w: a lens of %v by %v mm at %v mm, "+
			"focused %v m off; give a frame and focal length over nothing "+
			"and a finite focus beyond the focal length", ErrWalk,
			frameWidth, frameHeight, focal, focus)
	}
	return Lens{frameWidth: frameWidth, frameHeight: frameHeight,
		focal: focal, focus: focus}, nil
}

// image is how far behind the lens the film stands, in millimetres: the
// lens law's image of the focus, as ninjafilm/lens.NewThin puts it.
func (glass Lens) image() float64 {
	if glass.pinhole {
		return glass.focal
	}
	focus := glass.focus / millimetre
	return glass.focal * focus / (focus - glass.focal)
}

// Camera is a camera of a set made ready to project places: where its
// eye stands and how it faces, its lens, and the picture's size.
type Camera struct {
	eye           maths.Subspace
	glass         Lens
	width, height int
	// unit is what the camera's places are measured in, for its scale
	// bars; level is whether its bars lie on level ground, as a set's
	// do, or square across the picture, as a scene's do
	unit  string
	level bool
}

// NewCamera is a camera, refused for a picture under a pixel across.
// The picture's height follows from the frame's shape, rounded, as the
// tracer's rig makes it.
func NewCamera(eye maths.Subspace, glass Lens, width int) (Camera, error) {
	if width < 1 {
		return Camera{}, fmt.Errorf("%w: a picture %d across; give one "+
			"pixel or more", ErrWalk, width)
	}
	if glass.focal == 0 {
		return Camera{}, fmt.Errorf("%w: a camera with no lens; make one "+
			"with NewLens", ErrWalk)
	}
	height := int(math.Round(float64(width) * glass.frameHeight /
		glass.frameWidth))
	return Camera{eye: eye, glass: glass, width: width,
		height: max(height, 1), unit: metres, level: true}, nil
}

// metres is a set's unit. Defined: a set's places are in metres.
const metres = "m"

// Eye is where a scene's camera stands, what it looks at and which way
// is up, in the scene's own units, y up.
type Eye struct {
	from, lookAt, up maths.Vec
}

// NewEye is an eye, refused when it looks at where it stands, or its up
// is nothing or along the way it looks.
func NewEye(from, lookAt, up maths.Vec) (Eye, error) {
	ahead := lookAt.Sub(from)
	if ahead.Len() == 0 || up.Len() == 0 ||
		math.Abs(ahead.Unit().Dot(up.Unit())) > 1-roundingSlack {
		return Eye{}, fmt.Errorf("%w: an eye at %v looking at %v with up "+
			"%v; look somewhere else, with an up across the look",
			ErrWalk, from, lookAt, up)
	}
	return Eye{from: from, lookAt: lookAt, up: up}, nil
}

// PinholeCamera is a scene's camera: a pinhole at an eye seeing wide
// degrees across a picture of a size, as film.Pinhole draws, its places
// in a unit. Its pixels are square, so its height says the angle down.
func PinholeCamera(eye Eye, wide float64, size image.Point,
	unit string) (Camera, error) {
	if !(wide > 0 && wide < halfTurnDegrees) || size.X < 1 || size.Y < 1 ||
		unit == "" {
		return Camera{}, fmt.Errorf("%w: a pinhole %v degrees wide on a "+
			"picture of %v in %q; give an angle between nought and %v, a "+
			"picture, and a unit", ErrWalk, wide, size, unit,
			halfTurnDegrees)
	}
	return pinholeAt(maths.Subspace{Origin: eye.from,
		Orientation: maths.LookAtUp(eye.from, eye.lookAt, eye.up)},
		wide, size, unit), nil
}

// pinholeAt is a pinhole at an eye already faced, wide degrees across a
// picture of a size, its places in a unit: the maths PinholeCamera and
// a set's rigless film share. The angle and size are the caller's to
// check.
func pinholeAt(eye maths.Subspace, wide float64, size image.Point,
	unit string) Camera {
	across := float64(size.X)
	glass := Lens{frameWidth: across, frameHeight: float64(size.Y),
		focal:   across * half / math.Tan(wide*maths.Degree*half),
		pinhole: true}
	return Camera{eye: eye, glass: glass, width: size.X, height: size.Y,
		unit: unit}
}

// halfTurnDegrees is a half turn. Defined.
const halfTurnDegrees = 180.0

// CameraOf is a camera of a set by name, its eye where the set puts it
// among its places, through the set's rig, the picture so many pixels
// across. A film with no rig that says how wide it sees is the pinhole
// hillside draws it with (rigless). A set with neither, a camera it
// lacks, or a lens focused on a thing not among the places, is refused
// by name.
func CameraOf(read set.Set, name string, places Places, width int) (Camera,
	error) {
	film, has := read.Film()
	rig, rigged := film.Rig()
	if !has || (!rigged && !(film.Wide() > 0)) {
		return Camera{}, fmt.Errorf("%w: the set has no film with a rig "+
			"(frame and lens) or a wide; give its film one", ErrWalk)
	}
	shot, err := read.Shot(name, places.where)
	if err != nil {
		return Camera{}, fmt.Errorf("%w: %w; the set's cameras are %v",
			ErrWalk, err, read.Cameras())
	}
	if !rigged {
		return rigless(shot.Eye(), film, width)
	}
	focal, _, focus := rig.Lens()
	at, found := places.where(focus)
	if !found {
		return Camera{}, fmt.Errorf("%w: the lens is focused on %q, which "+
			"is not among the places", ErrWalk, focus)
	}
	across, down := rig.Frame()
	glass, err := NewLens(across, down, focal,
		at.Sub(shot.Eye().Origin).Len())
	if err != nil {
		return Camera{}, err
	}
	return NewCamera(shot.Eye(), glass, width)
}

// rigless is a set's camera whose film has no rig, only how wide it
// sees: the pinhole hillside draws it with (film.Pinhole, the angle
// across the picture, square pixels), the picture so many pixels across
// and as high as the film's own shape makes it, rounded, in metres on
// level ground as a set's places are.
func rigless(eye maths.Subspace, film set.Film, width int) (Camera,
	error) {
	across, down, _ := film.Size()
	if width < 1 || across < 1 || down < 1 {
		return Camera{}, fmt.Errorf("%w: a picture %d across from a film "+
			"of %d by %d; give a pixel or more and a film with a size",
			ErrWalk, width, across, down)
	}
	height := int(math.Round(float64(width) * float64(down) /
		float64(across)))
	view := pinholeAt(eye, film.Wide(), image.Pt(width, max(height, 1)),
		metres)
	view.level = true
	return view, nil
}

// Size is the picture's width and height in pixels.
func (view Camera) Size() (width, height int) {
	return view.width, view.height
}

// Eye is where the camera stands.
func (view Camera) Eye() Place { return placeOf(view.eye.Origin) }

// Pixel is where a place falls on a picture: its column and row, the
// middle of the top left pixel at nought and nought; whether the place
// is in front of the eye; and whether it falls on the picture.
type Pixel struct {
	x, y      float64
	ahead, on bool
}

// X is the pixel's column.
func (seen Pixel) X() float64 { return seen.x }

// Y is the pixel's row.
func (seen Pixel) Y() float64 { return seen.y }

// Ahead is whether the place is in front of the eye. A place behind it
// has no pixel; its X and Y are nought.
func (seen Pixel) Ahead() bool { return seen.ahead }

// On is whether the place falls on the picture.
func (seen Pixel) On() bool { return seen.on }

// Project is the pixel a place is drawn at: seen along the ray through
// the lens's middle onto the film at the image distance, as the tracer's
// thin lens sees it (ninjafilm/lens.Thin.Ray).
func (view Camera) Project(at Place) Pixel {
	return view.projected(view.eye.In(at.world()))
}

// Toward is the pixel a way in the sky is drawn at, as far off as the
// moon, at an azimuth from north toward east and an altitude over the
// level, in degrees: where the eye's own position no longer matters.
func (view Camera) Toward(azimuth, altitude float64) Pixel {
	way := skyward(azimuth, altitude)
	return view.projected(view.eye.Orientation.In(way))
}

// skyward is a way in the world's space from an azimuth and altitude in
// degrees, as hillside's skyward makes it: north is minus z, east plus x.
func skyward(azimuth, altitude float64) maths.Vec {
	round, up := azimuth*maths.Degree, altitude*maths.Degree
	return maths.V(math.Cos(up)*math.Sin(round), math.Sin(up),
		0-math.Cos(up)*math.Cos(round))
}

// projected is the pixel of a point or way in the camera's own space,
// which looks down minus z with x right and y up.
func (view Camera) projected(local maths.Vec) Pixel {
	if !(local.Z < 0) {
		return Pixel{}
	}
	image := view.glass.image()
	across := image * local.X / (0 - local.Z) / view.glass.frameWidth
	up := image * local.Y / (0 - local.Z) / view.glass.frameHeight
	x := (across+half)*float64(view.width) - half
	y := (half-up)*float64(view.height) - half
	on := x >= 0-half && y >= 0-half && x < float64(view.width)-half &&
		y < float64(view.height)-half
	return Pixel{x: x, y: y, ahead: true, on: on}
}
