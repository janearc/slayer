//go:build draw

package take_test

import (
	"image"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm"
	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/film"
	"github.com/janearc/slayer/ninjafilm/frame"
	"github.com/janearc/slayer/render/carry"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/take"
	"github.com/janearc/slayer/trace"
)

// walk is a path: from one place to another in a straight line, facing
// a point the whole way.
type walk struct {
	from, to, toward maths.Vec
	over             time.Duration
}

// shoot films two seconds of a world on a stock, the camera walking
// while it looks at one point, and gives back the frames as a projector
// lit to an exposure would show them.
func shoot(world *thing.World, stock string, exposure float64, width,
	height int) ([]*image.RGBA, error) {
	// 1. a roll of a stock. the seed is which roll.
	roll, err := ninjafilm.Roll(stock, 1921)
	if err != nil {
		return nil, err
	}
	// 2. how the camera is cranked: sixteen frames a second, a
	// shutter open 170 degrees, three glimpses through each opening.
	pace, err := take.Cranked(16, 170*maths.Degree, 3)
	if err != nil {
		return nil, err
	}
	// 3. the path, cranked, for two seconds.
	length := 2 * time.Second
	path := walk{from: maths.V(-2, 1.5, 8), to: maths.V(2, 1.5, 8),
		toward: maths.V(0, 1, 0), over: length}
	shot, err := take.Take(path, pace, length)
	if err != nil {
		return nil, err
	}
	// the lens looks through the roll's own gauge, with square pixels.
	lens := film.NewLens(50, roll.Gauge(), 0)
	var reel []*image.RGBA
	for number := range shot.Frames() {
		// 4. render what the camera saw at each glimpse.
		var seen []*frame.Frame
		for _, glimpse := range shot.Glimpses(number) {
			drawn, err := trace.Sampled(world, glimpse.Eye(), lens,
				glimpse.At(), width, height, 2)
			if err != nil {
				return nil, err
			}
			seen = append(seen, carry.Ninja(drawn))
		}
		// 5. the roll makes a frame of film of them.
		printed, err := roll.Print(number, seen)
		if err != nil {
			return nil, err
		}
		// and a projector throws it on a screen.
		reel = append(reel, colour.Develop(printed, exposure))
	}
	return reel, nil
}

// set is a ball of a colour on grey ground, under a sun
func set(ball maths.Light) *thing.World {
	world := thing.NewWorld()
	world.Lamp(thing.Sun(maths.V(1, 3, 2), maths.Grey(1).Scale(matteLamp)))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Colour: ball},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(0, 1, 0)})})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 500},
		Material: &thing.Material{Colour: maths.Grey(0.3)},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(0, -500, 0)})})
	return world
}

// middle is the pixel at the middle of a picture, where the ball is
func middle(picture *image.RGBA) (red, green, blue int) {
	size := picture.Bounds().Size()
	at := picture.RGBAAt(size.X/2, size.Y/2)
	return int(at.R), int(at.G), int(at.B)
}

func TestEveryStockFilmsAScene(t *testing.T) {
	if testing.Short() {
		t.Skip("a smoke test: it films a scene on every stock")
	}
	names, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	exposures := map[string]float64{"soot": 2, "ash": 2, "smoke": 2,
		"dusk": 4, "lantern": 4, "pollen": 16}
	for _, name := range names {
		reel, err := shoot(set(maths.RGB(0.8, 0.1, 0.1)), name,
			exposures[name], 192, 24)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(reel) != 32 {
			t.Fatalf("%s: two seconds at sixteen is %d frames",
				name, len(reel))
		}
		lit := 0
		for index, value := range reel[0].Pix {
			if index%4 != 3 && value > 16 {
				lit++
			}
		}
		if lit < len(reel[0].Pix)/8 {
			t.Errorf("%s: its first frame is nearly black: %d "+
				"of %d lamps lit", name, lit, len(reel[0].Pix))
		}
		differ := 0
		for index, value := range reel[0].Pix {
			if value != reel[31].Pix[index] {
				differ++
			}
		}
		if differ < len(reel[0].Pix)/20 {
			t.Errorf("%s: the camera walked and %d of %d "+
				"lamps changed",
				name, differ, len(reel[0].Pix))
		}
	}
}

// matteLamp is how many times stronger the sun in set is made than the
// strength set names, for a matte surface to show its colour times that
// strength. trace reads a lamp's strength as irradiance, so a matte surface
// shows its colour over pi times it; the film's exposures are chosen for a
// white surface square on to the sun showing one unit of radiance.
const matteLamp = math.Pi

// a red ball and a blue one, on the stock that cannot see red and on
// the slide: the first prints the red ball dark and the blue one light,
// the second keeps them red and blue
func TestTheStocksTellRedFromBlueAsTheyShould(t *testing.T) {
	if testing.Short() {
		t.Skip("a smoke test: it films two scenes on two stocks")
	}
	film := func(stock string, ball maths.Light) (int, int, int) {
		reel, err := shoot(set(ball), stock, 4, 192, 24)
		if err != nil {
			t.Fatal(err)
		}
		return middle(reel[16])
	}
	red, blue := maths.RGB(0.8, 0.05, 0.05), maths.RGB(0.05, 0.05, 0.8)
	onAshRed, _, _ := film("ash", red)
	onAshBlue, _, _ := film("ash", blue)
	if onAshRed*2 > onAshBlue {
		t.Errorf("ash prints a red ball at %d and a blue one at %d",
			onAshRed, onAshBlue)
	}
	r, g, b := film("lantern", red)
	if r < 2*g || r < 2*b {
		t.Errorf("a slide prints a red ball as %d, %d, %d", r, g, b)
	}
	r, g, b = film("lantern", blue)
	if b < 2*r || b < 2*g {
		t.Errorf("a slide prints a blue ball as %d, %d, %d", r, g, b)
	}
}

func (path walk) At(at time.Duration) maths.Subspace {
	along := at.Seconds() / path.over.Seconds()
	eye := path.from.Add(path.to.Sub(path.from).Scale(along))
	return maths.Subspace{Origin: eye,
		Orientation: maths.LookAt(eye, path.toward)}
}
