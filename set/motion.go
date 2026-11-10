package set

import (
	"fmt"
	"slices"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	yaml "gopkg.in/yaml.v3"
)

// A Motion is a run through keyframes that carries a thing: how many
// seconds the run takes, the ease it runs by, whether it holds its last
// place after, and its frames, each a percentage of the run and a place.
type Motion struct {
	name, ease string
	seconds    float64
	hold       bool
	frames     []frame
}

// a frame is a place at a percentage of a run.
type frame struct {
	percent float64
	at      [3]float64
}

// A Frame is a place at a share of a run.
type Frame struct {
	Share float64
	At    maths.Vec
}

// Name is the motion's name, which a sphere uses to be carried by it.
func (made Motion) Name() string { return made.name }

// Run is how long the run is, the ease it runs by, and whether it holds
// its last place after.
func (made Motion) Run() (length time.Duration, ease string, hold bool) {
	return time.Duration(made.seconds * float64(time.Second)), made.ease,
		made.hold
}

// Frames is the run's frames, in order, each a share of the run from
// nought to one and a place in the world's space.
func (made Motion) Frames() []Frame {
	frames := make([]Frame, len(made.frames))
	for index, each := range made.frames {
		frames[index] = Frame{Share: each.percent / 100,
			At: fromMap(each.at)}
	}
	return frames
}

// the longest a run may be, in seconds: 292 years, a little under the
// most a time.Duration holds, so its nanoseconds never overflow one
const longest = 292 * 365.25 * 24 * 3600

// motions reads each motion, by its name.
func (reading reader) motions(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Motion{name: name.Value}
		err := reading.keys(block, map[string]step{
			"seconds": reading.number(&made.seconds, 1/float64(time.Second),
				longest),
			"ease":   reading.word(&made.ease, thing.Eases...),
			"hold":   reading.yes(&made.hold),
			"frames": reading.framed(&made.frames),
		}, nil)
		if err == nil {
			err = reading.all(block, "a motion", "seconds", "ease", "hold",
				"frames")
		}
		reading.set.world.motions = append(reading.set.world.motions, made)
		return err
	})
}

// framed is a step that reads a run's frames: a list of at least one,
// each its percent of the run and its place, the percents never going
// back.
func (reading reader) framed(into *[]frame) step {
	return func(name, value *yaml.Node) error {
		if value.Kind != yaml.SequenceNode || len(value.Content) == 0 {
			return reading.refuse(value, "%s takes a list of frames, each "+
				"{percent: P, at: [x, y, z]}", name.Value)
		}
		for _, item := range value.Content {
			var made frame
			err := reading.keys(item, map[string]step{
				"percent": reading.number(&made.percent, 0, 100),
				"at":      reading.three(&made.at, -farthest, farthest),
			}, nil)
			if err == nil {
				err = reading.all(item, "a frame", "percent", "at")
			}
			if err == nil && len(*into) > 0 &&
				made.percent < (*into)[len(*into)-1].percent {
				err = reading.refuse(item, "a frame at %v%% after one at "+
					"%v%%: frames go in order", made.percent,
					(*into)[len(*into)-1].percent)
			}
			if err != nil {
				return err
			}
			*into = append(*into, made)
		}
		return nil
	}
}

// carried refuses a sphere made of a material the world does not hold,
// or carried by a motion it does not hold.
func (reading reader) carried(world *World) error {
	for _, each := range world.spheres {
		named := func(held Material) bool {
			return held.name == each.material
		}
		if !slices.ContainsFunc(world.materials, named) {
			return fmt.Errorf("%s: sphere %s is made of %q, and the world "+
				"holds no material by that name", reading.file, each.name,
				each.material)
		}
		moved := func(held Motion) bool { return held.name == each.motion }
		if each.motion != "" && !slices.ContainsFunc(world.motions, moved) {
			return fmt.Errorf("%s: sphere %s is carried by %q, and the "+
				"world holds no motion by that name", reading.file,
				each.name, each.motion)
		}
	}
	return nil
}
