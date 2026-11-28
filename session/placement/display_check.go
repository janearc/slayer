package placement

import (
	"fmt"
	"strings"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// DisplayThingError is a refusal that names the placement it came from. Its
// text is the placement's name, a colon and the refusal, as Scene has always
// worded it; Unwrap gives the refusal alone.
type DisplayThingError struct {
	// Thing is the name of the placement the display refused.
	Thing string
	// Err is what the display said about it.
	Err error
}

// Error is the placement's name and its refusal.
func (refusal *DisplayThingError) Error() string {
	return fmt.Sprintf("%s: %v", refusal.Thing, refusal.Err)
}

// Unwrap is the refusal without the placement's name.
func (refusal *DisplayThingError) Unwrap() error { return refusal.Err }

// CheckDisplay says whether the working display would take read, running the
// checks that cost nothing in the order Scene runs them: the set is in space
// with a seed and nothing unrendered, it has a film size, no rig and a fixed
// camera, and every placement passes displayGate. It makes no mesh. A
// placement's refusal is a *DisplayThingError. It returns nil when Scene
// would go on to make meshes, which can still fail.
func CheckDisplay(read set.Set) error {
	if _, err := displayShot(read); err != nil {
		return err
	}
	for _, each := range read.Things() {
		if _, _, err := displayGate(read, each); err != nil {
			return &DisplayThingError{Thing: each.Name(), Err: err}
		}
	}
	return nil
}

// displayShotFaults says which of the first check's three terms read fails:
// not in space, no seed, or keys nothing renders yet (the first is named).
func displayShotFaults(read set.Set) string {
	var faults []string
	if !read.InSpace() {
		faults = append(faults, "the set is not in space")
	}
	if !read.HasSeed() {
		faults = append(faults, "no seed")
	}
	if unrendered := read.Unrendered(); len(unrendered) != 0 {
		faults = append(faults, fmt.Sprintf("%d unrendered, first %s", len(unrendered), unrendered[0]))
	}
	return strings.Join(faults, ", ")
}

// displayShot is the part of a scene that needs no mesh: its film size, its
// first fixed camera and that camera's lens. It holds every set-wide check
// the display makes, in the order it makes them.
func displayShot(read set.Set) (*DisplayScene, error) {
	if !read.InSpace() || !read.HasSeed() || len(read.Unrendered()) != 0 {
		return nil, fmt.Errorf("working display needs supported native space placements and an explicit seed (%s)", displayShotFaults(read))
	}
	if world, has := read.World(); has {
		for _, body := range world.Bodies() {
			if _, flowing := body.Flowing(); flowing {
				return nil, fmt.Errorf("flowing globes require producer staging")
			}
		}
	}
	settings, has := read.Film()
	width, height, _ := settings.Size()
	if !has || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("working display needs an authored film size")
	}
	if _, rig := settings.Rig(); rig {
		return nil, fmt.Errorf("working display does not support lens rigs")
	}
	places := map[string]maths.Vec{}
	for _, each := range read.Things() {
		places[each.Name()] = SpaceAt(each).Origin
	}
	shots, err := read.Shots(func(name string) (maths.Vec, bool) { at, found := places[name]; return at, found })
	if err != nil {
		return nil, err
	}
	if len(shots) == 0 {
		return nil, fmt.Errorf("working display needs a fixed camera")
	}
	wide := settings.Wide() * maths.Degree
	if shots[0].Lens().Wide > 0 {
		wide = shots[0].Lens().Wide
	}
	lens, err := film.PinholeOf(wide)
	if err != nil {
		return nil, err
	}
	return &DisplayScene{Eye: shots[0].Eye(), Focus: shots[0].Focus(), Lens: *lens, Width: width, Height: height}, nil
}
