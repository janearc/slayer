package bodies_test

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// Every template places its parts: a rock world is a thing, with an air
// and moons where it has them; a giant is a volume with its rings; the
// hole is a dark ball and a disc.
func TestTemplatesPlaceTheirParts(t *testing.T) {
	for _, c := range []struct {
		body            bodies.Body
		things, volumes int
	}{
		{template.Mercury(), 1, 0},
		{template.Venus(), 1, 1},
		{template.Earth(), 2, 3}, // the air, the aurora and the clouds
		{template.Mars(), 3, 1},
		{template.Jupiter(), 4, 3}, // the gas, its aurora and io's plume
		{template.Saturn(), 3, 4},  // the gas, its aurora, the rings and titan's air
		{template.Uranus(), 3, 2},
		{template.Neptune(), 1, 1},
		{template.Pluto(), 1, 0},
		{template.Charon(), 1, 0},
		{template.Quasar(), 0, 2}, // the disc and the jets; the hole is a field
	} {
		w := thing.NewWorld()
		if err := c.body.Place(w, maths.Vec{}, 1/c.body.Radius, 4); err != nil {
			t.Fatal(err)
		}
		if len(w.Things) != c.things || len(w.Volumes) != c.volumes {
			t.Errorf("%s places %d things and %d volumes, want %d and %d",
				c.body.Name, len(w.Things), len(w.Volumes), c.things, c.volumes)
		}
	}
}

// A moon goes round at kepler's period for its parent's mass, at its
// orbit's radius in the world's units: io round jupiter in a day and
// three quarters, and the moon round the earth in twenty-seven days.
func TestMoonsOrbitByGravity(t *testing.T) {
	for _, c := range []struct {
		body bodies.Body
		name string
		days float64
	}{{template.Jupiter(), "io", 1.77}, {template.Earth(), "moon", 27.3}} {
		w := thing.NewWorld()
		scale := 1 / c.body.Radius
		if err := c.body.Place(w, maths.V(5, 0, 0), scale, 2); err != nil {
			t.Fatal(err)
		}
		var moon *thing.Thing
		for _, th := range w.Things {
			if th.Material.Name == c.name {
				moon = th
			}
		}
		if moon == nil {
			t.Fatalf("%s has no %s", c.body.Name, c.name)
		}
		o, ok := moon.Motion.(thing.Orbit)
		if !ok {
			t.Fatalf("%s is not on an orbit", c.name)
		}
		if d := o.Period.Hours() / 24; math.Abs(d-c.days) > 0.02*c.days {
			t.Errorf("%s goes round in %.2f days, want %.2f", c.name, d, c.days)
		}
		if got := o.At(0).Origin.Sub(maths.V(5, 0, 0)).Len(); math.Abs(got-c.body.Moons[0].Orbit*scale) > 1e-9 {
			t.Errorf("%s orbits at %v", c.name, got)
		}
	}
}

// Pluto and charon go round a point between them, nearer pluto by their
// masses, in six and a third days, and always opposite each other.
func TestBinaryGoesRoundTheCommonCentre(t *testing.T) {
	w := thing.NewWorld()
	// 19,591 km apart, their true distance
	if err := bodies.Binary(w, template.Pluto(), template.Charon(), maths.Vec{}, 19591, 1e-3, 3); err != nil {
		t.Fatal(err)
	}
	if len(w.Things) != 2 {
		t.Fatalf("%d things", len(w.Things))
	}
	p, c := w.Things[0].Motion.(thing.Orbit), w.Things[1].Motion.(thing.Orbit)
	if math.Abs(p.Radius-2.126) > 0.05 || math.Abs(c.Radius-17.46) > 0.1 {
		t.Errorf("pluto orbits at %v and charon at %v", p.Radius, c.Radius)
	}
	if d := p.Period.Hours() / 24; math.Abs(d-6.39) > 0.05 {
		t.Errorf("they go round in %.2f days", d)
	}
	a, b := p.At(100000e9).Origin, c.At(100000e9).Origin
	if apart := a.Sub(b).Len(); math.Abs(apart-19.591) > 1e-6 {
		t.Errorf("at some time they are %v apart", apart)
	}
	if a.Dot(b) >= 0 {
		t.Error("they are on the same side of the centre")
	}
}

