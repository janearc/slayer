# Lighting a set: night exteriors and bright interiors

A practical light has two parts: the fitting visible in the frame and
the illumination it casts. In the historical API described here, the
manufactory supplies geometry, so a lantern also needs a separate emitter
to light the scene around it.

This guide targets `manufacture` at `3455095`, Lamb at `a623630`, and
Slayer’s `emit` on `trees` at `367cb8a`. Dimensions and behaviour were
measured on 2026-10-06. The examples and failure reports apply to those
revisions, not the current seeded construction API.

## The fittings

Each of the four fittings accepts a base point and parameter struct
and returns triangles.

| fitting | call | size | triangles | material ids | where its light belongs |
|---|---|---|---|---|---|
| sconce | `SconceBracket(base, DefaultSconceBracketParams())` | an arm 0.30 out along x and 0.25 up | 28 | 9 bracket, 10 bulb | the bulb: 0.265 along x, 0.15 up |
| pendant | `PendantLight(base, DefaultPendantParams())` | 0.16 wide, 1.20 long | 32 | 11 cord, 12 shade | just past the shade: 1.25 along z |
| lantern | `Lantern(base, DefaultLanternParams())` | a box 0.15 by 0.15, 0.30 tall | 10 | 13 top, 14 the four panels | its middle: 0.15 up |
| spotlight | `Spotlight(base, DefaultSpotlightParams())` | 0.16 wide, 0.40 long | 44 | 15 mount, 16 head | the head's far end: 0.40 along z |

Three things about them:

- **Up is z**, as for all the manufactory's pieces. doc/clutter.md has
  `upright` and `dressed`, which this guide uses.
- **The pendant is made standing up**: its cord rises a metre from its
  base and its shade sits on top. to hang it, turn it over and put its
  base on the ceiling. The code below does.
- **The lantern has no floor.** it is four panels and a lid. From below
  you see inside it.

## The light

The light is one of ninjafilm's emitters, or slayer's point lamp, added to
the world with `world.Lamp`.

| lamp | whose | what it is | use it for |
|---|---|---|---|
| `emit.Lit{At, Strength, Beyond}` | slayer | a point that shines every way, falling off with the square of distance | a flame, a bulb |
| `film.NewSpot(at, aim, strength, cone, soft)` | ninjafilm | a cone of light from a point | a spotlight, a pendant's pool on a desk |
| `film.NewFlood(up, strength)` | ninjafilm | an even dome of light from above | an overcast sky, a lit ceiling |
| `film.NewSun(toward, strength)` | ninjafilm | parallel light from one direction | the sun, the moon |

`Strength` is the light one unit away. `Beyond` on a point lamp is how
far from the lamp its light starts, so that the fitting round it does
not shade it: set it just past the fitting's own glass.

**Renderer compatibility at these revisions:** Lamb’s `trace.Sampled`
accepts all four lamp types. Slayer’s GPU then accepted only `emit.Lit`;
a type assertion in `gpu/parts.go` panicked on other lamps. Current GPU
validation has changed, so this is a historical limitation.

Use the historical CPU path for these spotlight and flood examples.
Check current GPU validation before adapting them to a later revision.

## The colour of the light

Warm flame interacts strongly with a stock’s limited spectral sensitivity.
libtheme computes thermal colour from temperature:

```go
// glowing is the light of something at a temperature, in kelvin, as
// lamb's three lamps, with a luminance of one. a flame is redder than
// a screen's red can go, so its blue comes out a hair below nought;
// that is held at nought.
func glowing(kelvin float64) maths.Light {
	red, green, blue := srgb.Light(swatch.Planckian(kelvin))
	return maths.RGB(math.Max(0, red), math.Max(0, green),
		math.Max(0, blue))
}
```

| source | kelvin | red, green, blue |
|---|---|---|
| an oil flame or a candle | 1900 | 2.64, 0.61, 0.00 |
| a filament bulb | 3000 | 1.77, 0.84, 0.27 |
| daylight | 6500 | 1.04, 0.98, 1.04 |

The lamp values are libtheme's, computed. The temperatures are the
usual round figures for each source and were not measured here.

## Putting a fitting and its light together

A lantern on a post, lit:

```go
// lantern is a lit lantern standing at a place: the box, its panels
// glowing, and a point of flame-light in its middle.
func lantern(world *thing.World, at maths.Vec) error {
	flame := glowing(1900)
	made := map[int]*thing.Material{
		13: {Name: "lid", Colour: maths.Grey(0.05)},
		14: {Name: "panel", Colour: maths.Grey(0.1), Glow: flame},
	}
	box := manufacture.Lantern(geom.Vec3{},
		manufacture.DefaultLanternParams())
	things, err := dressed(box, made, maths.Subspace{Origin: at,
		Orientation: upright})
	if err != nil {
		return err
	}
	for _, one := range things {
		world.Add(one)
	}
	world.Lamp(emit.Lit{At: at.Add(maths.V(0, 0.15, 0)), Beyond: 0.2,
		Strength: flame})
	return nil
}
```

