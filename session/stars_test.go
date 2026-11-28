package session

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// shining is a star of a radius in metres, a temperature and a
// luminosity in watts.
func shining(radius, kelvin, watts float64) generate.Shine {
	return generate.Shine{Name: "a star", Mass: measure.SolarMasses(1),
		Temperature: measure.Kelvin(kelvin),
		Luminosity:  measure.Watts(watts), Radius: measure.Metres(radius)}
}

// A star's lamp stands where the star stands, starts at the star's
// surface so the ball does not shade what it lights, and falls with the
// square of the distance. The anti-test: it does not fall off in
// proportion to the distance, and inside the ball it gives nothing.
func TestAStarsLampFallsBySquareFromWhereItStands(t *testing.T) {
	at := maths.V(100, 200, -300)
	_, lamp, err := starOf(shining(5, 9500, 1e3), at, 1, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if lamp.At != at || lamp.Beyond != 5 {
		t.Errorf("a lamp at %v from %v, want %v from 5", lamp.At,
			lamp.Beyond, at)
	}
	nearWay, nearReach, nearLight := lamp.Light(at.Add(maths.V(50, 0, 0)),
		maths.Vec{})
	_, _, farLight := lamp.Light(at.Add(maths.V(100, 0, 0)), maths.Vec{})
	if nearWay.Dot(maths.V(-1, 0, 0)) < 0.999 || !(nearReach > 0) {
		t.Errorf("light comes by %v over %v", nearWay, nearReach)
	}
	// both are measured from the centre, at 50 and 100 metres
	if ratio := nearLight.R / farLight.R; math.Abs(ratio-4) > 1e-9 {
		t.Errorf("twice as far gives %v times less light, want 4", ratio)
	}
	if _, _, inside := lamp.Light(at.Add(maths.V(1, 0, 0)),
		maths.Vec{}); inside != (maths.Light{}) {
		t.Errorf("inside the ball a lamp gives %v", inside)
	}
}

// Two stars light in two colours: a hot one's light is bluer than a cool
// one's, in its lamp and in its ball. The anti-test: neither is grey.
func TestTwoStarsLightInTwoColours(t *testing.T) {
	hotBall, hot, err := starOf(shining(5, 25000, 1e3), maths.Vec{}, 1, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	coolBall, cool, err := starOf(shining(5, 5800, 1e3), maths.Vec{}, 1, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	blueness := func(light maths.Light) float64 { return light.B / light.R }
	if !(blueness(hot.Strength) > blueness(cool.Strength)) {
		t.Errorf("a hot lamp's blue over red is %v, a cool lamp's %v",
			blueness(hot.Strength), blueness(cool.Strength))
	}
	if !(blueness(hotBall.Material.Glow) > blueness(coolBall.Material.Glow)) {
		t.Errorf("a hot ball's blue over red is %v, a cool ball's %v",
			blueness(hotBall.Material.Glow), blueness(coolBall.Material.Glow))
	}
	if hotBall.Material.Limb >= coolBall.Material.Limb {
		t.Errorf("the hot ball is darkened by %v at its limb, the cool "+
			"ball by %v", hotBall.Material.Limb, coolBall.Material.Limb)
	}
}

// What the film's exposure scales it scales both: double the exposure,
// double the ball's light and the lamp's.
func TestExposureScalesBothTheBallAndTheLamp(t *testing.T) {
	ball, lamp, _ := starOf(shining(5, 9500, 1e3), maths.Vec{}, 1, 1, 0)
	ball2, lamp2, _ := starOf(shining(5, 9500, 1e3), maths.Vec{}, 1, 2, 0)
	if got := ball2.Material.Glow.G / ball.Material.Glow.G; math.Abs(got-2) > 1e-9 {
		t.Errorf("the ball's light is %v times for twice the exposure", got)
	}
	if got := lamp2.Strength.G / lamp.Strength.G; math.Abs(got-2) > 1e-9 {
		t.Errorf("the lamp's light is %v times for twice the exposure", got)
	}
}

// A star too small to cover a pixel is drawn wider and dimmer, with the
// light of its disc unchanged: the ball's area times its glow is the same
// at any drawn size. The anti-test: a star already wider than the least
// is not changed.
func TestASmallStarIsDrawnWiderAndDimmerWithItsLightKept(t *testing.T) {
	area := func(ball thing.Thing) float64 {
		radius := ball.Shape.(shape.Sphere).Radius
		return radius * radius * ball.Material.Glow.G
	}
	true1, _, _ := starOf(shining(5, 9500, 1e3), maths.Vec{}, 1, 1, 0)
	wide, _, _ := starOf(shining(5, 9500, 1e3), maths.Vec{}, 1, 1, 500)
	if radius := wide.Shape.(shape.Sphere).Radius; radius != 500 {
		t.Errorf("the small star is drawn %v wide, want 500", radius)
	}
	if got, want := area(wide), area(true1); math.Abs(got-want) > want*1e-9 {
		t.Errorf("the wide ball's light is %v, the star's %v", got, want)
	}
	kept, _, _ := starOf(shining(5, 9500, 1e3), maths.Vec{}, 1, 1, 2)
	if kept.Shape.(shape.Sphere).Radius != 5 || area(kept) != area(true1) {
		t.Errorf("a star wider than the least was changed")
	}
}

// probeLit is the light on the front of a white ball, five metres in
// front of the eye, from a star far behind the eye, with a blocker
// between them where asked.
func probeLit(t *testing.T, blocked bool) float64 {
	t.Helper()
	world := thing.NewWorld()
	ball, lamp, err := starOf(shining(1, 9500, 1e6), maths.V(0, 0, 50), 1, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	world.Add(ball)
	world.Lamp(lamp)
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "probe", Colour: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 0, -5)})})
	if blocked {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 3},
			Material: &thing.Material{Name: "blocker",
				Colour: maths.Grey(1)},
			Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, 10)})})
	}
	const side = 5
	lens := film.Pinhole{Wide: 10 * maths.Degree}
	picture, err := trace.Render(world, maths.Local(), lens, 0, side, side)
	if err != nil {
		t.Fatal(err)
	}
	return picture.At(side/2, side/2).Light().R
}