// Venus's air passes radio and its ground returns it by roughness: the
// placed air's material passes a radio band, the ground's returns it,
// and the ground is rougher where it is rock than where it is plain.
func TestVenusIsMappedByRadio(t *testing.T) {
	radio := maths.Band{Nanometres: 1.26e8}
	w := thing.NewWorld()
	v := template.Venus()
	if err := v.Place(w, maths.Vec{}, 1/v.Radius, 8); err != nil {
		t.Fatal(err)
	}
	air := w.Volumes[0].Material
	if air.Radiant == nil || air.Radiant.Through(radio) != 1 {
		t.Fatal("venus's air does not pass radio")
	}
	ground := w.Things[0].Material
	if ground.Radiant == nil || ground.Radiant.Scatters(radio) != 0.9 || ground.Rough == nil {
		t.Fatal("venus's ground does not return radio by roughness")
	}
	lo, hi := 2.0, 0.0
	for i := range 200 {
		k := float64(i)
		p := maths.V(math.Cos(k), math.Sin(k*0.7), math.Sin(k)).Unit()
		r := ground.Roughness(p)
		lo, hi = math.Min(lo, r), math.Max(hi, r)
	}
	if !(lo < 0.3 && hi > 0.6) {
		t.Errorf("roughness runs only %.2f to %.2f", lo, hi)
	}
}

// The sun's schwarzschild radius is about three kilometres, and the
// quasar's disc is sized by it.
func TestSchwarzschild(t *testing.T) {
	if rs := bodies.Schwarzschild(1.989e30); math.Abs(rs-2.95) > 0.02 {
		t.Errorf("the sun's horizon would be %v km", rs)
	}
	q := template.Quasar()
	if q.Disc.Inner != 3*q.Radius || q.Disc.Outer != 30*q.Radius {
		t.Errorf("the disc runs %v to %v for a radius of %v", q.Disc.Inner,
			q.Disc.Outer, q.Radius)
	}
}

// A ground is a world in its look: mars has a dead volcano of rock and
// a canyon, and io has live ones.
func TestGroundsHaveTheirFeatures(t *testing.T) {
	count := func(b bodies.Body, detail int, want world.Biome) int {
		g := b.Ground
		tr := g.Terrain
		tr.Tiling = hexes.Sphere{N: max(1, detail)}
		plain := world.New(tr)
		fs, err := g.Features(plain)
		if err != nil {
			t.Fatal(err)
		}
		tr.Features = fs
		w := world.New(tr)
		n := 0
		for _, tile := range w.Tiles() {
			if w.Biome(tile) == want {
				n++
			}
		}
		return n
	}
	if n := count(template.Mars(), 10, world.Rock); n < 5 {
		t.Errorf("mars has %d rock tiles", n)
	}
	if n := count(template.Mars(), 10, world.Lava); n != 0 {
		t.Errorf("mars has %d lava tiles", n)
	}
	io := template.Jupiter().Moons[0].Body
	if n := count(io, 10, world.Lava); n < 5 {
		t.Errorf("io has %d lava tiles", n)
	}
}

// A moon is placed with its parts: io's plume goes round with io, a
// unit out from its centre, and titan's air is in the world.
func TestMoonsCarryTheirParts(t *testing.T) {
	w := thing.NewWorld()
	j := template.Jupiter()
	scale := 1 / j.Radius
	if err := j.Place(w, maths.Vec{}, scale, 8); err != nil {
		t.Fatal(err)
	}
	var plume *thing.Volume
	for _, v := range w.Volumes {
		if v.Material.Name == "io plume" {
			plume = v
		}
	}
	if plume == nil {
		t.Fatal("io's plume was not placed")
	}
	o := j.Orbit(0, maths.Vec{}, scale)
	for _, at := range []time.Duration{0, o.Period / 4, o.Period / 2} {
		from := plume.Motion.At(at).Origin.Sub(o.At(at).Origin).Len()
		if math.Abs(from-1821.6*scale) > 1e-9 {
			t.Fatalf("at %v the plume stands %v from io's centre", at, from)
		}
	}
	s := template.Saturn()
	w = thing.NewWorld()
	if err := s.Place(w, maths.Vec{}, 1/s.Radius, 8); err != nil {
		t.Fatal(err)
	}
	for _, v := range w.Volumes {
		if v.Material.Name == "titan air" {
			return
		}
	}
	t.Fatal("titan's air was not placed")
}

// Stand is on the ground, where asked, in the body's own turn.
func TestStandIsOnTheGroundAndUpright(t *testing.T) {
	space := maths.Subspace{Origin: maths.V(10, 0, 0),
		Orientation: maths.About(maths.Up, math.Pi/2)}
	at, up := bodies.Stand(space, 2, maths.XAxis, 0.5)
	if at.Sub(maths.V(10, 0, -2.5)).Len() > 1e-9 || up.Sub(maths.V(0, 0, -1)).Len() > 1e-9 {
		t.Fatalf("standing at %v with up %v", at, up)
	}
}

