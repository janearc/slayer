package job

import (
	"errors"
	"strconv"

	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/maths"
)

// Camera is the job's camera: one the scene defines, by name, or a path
// of moves in scene metres, and either way perhaps a zoom. The path and
// the zoom are checked when the job is read and cannot change after.
type Camera struct {
	name string
	unit string
	path *camera.Path
	zoom *camera.Zoom
}

// Name is the scene's camera the job names, and whether it names one.
func (lens Camera) Name() (string, bool) { return lens.name, lens.name != "" }

// Unit is the unit of a path's points, as the job writes it, and
// whether it says one. The renderer holds it to the scene's own.
func (lens Camera) Unit() (string, bool) { return lens.unit, lens.unit != "" }

// Path is the camera's path of moves, and whether the job says one. It
// is sampled from elapsed scene time since the film's start.
func (lens Camera) Path() (*camera.Path, bool) {
	return lens.path, lens.path != nil
}

// Zoom is the change of the lens's angle across the picture, and whether
// the job says one. It is sampled as the path is.
func (lens Camera) Zoom() (*camera.Zoom, bool) {
	return lens.zoom, lens.zoom != nil
}

// errSmoothSpeed is a move eased smooth at a constant speed, which
// cannot be both.
var errSmoothSpeed = errors.New("camera: smooth easing needs duration; " +
	"speed means constant speed")

// lens is the camera the wire says, its path and zoom built and checked,
// and where in the document it went wrong when it did.
func (wire *cameraWire) lens() (Camera, []string, error) {
	path, where, err := wire.path()
	if err != nil {
		return Camera{}, where, err
	}
	zoom, err := wire.zoomPath()
	if err != nil {
		return Camera{}, []string{"camera", "zoom"}, err
	}
	return Camera{name: wire.Name, unit: wire.Unit, path: path,
		zoom: zoom}, nil, nil
}

// zoomPath is the zoom, or nil where the camera says none and the
// scene's own lens is kept.
func (wire *cameraWire) zoomPath() (*camera.Zoom, error) {
	if wire.Zoom == nil {
		return nil, nil
	}
	zoom := wire.Zoom
	return camera.NewZoom(zoom.From.Value, zoom.To.Value,
		zoom.Duration.duration(), zoom.Ease)
}

// path is the camera's moves as a checked path, or nil for a camera
// named and not moved; where says which move went wrong.
func (wire *cameraWire) path() (*camera.Path, []string, error) {
	if wire.Moves == nil {
		return nil, nil, nil
	}
	moves := make([]camera.Move, len(wire.Moves))
	for index, in := range wire.Moves {
		move, err := in.move()
		if err != nil {
			where := []string{"camera", "moves", strconv.Itoa(index)}
			return nil, where, err
		}
		moves[index] = move
	}
	up := maths.Up
	if wire.Up != nil {
		up = wire.Up.vec()
	}
	path, err := camera.New(moves, wire.LookAt.vec(), up)
	return path, []string{"camera", "moves"}, err
}

// move is one move, at its speed or over its duration, with its own look
// where it says one.
func (in moveWire) move() (camera.Move, error) {
	var move camera.Move
	var err error
	if in.Speed != nil {
		if in.Ease == "smooth" {
			return move, errSmoothSpeed
		}
		move, err = camera.Travel(in.From.vec(), in.To.vec(),
			in.Speed.Value)
	} else {
		move, err = camera.Timed(in.From.vec(), in.To.vec(),
			in.Duration.duration(), in.Ease)
	}
	if err == nil && in.Look != nil {
		move, err = move.Looking(in.Look.From.vec(), in.Look.To.vec())
	}
	return move, err
}
