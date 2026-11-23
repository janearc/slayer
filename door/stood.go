package door

// The places table: draw --places PATH and assess FILE --places PATH
// write, from the scene's staging, where each thing of a set was stood,
// as the session writes it (Session.Places): a table of commas of what
// it is of, its kind, east, north and height. It is not the census,
// which counts geometry.

import (
	"errors"
	"fmt"

	"github.com/janearc/puzzlemouth/pzm"
)

// placesFlag names the file draw and assess write the places table to.
const placesFlag = "places"

// placer is a source that says where each thing of its staged scene was
// stood, and writes it: a session.
type placer interface {
	Places(path string) error
}

// errUnplaced is the refusal of a places table of a scene with nothing
// stood on a ground.
var errUnplaced = errors.New("this scene stands nothing on a ground, so " +
	"it has no places to write: frames already rendered, a scene of " +
	"maps and a set of a world and no ground have none")

// Places is the session's table of where things were stood, written to
// path.
func (made sessionSource) Places(path string) error {
	return made.session.Places(path)
}

// withPlaces is the run with the places table the call asks for, where
// it asks for one, a relative path under directory; a scene of maps,
// which stages no things, is refused.
func (run runner) withPlaces(call pzm.Call, directory string) (runner,
	error) {
	if !call.Given(placesFlag) {
		return run, nil
	}
	if _, isMaps := mapsOf(run.wanted); isMaps {
		return runner{}, errUnplaced
	}
	run.at.stood = resolved(call.Value(placesFlag), directory)
	return run, nil
}

// placed writes the places table of an opened source where the run
// asks for one, and says so; it asks for none and writes none where the
// places say no file. A source that cannot say its places is refused.
func (run runner) placed(opened source) error {
	if run.at.stood == "" {
		return nil
	}
	placing, places := opened.(placer)
	if !places {
		return errUnplaced
	}
	if err := placing.Places(run.at.stood); err != nil {
		return err
	}
	_, err := fmt.Fprintf(run.out, "wrote the places %s\n", run.at.stood)
	return err
}
