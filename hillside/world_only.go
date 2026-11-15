package hillside

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// nothing is no amount at all.
const nothing = 0 // defined: no amount at all

// unshifted is no shift: the zero place.
var unshifted maths.Vec

// worldOnly is whether a set is drawn from its world half alone: it says
// a world, and no terrain, planet, things or stands for hillside to stand
// anything on.
func worldOnly(read set.Set) bool {
	_, hasWorld := read.World()
	_, grounded := read.Ground()
	_, planeted := read.Planet()
	return hasWorld && !grounded && !planeted &&
		len(read.Things()) == nothing && len(read.Stands()) == nothing
}

// saysFilm is whether a film says anything: a size, a lens, an exposure,
// a print or a rig. a film that says none is no film, and a set drawn from
// its world half has to say one; a size it does not say is the one the
// command's -w, -h and -n say.
func saysFilm(film set.Film) bool {
	width, _, _ := film.Size()
	_, _, printed := film.Print()
	_, rigged := film.Rig()
	return width > nothing || positive(film.Wide()) ||
		positive(film.Exposure()) || printed || rigged
}

// positive is whether a number is more than nought.
func positive(x float64) bool { return x > nothing }

// nowhere answers a camera that stands off an asset: a set drawn from its
// world half places none of its assets, so such a camera is refused by
// name.
func nowhere(string) (maths.Vec, bool) { return unshifted, false }

// drawing is what staging a set drawn from its world half came to: the
// world, its tally and what was asked with the set in it, or why not.
type drawing struct {
	world *thing.World
	made  tally
	given asked
	err   error
}

// results are the drawing as staged gives them.
func (done drawing) results() (*thing.World, tally, asked, error) {
	return done.world, done.made, done.given, done.err
}

// refused is a drawing that failed, with what was asked as it was.
func refused(given asked, err error) drawing {
	return drawing{given: given, err: err}
}

// stagedWorld is a set drawn from its world half alone, staged as stage.World
// stages it: the set's own lamps and sky, and no light of hillside's.
// Its eye is the first camera's and its lens the film's. A film that
// says a print is developed by it, with the film's exposure; the lens is
// the film's width, which a film that says none leaves as the pinhole's
// own.
func stagedWorld(read set.Set, given asked) drawing {
	said, _ := read.World()
	world, err := stage.World(said)
	if err != nil {
		return refused(given, fmt.Errorf("%s: %w", given.set, err))
	}
	world.VolumeIllumination = read.VolumeIllumination()
	eye, err := cameraEye(read, given.camera)
	if err != nil {
		return refused(given, fmt.Errorf("%s: %w", given.set, err))
	}
	given.eye = eye
	film, _ := read.Film()
	if !given.said.wide {
		given.wide = film.Wide()
	}
	given.print = filmPrint(film)
	if given.said.exposure {
		// the job's exposure develops the picture, through the set's
		// print or hillside's own: a world of a set's pieces has no
		// lamps of hillside's for it to scale
		tone := given.tone()
		tone.Exposure = given.exposure
		given.print = &tone
	}
	making := &staging{given: given, places: map[string]maths.Vec{}}
	if given.overlay, err = making.overlaid(read); err != nil {
		return refused(given, err)
	}
	return drawing{world: world, made: talliedWorld(world), given: given}
}

// cameraEye is where the set's camera of a name stands and faces, or
// its first camera where the name is nothing, or nil and why not.
func cameraEye(read set.Set, name string) (*maths.Subspace, error) {
	if name == "" {
		return firstEye(read)
	}
	shot, err := namedShot(read, name, nowhere)
	if err != nil {
		return nil, err
	}
	eye := shot.Eye()
	return &eye, nil
}

// firstEye is where the set's first camera stands and faces, or nil and
// why not.
func firstEye(read set.Set) (*maths.Subspace, error) {
	for _, name := range read.Cameras() {
		shot, err := read.Shot(name, nowhere)
		if err != nil {
			return nil, err
		}
		eye := shot.Eye()
		return &eye, nil
	}
	return nil, errors.New("no camera")
}

// filmPrint is the print a film says, with its exposure, or nil where it
// says none.
func filmPrint(film set.Film) *colour.Tone {
	if _, _, says := film.Print(); !says {
		return nil
	}
	tone := stage.Print(film)
	return &tone
}

// talliedWorld is what a world of a set's pieces came to: its things and
// the faces of their shapes, and its lamps.
func talliedWorld(world *thing.World) tally {
	made := tally{things: len(world.Things), lamps: len(world.Lamps)}
	for _, each := range world.Things {
		made.faces += facesIn(each.Shape)
	}
	made.kept = made.faces
	return made
}

// tone is how the picture is developed: through the print its film says,
// where it is a set drawn from its world half and says one, and else
// through hillside's own.
func (given asked) tone() colour.Tone {
	if given.print != nil {
		return *given.print
	}
	return developing
}
