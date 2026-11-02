package set

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// Shot is one camera made ready to render: its name and title, where its
// eye stands and which way it faces, and the lens it sees through.
type Shot struct {
	name, title string
	eye         maths.Subspace
	lens        film.Pinhole
}

// Name is the camera's key in the set file.
func (shot Shot) Name() string { return shot.name }

// Title is what the set file calls the camera for people.
func (shot Shot) Title() string { return shot.title }

// Eye is where the camera stands and which way it faces, in the world's space.
func (shot Shot) Eye() maths.Subspace { return shot.eye }

// Lens is what the camera sees through. Every camera has a film.Pinhole
// for now, since the set file does not say a lens yet.
func (shot Shot) Lens() film.Pinhole { return shot.lens }

// Where says where an asset stands, in the world's space. The set file names
// its assets but does not place them yet, so whoever builds the scene
// answers this, and false means there is no such asset.
type Where func(asset string) (maths.Vec, bool)

// Shots is every camera the set asks for, made ready to render, in the
// order the set file gives them.
func (s Set) Shots(where Where) ([]Shot, error) {
	shots := make([]Shot, 0, len(s.cameras))
	for _, each := range s.cameras {
		shot, err := each.shot(where)
		if err != nil {
			return nil, fmt.Errorf("camera %s: %w", each.name, err)
		}
		shots = append(shots, shot)
	}
	return shots, nil
}

// Cameras is every camera's name, in the order the set file gives them.
func (s Set) Cameras() []string {
	names := make([]string, 0, len(s.cameras))
	for _, each := range s.cameras {
		names = append(names, each.name)
	}
	return names
}

// Shot is the camera of a name, made ready to render. It refuses a name
// the set has no camera by.
func (s Set) Shot(name string, where Where) (Shot, error) {
	for _, each := range s.cameras {
		if each.name == name {
			shot, err := each.shot(where)
			if err != nil {
				return Shot{}, fmt.Errorf("camera %s: %w", name, err)
			}
			return shot, nil
		}
	}
	return Shot{}, fmt.Errorf("the set has no camera %s", name)
}

// shot stands the camera off the asset it is placed from and faces it
// toward the asset it points to, tipped up from level; or stands it at
// its place, facing the place it looks at.
func (shooting camera) shot(where Where) (Shot, error) {
	if shooting.at != nil {
		return shooting.standing()
	}
	base, found := where(shooting.from)
	if !found {
		return Shot{}, fmt.Errorf(
			"stands off %q, which is not on the set", shooting.from)
	}
	target, found := where(shooting.pointTo)
	if !found {
		return Shot{}, fmt.Errorf("points to %q, which is not on "+
			"the set", shooting.pointTo)
	}
	eye := base.Add(fromMap(shooting.standoff))
	ahead, heading, err := facing(eye, target, shooting.alpha)
	if err != nil {
		return Shot{}, err
	}
	return Shot{
		name:  shooting.name,
		title: shooting.title,
		eye: maths.Subspace{
			Origin: eye,
			Orientation: maths.LookAtUp(eye, eye.Add(ahead),
				upFor(ahead, heading)),
		},
	}, nil
}

// standing stands the camera at its place, facing the place it looks at.
// Looking straight up or down, or at where it stands, gives no heading.
func (shooting camera) standing() (Shot, error) {
	eye, look := fromMap(*shooting.at), fromMap(*shooting.lookAt)
	ahead := look.Sub(eye)
	across := ahead
	across.Y = 0
	if across.Len() == 0 {
		return Shot{}, fmt.Errorf("looks straight up or down, or at " +
			"where it stands, so it has no heading")
	}
	return Shot{
		name:  shooting.name,
		title: shooting.title,
		eye: maths.Subspace{
			Origin: eye,
			Orientation: maths.LookAtUp(eye, look,
				upFor(ahead, across.Unit())),
		},
	}, nil
}

// fromMap turns a set file's metres, which run as a map's do (x east, y
// north, z up), into the world's space, which is y up and right handed, so
// north is its minus z.
//
// north is taken from nought rather than turned about, so a file's
// nought is the world's nought and never minus nought, which an angle
// worked from it could tell apart.
func fromMap(metres [3]float64) maths.Vec {
	return maths.V(metres[0], metres[2], 0-metres[1])
}

// facing is the way an eye at from looks, ahead: toward to across the
// ground, its heading, then tipped up from level by alpha degrees.
// Standing straight above or below what it points to gives no heading.
func facing(from, to maths.Vec, alpha float64) (ahead, heading maths.Vec,
	err error) {
	across := to.Sub(from)
	across.Y = 0
	if across.Len() == 0 {
		return maths.Vec{}, maths.Vec{}, fmt.Errorf(
			"stands straight above or below what it points to, " +
				"so it has no heading")
	}
	heading = across.Unit()
	tip := alpha * math.Pi / 180
	ahead = heading.Scale(math.Cos(tip)).Add(maths.V(0, math.Sin(tip), 0))
	return ahead, heading, nil
}

// upFor is which way is up for an eye facing ahead: the world's up, except
// for an eye looking straight up or down, whose up is its heading.
func upFor(ahead, heading maths.Vec) maths.Vec {
	up := maths.V(0, 1, 0)
	if math.Abs(ahead.Unit().Dot(up)) > 1-1e-9 {
		return heading
	}
	return up
}