// The bodies that are not planets: a quasar's jets are beamed and
// nested in its space; a pulsar's star turns once its Day, 33 ms, and its beams
// with it, tilted; a magnetar's flare is nothing before it goes off and
// grows at its speed after.
func TestStarsJetsAndFlares(t *testing.T) {
	w := thing.NewWorld()
	q := template.Quasar()
	if err := q.Place(w, maths.Vec{}, 1/q.Radius, 4); err != nil {
		t.Fatal(err)
	}
	var jets *thing.Volume
	for _, v := range w.Volumes {
		if v.Material.Name == "quasar jets" {
			jets = v
		}
	}
	if jets == nil || jets.Beam == nil || jets.Beam.Speed != 0.6 {
		t.Fatal("the quasar's jets are not beamed")
	}

	w = thing.NewWorld()
	p := template.Pulsar()
	if err := p.Place(w, maths.Vec{}, 1/p.Radius, 4); err != nil {
		t.Fatal(err)
	}
	star := w.Things[0]
	if star.Material.Name != "pulsar" || star.Material.Glow == (maths.Light{}) {
		t.Fatalf("the pulsar's star is %q, glow %v", star.Material.Name, star.Material.Glow)
	}
	// a quarter of a turn on, the beams' axis has swung a quarter round
	axis := func(at time.Duration) maths.Vec {
		return w.Volumes[0].At(at).Orientation.Out(maths.Up)
	}
	a0, a1 := axis(0), axis(p.Day/4)
	if math.Abs(a0.Y-a1.Y) > 1e-9 || math.Abs(a0.Dot(a1)-0.5) > 1e-6 {
		t.Errorf("the beams' axis went from %v to %v in a quarter turn", a0, a1)
	}
	if math.Abs(a0.Y-math.Cos((45)*maths.Degree)) > 1e-9 {
		t.Errorf("the beams lean %v from the spin", a0.Y)
	}

	w = thing.NewWorld()
	m := template.Magnetar()
	if err := m.Place(w, maths.Vec{}, 1/m.Radius, 4); err != nil {
		t.Fatal(err)
	}
	flare := w.Volumes[0]
	if flare.Material.Name != "magnetar flare" {
		t.Fatalf("the first volume is %q", flare.Material.Name)
	}
	if s := flare.At(0).Scale; s > 1e-6 {
		t.Errorf("before the flare its scale is %v", s)
	}
	if s := flare.At(m.Flare.At + time.Millisecond).Scale; math.Abs(s-30000/12*1e-3) > 1e-9 {
		t.Errorf("a millisecond in the fireball's scale is %v", s)
	}
}

// The earth is placed with its clouds as a volume and its cities as a
// shine on the ground: lit somewhere on land, dark at sea.
func TestEarthHasCloudsAndCities(t *testing.T) {
	w := thing.NewWorld()
	e := template.Earth()
	e.Moons = nil
	if err := e.Place(w, maths.Vec{}, 1/e.Radius, 8); err != nil {
		t.Fatal(err)
	}
	clouds := false
	for _, v := range w.Volumes {
		if v.Material.Name == "earth clouds" {
			clouds = true
		}
	}
	if !clouds {
		t.Error("no clouds")
	}
	ground := w.Things[0].Material
	if ground.Shine == nil {
		t.Fatal("no cities")
	}
	lit := 0
	for i := range 2000 {
		k := int64(i)
		p := maths.V(noise.Hash(1, k)-0.5, noise.Hash(2, k)-0.5, noise.Hash(3, k)-0.5).Unit()
		if ground.Shine.At(p).Luminance() > 0 {
			lit++
		}
	}
	if lit < 20 || lit > 600 {
		t.Errorf("%d of 2000 points on the earth are lit at night", lit)
	}
}

// A ground with a map wears it in place of its paint, and one with
// lights shines them in place of its cities.
func TestAGroundWearsItsMapAndLights(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{200, 100, 50, 255})
		}
	}
	e := template.Earth()
	e.Moons = nil
	e.Ground.Map, e.Ground.Lights = img, img
	w := thing.NewWorld()
	if err := e.Place(w, maths.Vec{}, 1/e.Radius, 4); err != nil {
		t.Fatal(err)
	}
	m := w.Things[0].Material
	paint := m.Texture.At(maths.V(0, 0, 1))
	if !(paint.R > paint.G && paint.G > paint.B) {
		t.Errorf("the map paints %v", paint)
	}
	if m.Shine == nil || m.Shine.At(maths.V(0, 0, 1)).Luminance() > 0.01 {
		t.Error("the lights are not the picture, scaled down")
	}
}