// The anti-test: a star never lights what is behind another body. The
// probe is lit by the star with nothing between them, and not at all
// with a ball in the way.
func TestAStarDoesNotLightWhatIsBehindAnotherBody(t *testing.T) {
	if open := probeLit(t, false); !(open > 0) {
		t.Fatalf("the probe in the open is lit by %v", open)
	}
	if shaded := probeLit(t, true); shaded != 0 {
		t.Errorf("the probe in a ball's shadow is lit by %v", shaded)
	}
}

// A star's ball is brighter at the centre of its disc than at its limb,
// by the limb's law: seen from close, the centre pixel's light over the
// edge pixel's is the law's share at the edge's angle.
func TestAStarsDiscIsDarkerAtItsLimb(t *testing.T) {
	world := thing.NewWorld()
	ball, _, err := starOf(shining(1, 9500, 1e6), maths.V(0, 0, -3), 1, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	world.Add(ball)
	const side = 41
	// the disc subtends about 38 degrees from three metres; this frame
	// is 40 across, so its edge pixel is just inside the limb
	lens := film.Pinhole{Wide: 40 * maths.Degree}
	picture, err := trace.Render(world, maths.Local(), lens, 0, side, side)
	if err != nil {
		t.Fatal(err)
	}
	centre := picture.At(side/2, side/2).Light().G
	edge := picture.At(side/2, 1).Light().G
	if !(centre > 0 && edge > 0 && edge < centre*0.75) {
		t.Errorf("the disc's centre is %v and its limb %v", centre, edge)
	}
}
