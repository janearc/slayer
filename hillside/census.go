package hillside

// The census: a frame's world counted as it was staged, with the camera
// the frame would be drawn from, and nothing drawn. The world comes from
// the same inspection as Inspect, and the camera from the same steps a
// frame takes before it draws (eyed), so the numbers are the ones a
// draw of that frame traces.

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The lenses a census says a camera looks through.
const (
	lensPinhole = "pinhole"
	lensJob     = "the job's lens"
	lensRig     = "the set's rig"
)

// Census is a frame's world counted as staged, at its scene time, and
// the camera it would be drawn from; where the scene cannot say its
// camera without drawing, Unseen says so and Eye is nil.
type Census struct {
	At     time.Duration
	World  thing.Census
	Eye    *Eye
	Unseen string
}

// Eye is a frame's camera as staged: where it stands, the way it looks
// and which way is up, in the scene's own units, the lens it looks
// through, and the angle across the frame in degrees, which a set's rig
// says for itself and is nought for one.
type Eye struct {
	Position, Look, Up maths.Vec
	Lens               string
	Wide               float64
}

// upward is up in an eye's own space: y is up. Defined by the scene's
// coordinates (OPERATION.md, building a scene).
var upward = maths.V(0, 1, 0)

// eyeInspector is a source that says a frame's camera as the frame takes
// it before drawing: the steps its light takes first.
type eyeInspector interface {
	eyed(at time.Duration, given asked) (asked, error)
}

// unseen is why a source does not say its camera: a retained film's
// and a rig's eye is worked out as each frame is drawn.
const unseen = "this scene's camera is worked out as its frames are " +
	"drawn (a retained film, or a set's rig), and the census draws none"

// Census counts the world for frame index as staged, at the job's exact
// time, and says its camera, drawing nothing.
func (session *Session) Census(index int) (Census, error) {
	world, err := session.inspected(index)
	if err != nil {
		return Census{}, err
	}
	counted, err := thing.Count(world)
	if err != nil {
		return Census{}, err
	}
	made := Census{At: session.times[index], World: counted}
	source, says := session.source.(eyeInspector)
	if !says {
		made.Unseen = unseen
		return made, nil
	}
	given, err := session.asking(Ask{})
	if err != nil {
		return Census{}, err
	}
	if given, err = source.eyed(made.At, given); err != nil {
		return Census{}, err
	}
	made.Eye = eyeOf(given)
	return made, nil
}

// eyeOf is the camera of what a frame was asked, as staged; nil where
// it stages no eye.
func eyeOf(given asked) *Eye {
	if given.eye == nil {
		return nil
	}
	eye := *given.eye
	made := Eye{Position: eye.Origin, Look: eye.Forward(),
		Up: eye.Orientation.Out(upward), Lens: lensPinhole,
		Wide: given.wide}
	switch {
	case given.frame != nil:
		made.Lens, made.Wide = lensRig, nothing
	case given.optics.aperture > nothing:
		made.Lens = lensJob
	}
	return &made
}

// inspected is the world for frame index as Inspect samples it: refused
// when the session is closed, the frame is not the job's, or the source
// gives no world.
func (session *Session) inspected(index int) (*thing.World, error) {
	if session.closed {
		return nil, errClosed
	}
	if index < nothing || index >= len(session.times) {
		return nil, fmt.Errorf("hillside: no frame %d; the job has %d, "+
			"counted from zero", index, len(session.times))
	}
	source, ok := session.source.(sceneInspector)
	if !ok {
		return nil, fmt.Errorf("hillside: %T does not support scene "+
			"inspection", session.source)
	}
	world, err := source.inspect(session.times[index])
	if err != nil {
		return nil, err
	}
	if world == nil {
		return nil, fmt.Errorf("hillside: %T returned no world for "+
			"inspection", session.source)
	}
	return world, nil
}