// A body with an aurora places it as a volume in a shell above the
// surface, glowing on the oval and dark at the equator.
func TestAuroraIsPlaced(t *testing.T) {
	w := thing.NewWorld()
	e := template.Earth()
	e.Moons = nil
	if err := e.Place(w, maths.Vec{}, 1/e.Radius, 6); err != nil {
		t.Fatal(err)
	}
	var aurora *thing.Volume
	for _, v := range w.Volumes {
		if v.Material.Name == "earth aurora" {
			aurora = v
		}
	}
	if aurora == nil {
		t.Fatal("no aurora")
	}
	up := 200 / e.Radius
	axis := maths.About(maths.XAxis, e.Aurora.Tilt).Out(maths.Up)
	o := maths.Upright(axis)
	on := axis.Scale(math.Sin(e.Aurora.Lat)).Add(o.X.Scale(math.Cos(e.Aurora.Lat))).Scale(1 + up)
	if d := aurora.Medium.Density(on); d <= 0 {
		t.Errorf("the oval is dark: %v", d)
	}
	if d := aurora.Medium.Density(o.X.Scale(1 + up)); d != 0 {
		t.Errorf("the equator glows %v", d)
	}
	if c := aurora.Material.Shine.At(on); !(c.G > c.R && c.G > c.B) {
		t.Errorf("the earth's aurora glows %v, not green", c)
	}
}

// A comet aimed at a sun has its coma and its tails placed, the tails
// streaming away from the sun and the dust tail bending back along
// the trail; a hot jupiter has one tail.
func TestTailsStreamAwayFromTheSun(t *testing.T) {
	w := thing.NewWorld()
	c := template.HaleBopp()
	sun := maths.V(-100, 0, 0)
	c.Aim(sun, maths.V(0, 0, 1))
	scale := 1e-6
	at := maths.V(0, 0, 0)
	if err := c.Place(w, at, scale, 2); err != nil {
		t.Fatal(err)
	}
	var dust, ion *thing.Volume
	for _, v := range w.Volumes {
		switch {
		case v.Material.Name == "hale-bopp coma":
		case dust == nil && v.Material.Name == "hale-bopp tail":
			dust = v
		case v.Material.Name == "hale-bopp tail":
			ion = v
		}
	}
	if dust == nil || ion == nil {
		t.Fatal("a tail is missing")
	}
	// the tail's y is away from the sun: plus x
	if away := dust.At(0).Orientation.Out(maths.Up); away.Sub(maths.XAxis).Len() > 1e-9 {
		t.Errorf("the dust tail streams %v", away)
	}
	// the dust tail's x is back along the trail: minus z
	if back := dust.At(0).Orientation.Out(maths.XAxis); back.Sub(maths.V(0, 0, -1)).Len() > 1e-9 {
		t.Errorf("the dust tail bends toward %v", back)
	}
	// ten units down the tail there is gas, and none before the head
	if d := dust.Medium.Density(maths.V(0.35*100/40, 10, 0)); d <= 0 {
		t.Errorf("ten units out on its bend the dust tail is %v", d)
	}
	h := template.HotJupiter()
	if len(h.Tails) != 1 || h.Day != time.Duration((18)*float64(time.Hour)) {
		t.Error("a hot jupiter has one tail and an eighteen hour day")
	}
}

// A body with no ground has no world to give and says so; a body whose
// feature has nowhere to go refuses to be placed, naming the body and
// the feature, and puts nothing in the world; and the templates' own
// features all find their land.
func TestWorldAndPlaceRefuseWithTheReason(t *testing.T) {
	if _, err := template.Jupiter().World(4); err == nil ||
		!strings.Contains(err.Error(), "jupiter has no ground") {
		t.Errorf("jupiter's world: %v", err)
	}
	v := template.Venus()
	v.Ground.Features = func(w *world.World) ([]world.Feature, error) {
		if _, ok := w.Find(maths.V(0, 1, 0), func(world.Biome) bool { return false }); !ok {
			return nil, fmt.Errorf("no land for maxwell")
		}
		return nil, nil
	}
	w := thing.NewWorld()
	err := v.Place(w, maths.Vec{}, 1/v.Radius, 4)
	if err == nil || !strings.Contains(err.Error(), "venus: no land for maxwell") {
		t.Errorf("venus placed with %v", err)
	}
	if len(w.Things)+len(w.Volumes) != 0 {
		t.Errorf("a refused venus still put %d things in the world", len(w.Things))
	}
	for _, b := range []bodies.Body{template.Venus(), template.Mars(), template.Pluto(), template.Jupiter().Moons[0].Body} {
		if _, err := b.World(6); err != nil {
			t.Errorf("%s: %v", b.Name, err)
		}
	}
}
