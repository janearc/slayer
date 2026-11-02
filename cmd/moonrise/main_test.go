package main

import (
	"errors"
	"io"
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
)

// mustAsk is what the command is asked by these arguments, drawn small.
func mustAsk(t *testing.T, arguments ...string) asked {
	t.Helper()
	given, err := parse(append([]string{"-w", "36", "-h", "25", "-n", "1",
		"-out", t.TempDir() + "/moonrise.pfm"}, arguments...), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return given
}

// mustScene is the scene as asked, or the test fails.
func mustScene(t *testing.T, given asked) scene {
	t.Helper()
	made, err := compose(given)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// the moon stands at the altitude asked, straight ahead, and is
// 0.518 degrees across: its radius and distance as nasa's moon fact
// sheet gives them, 1737.4 km at 384,400 km.
func TestTheMoonIsWhereAndAsLargeAsAsked(t *testing.T) {
	made := mustScene(t, mustAsk(t, "-moon", "0.3"))
	toward := made.moon.Sub(made.eye.Origin)
	altitude := math.Asin(toward.Unit().Y) * 180 / math.Pi
	across := 2 * math.Atan(moonRadius/toward.Len()) * 180 / math.Pi
	if math.Abs(altitude-0.3) > 1e-9 || math.Abs(across-0.5179) > 1e-3 {
		t.Errorf("the moon is %.6f deg up and %.4f across", altitude,
			across)
	}
}

// the sun is distant and behind the eye, at the altitude asked: the
// way to it points back past the eye, away from the moon.
func TestTheSunIsDistantAndBehind(t *testing.T) {
	made := mustScene(t, mustAsk(t, "-sun", "2"))
	sun, distant := made.world.Lamps[0].(emit.Distant)
	if !distant {
		t.Fatalf("the sun is a %T", made.world.Lamps[0])
	}
	toward, far, _ := sun.Light(maths.Vec{}, maths.Up)
	ahead := made.moon.Sub(made.eye.Origin)
	if !math.IsInf(far, 1) || toward.Dot(ahead) >= 0 ||
		math.Abs(math.Asin(toward.Y)*180/math.Pi-2) > 1e-9 {
		t.Errorf("the sun is toward %v, %g away", toward, far)
	}
}

// the ground is the earth, a globe whose top is under the tree.
func TestTheTreeStandsOnTheEarth(t *testing.T) {
	made := mustScene(t, mustAsk(t))
	globes := 0
	for _, each := range made.world.Things {
		if _, round := each.Shape.(globe.Globe); round {
			globes++
		}
	}
	if globes != 2 {
		t.Errorf("there are %d globes; the earth and the moon are two",
			globes)
	}
}

// at the middle of the frame the moon is lit: it is clear of the
// earth's shadow, which an exactly opposite sun would put it in.
func TestTheMoonIsLitNotEclipsed(t *testing.T) {
	given := mustAsk(t, "-moon", "1.5", "-sun", "2")
	picture, err := draw(mustScene(t, given), given)
	if err != nil {
		t.Fatal(err)
	}
	if middle := picture.At(given.width/2, given.height/2); middle.R <= 0 {
		t.Errorf("the moon's middle is %v", middle)
	}
}

// where the raw goes must be said, and a lens of zero, a sun below the horizon
// or a frame of no size is refused.
func TestWhatCannotBeDrawnIsRefused(t *testing.T) {
	for _, wrong := range [][]string{{}, {"-out", "x.pfm", "-wide", "0"},
		{"-out", "x.pfm", "-sun", "-1"}, {"-out", "x.pfm", "-w", "0"}} {
		if _, err := parse(wrong, io.Discard); !errors.Is(err, errAsked) {
			t.Errorf("%v is taken: %v", wrong, err)
		}
	}
}
