package lantern_test

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/scene/thing"
)

// the values the played tests hold, picked by the author of this file,
// 2026-11-12: twice as bright, half again as wide, a quarter turn
const (
	twice   = 2.0
	wider   = 1.5
	quarter = 0.25
	third   = 3
)

// played is zappy placed with its channels in a world of its own, at
// rest, and what Place put there, kept to compare. Placing zappy takes
// seconds, so the tests share one, each setting it to rest first.
func played(t *testing.T) (*lantern.Played, *thing.World, []*thing.Thing,
	[]thing.Lamp) {
	t.Helper()
	shared.once.Do(func() {
		shared.world = thing.NewWorld()
		shared.made, shared.err = lantern.Zappy().Play(shared.world,
			new(time.Duration))
		shared.things = append([]*thing.Thing(nil),
			shared.world.Things...)
		shared.lamps = append([]thing.Lamp(nil), shared.world.Lamps...)
	})
	if shared.err != nil {
		t.Fatal(shared.err)
	}
	if err := shared.made.Hold(nil); err != nil {
		t.Fatal(err)
	}
	return shared.made, shared.world, shared.things, shared.lamps
}

// shared is the one played zappy the tests share.
var shared struct {
	once   sync.Once
	made   *lantern.Played
	world  *thing.World
	things []*thing.Thing
	lamps  []thing.Lamp
	err    error
}

// changed are the places in the world whose thing is not the one Place
// put there.
func changed(world *thing.World, placed []*thing.Thing) []int {
	var moved []int
	for i, one := range world.Things {
		if one != placed[i] {
			moved = append(moved, i)
		}
	}
	return moved
}

// lampsChanged are the places in the world whose lamp is not Place's.
func lampsChanged(world *thing.World, placed []thing.Lamp) []int {
	var moved []int
	for i, one := range world.Lamps {
		if !reflect.DeepEqual(one, placed[i]) {
			moved = append(moved, i)
		}
	}
	return moved
}

// The star's brightness changes the star and its light, and nothing else:
// no shell, not the beams.
func TestStarBrightnessMovesTheStarAlone(t *testing.T) {
	made, world, things, lamps := played(t)
	if err := made.Hold(map[string]float64{
		"star-brightness": twice}); err != nil {
		t.Fatal(err)
	}
	if moved := changed(world, things); len(moved) != 1 || moved[0] != 0 {
		t.Errorf("brightness moved things %v, want the star alone", moved)
	}
	if moved := lampsChanged(world, lamps); len(moved) != 1 ||
		moved[0] != 0 {
		t.Errorf("brightness moved lamps %v, want the star's light", moved)
	}
}

// One shell's breath moves that shell's two sheets and nothing else.
func TestOneShellsBreathMovesThatShellAlone(t *testing.T) {
	made, world, things, lamps := played(t)
	err := made.Hold(map[string]float64{"shell-3-breath": wider})
	if err != nil {
		t.Fatal(err)
	}
	// the star, then two sheets to a shell: shell three is 5 and 6
	moved := changed(world, things)
	if len(moved) != 2 || moved[0] != 5 || moved[1] != 6 {
		t.Errorf("shell three's breath moved things %v, want 5 and 6",
			moved)
	}
	if len(lampsChanged(world, lamps)) != 0 {
		t.Error("a shell's breath moved a lamp")
	}
	grown := world.Things[5].Motion.At(time.Second).Scale
	if grown != wider {
		t.Errorf("shell three is grown %v, want %v", grown, wider)
	}
}

// The beams' channels make the beams again and leave the star's light.
func TestTheBeamsChannelsMoveTheBeamsAlone(t *testing.T) {
	for _, name := range []string{"beam-width", "beam-spin",
		"beam-brightness", "glare", "beam-cone", "beam-precess"} {
		made, world, things, lamps := played(t)
		if err := made.Hold(map[string]float64{name: quarter}); err != nil {
			t.Fatal(err)
		}
		moved := lampsChanged(world, lamps)
		if len(moved) != 1 || moved[0] != 1 ||
			len(changed(world, things)) != 0 {
			t.Errorf("%s moved lamps %v and things %v, want the beams",
				name, moved, changed(world, things))
		}
	}
}

// A shell's own channel, off its rest, wins over the all-shells one.
func TestAShellsOwnChannelWins(t *testing.T) {
	made, world, _, _ := played(t)
	if err := made.Hold(map[string]float64{"shells-breath": twice,
		"shell-1-breath": wider}); err != nil {
		t.Fatal(err)
	}
	first := world.Things[1].Motion.At(0).Scale
	second := world.Things[third].Motion.At(0).Scale
	if first != wider || second != twice {
		t.Errorf("shell one is grown %v and two %v, want %v and %v",
			first, second, wider, twice)
	}
}

// anti-tests: a channel zappy has not, and a value out of a channel's
// range, are refused by name, and nothing is changed.
func TestAChannelZappyHasNotIsRefused(t *testing.T) {
	made, world, things, _ := played(t)
	for _, wrong := range []struct {
		values map[string]float64
		says   string
	}{
		{map[string]float64{"star-brightnes": 1}, "star-brightness"},
		{map[string]float64{"shell-13-band": 1}, "shell-12-band"},
		{map[string]float64{"star-breath": twice}, "0.25 to 1.5"},
		{map[string]float64{"star-brightness": twice,
			"shells-open": wider}, "0 to 1"},
	} {
		err := made.Hold(wrong.values)
		if !errors.Is(err, lantern.ErrChannel) ||
			!strings.Contains(err.Error(), wrong.says) {
			t.Errorf("%v: %v; want ErrChannel saying %q", wrong.values,
				err, wrong.says)
		}
	}
	if len(changed(world, things)) != 0 {
		t.Error("a refused hold changed the world")
	}
}

// Every channel says what it does and rests within its range.
func TestEveryChannelSaysItselfAndRestsInRange(t *testing.T) {
	made, _, _, _ := played(t)
	for _, one := range made.Channels() {
		if one.Says == "" || one.Unit == "" || one.Rest < one.Least ||
			one.Rest > one.Most {
			t.Errorf("%+v", one)
		}
	}
}