`Glow` makes the panels appear luminous. `emit.Lit` illuminates the
surrounding scene. Without glow the source looks dark; without an
emitter, bright panels do not light the ground.

A pendant over a desk, hung from a point on the ceiling:

```go
// hanging turns the manufactory's up, z, to straight down.
var hanging = maths.About(maths.V(1, 0, 0), math.Pi/2)

// pendant is a lit pendant hung from a point on the ceiling: the cord
// and shade, and a cone of bulb-light down from inside the shade.
func pendant(world *thing.World, ceiling maths.Vec) error {
	bulb := glowing(3000)
	made := map[int]*thing.Material{
		11: {Name: "cord", Colour: maths.Grey(0.03)},
		12: {Name: "shade", Colour: maths.Grey(0.6), Glow: bulb},
	}
	piece := manufacture.PendantLight(geom.Vec3{},
		manufacture.DefaultPendantParams())
	things, err := dressed(piece, made, maths.Subspace{Origin: ceiling,
		Orientation: hanging})
	if err != nil {
		return err
	}
	for _, one := range things {
		world.Add(one)
	}
	down := maths.V(0, -1, 0)
	world.Lamp(film.Spot{At: ceiling.Add(down.Scale(1.25)), Aim: down,
		Strength: bulb.Scale(4), Cone: 50 * maths.Degree,
		Soft: 15 * maths.Degree})
	return nil
}
```

The spot sits just below the shade's rim, at 1.25 under the ceiling, so
the shade does not shadow its own light.

## Night exteriors: filming in the hills

Night scenes in the hills combine warm flame with early film that may
be nearly blind to it. Plan stock and illumination together before
spending time on a render.

An oil flame is 2.64 of red, 0.61 of green and no blue. Each stock
sees only some of that:

| stock | sees | of a flame of luminance one, it gets | against `smoke` |
|---|---|---|---|
| `smoke` | red, green, blue | 1.08 | the same |
| `ash` | green, blue | 0.31 | 1.8 stops under |
| `soot` | blue | 0 | nothing at all |
| `lantern` | each lamp in its own record | all of it, as orange | |

So:

- **On `soot`, a lantern-lit scene is black.** not dim: black. The
  stock cannot see the only light there is. No rating fixes it.
- **On `ash`, rate the roll up**: `roll.Rated(1.8)`, or carry more
  lanterns. The scene prints, in greys, and the flame itself prints
  paler than the eye expects.
- **For moonlight, use a `film.Sun` at 6500 and little strength.** all
  three early stocks see it. This is the night the hills were filmed
  by: a white light from one side, the lanterns in shot for show and
  doing almost none of the work.
- **Red is black on `ash` and `soot`.** a red banner lit by anything
  prints as a black one. dress the set in blues and greens if the set
  has to read.

The arithmetic is the stock's own: a record's exposure is the mean of
the lamps it sees. doc/design.md has it.

## Bright interiors: the lawyer’s office

A seated subject in a bright office benefits from broad white light
and gentle facial shadows. Visible fittings can then explain where
that illumination comes from.

- **Start with a flood.** `film.Flood{Strength: maths.Grey(1)}` is an
  evenly lit ceiling. On its own it gives a flat, shadowless room in
  which grey ground prints as itself; it was checked.
- **Add pendants in a row over the table** for the look of a room that
  is lit by something. space them about as far apart as they hang
  above the table, so their pools meet. That is a lighting designer's
  rule of thumb, not a measurement.
- **Add sconces on the walls** to keep the corners from going dark.
  the arm reaches out along x, so turn each to face into the room. its
  light is a point lamp at the bulb.
- **Use a spotlight for the witness.** aim the fitting and the lamp
  the same way:

```go
// spotlight is a lit spotlight at a place, aimed at a target.
func spotlight(world *thing.World, at, target maths.Vec) error {
	aim := target.Sub(at).Unit()
	made := map[int]*thing.Material{
		15: {Name: "mount", Colour: maths.Grey(0.05)},
		16: {Name: "head", Colour: maths.Grey(0.1)},
	}
	piece := manufacture.Spotlight(geom.Vec3{},
		manufacture.DefaultSpotlightParams())
	// the fitting's own z, along which its head points, becomes aim
	things, err := dressed(piece, made, maths.Subspace{Origin: at,
		Orientation: maths.LookAt(at.Add(aim), at)})
	if err != nil {
		return err
	}
	for _, one := range things {
		world.Add(one)
	}
	world.Lamp(film.Spot{At: at.Add(aim.Scale(0.45)), Aim: aim,
		Strength: glowing(3000).Scale(30),
		Cone:     15 * maths.Degree, Soft: 4 * maths.Degree})
	return nil
}
```

On film, a lawyer has one hazard a ninja does not: the suit. A navy
suit is blue, and `ash` and `soot` see blue very well, so it prints
pale grey. A red tie prints black. On `smoke` and `lantern` both print
as worn.

## Issues recorded in these historical APIs

- The fittings give no light. A fitting that came with its lamp, at
  the right point and past its own glass, would remove most of this
  guide.
- The pendant is made upside down for hanging.
- The lantern has no floor.
- Slayer's GPU panics on a lamp that is not a point lamp.
