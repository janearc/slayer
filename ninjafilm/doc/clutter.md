# Dressing a hillside

A few trees establish the hillside; leaf litter, grass, and undergrowth
make it feel like more than a set. This guide assembles those details
from the older manufactory into a Lamb world, including the adaptations
that revision required.

This is a historical guide to `manufacture` at `3455095`, on
`slayer-grav-mfg` and merged into `trees`. Dimensions, counts, and failures
were measured on 2026-10-06. For the later seeded construction design,
see [the manufactory](../../doc/manufactory.md).

## What the manufactory is, at this commit

At this revision, each kind has its own function taking a base point
and parameter struct and returning triangles:

```go
piece := manufacture.Fern(geom.Vec3{}, manufacture.DefaultFernParams())
```

This version has no shared kind lookup or construction entry point.
Callers provide placement and materials; the following sections show how.

## What is on the shelf for the wilderness

| piece | call | wide by tall | triangles | material ids | same every call |
|---|---|---|---|---|---|
| fern | `Fern(base, DefaultFernParams())` | 0.78 by 0.46 | 64 | 6 | no |
| grass, a tuft to the ankle | `Grass(base, AnkleGrassParams())` | 0.06 by 0.25 | 16 | 7 | yes |
| grass, a tuft to the waist | `Grass(base, WaistGrassParams())` | 0.06 by 1.20 | 24 | 7 | yes |
| shrub | `Shrub(base, DefaultShrubParams())` | 0.72 by 0.70 | 36 | 8 | no |
| loose leaves | `Foliage(at, 40, 0.05, rng)` | 0.33 by 0.14 | 80 | 1 | with your `rng`'s seed |
| conifer | `Conifer(base, DefaultConiferParams())` | 4.0 by 17.9 | 23,648 | 2 | yes |
| whorled conifer | `ConiferProper(base, DefaultProperConiferParams())` | 5.5 by 10.0 | 4,876 | 2, 25 | yes, by its `Seed` |
| spruce | `OrganicTree(base, DefaultSpruceParams())` | 9.9 by 12.4 | 1,296 | 2, 25 | no |
| birch | `OrganicTree(base, DefaultBirchParams())` | 6.9 by 12.0 | 856 | 2, 25 | no |
| oak | `OrganicTree(base, DefaultOakParams())` | 13.0 by 13.5 | 1,096 | 2, 25 | no |
| larch | `OrganicTree(base, DefaultLarchParams())` | 5.5 by 13.2 | 1,060 | 2, 25 | no |

Sizes are in the manufactory's units, which the defaults treat as
metres. material id 2 is wood and 25 is needles.

Not on the shelf: rocks, logs, deadfall, moss, mud, snow. A hillside
wants all of them. Until the manufactory makes them, a rock is a lamb
sphere pushed half into the ground.

## Five integration details

**1. up is z.** the manufactory builds upward along z. lamb's up is y.
turn every piece as you place it; the code below does.

**2. A triangle carries a number, not a material.** `Material` on a
triangle is one of the ids in the table. It means nothing to lamb. you
give each id a `thing.Material`.

**3. ferns, shrubs and the four organic trees are different every
call.** they draw from a generator seeded by the clock. Two calls give
two ferns, and so do two runs of your program.

For animation, build this set once before the take and reuse the world.
Rebuilding clock-seeded geometry inside the frame loop changes its
shape in every frame.

**4. grass will not mesh as it is made.** each blade is two triangles on
the same three points, back to back, and lamb's `shape.NewMesh` refuses
the pair as a closed piece that is inside out.

Keep one triangle from each pair. Lamb intersects it from both sides,
so this removes a degenerate closed mesh without losing visibility.

```go
// blades is grass with one face to a blade, which lamb will mesh.
func blades(grass []geom.Triangle) []geom.Triangle {
	var kept []geom.Triangle
	for index := 0; index < len(grass); index += 2 {
		kept = append(kept, grass[index])
	}
	return kept
}
```

**5. The organic trees hang below their own base.** their lowest
branches reach under the point you stood them on: the spruce by 2.05,
the birch by 0.89, the oak by 3.82, the larch by 0.47.

Stood on the ground, they go through it. either lift the tree by that
much and accept a bare trunk, or use the whorled conifer, which stands
on its base and is the same every time.

## From triangles to things

This turns one piece into lamb things, one for each material it uses,
standing at a place. It is the code that was run for the table above.

```go
// upright turns the manufactory's up, z, to lamb's up, y.
var upright = maths.About(maths.V(1, 0, 0), -math.Pi/2)

// dressed is a piece from the manufactory as lamb things, one for each
// material it uses, standing at a place.
func dressed(piece []geom.Triangle, made map[int]*thing.Material,
	place maths.Subspace) ([]thing.Thing, error) {
	corners := map[int][]maths.Vec{}
	for _, face := range piece {
		for _, corner := range face.V {
			at := place.Out(maths.V(corner[0], corner[1], corner[2]))
			corners[face.Material] = append(corners[face.Material], at)
		}
	}
	var things []thing.Thing
	for id, points := range corners {
		material, known := made[id]
		if !known {
			return nil, fmt.Errorf("no material for id %d", id)
		}
		faces := make([][3]int, len(points)/3)
		for index := range faces {
			faces[index] = [3]int{3 * index, 3*index + 1, 3*index + 2}
		}
		mesh, err := shape.NewMesh(points, faces, nil, false)
		if err != nil {
			return nil, err
		}
		things = append(things, thing.Thing{Shape: mesh,
			Material: material})
	}
	return things, nil
}
```

Assign a material to each generated ID. This historical generator does
not supply the fern’s colour.

```go
made := map[int]*thing.Material{
	1:  {Name: "dead leaf", Colour: maths.RGB(0.35, 0.22, 0.08)},
	2:  {Name: "bark", Colour: maths.RGB(0.20, 0.14, 0.09)},
	6:  {Name: "fern", Colour: maths.RGB(0.10, 0.38, 0.08)},
	7:  {Name: "grass", Colour: maths.RGB(0.25, 0.45, 0.10)},
	8:  {Name: "shrub", Colour: maths.RGB(0.08, 0.28, 0.07)},
	25: {Name: "needles", Colour: maths.RGB(0.04, 0.20, 0.08)},
}
```

Those colours are a starting point picked by eye. No leaf was measured.

## Scattering

One fern is a houseplant; a hillside needs placements. `forest` begins
with a grid, jitters positions, rotates specimens, and varies scale.
The seed reproduces the layout independently of any clock-seeded
geometry supplied to it.

```go
// strewn is one piece, grown afresh for each place, on a grid.
func strewn(world *thing.World, seed uint64, side int, spacing float64,
	made map[int]*thing.Material,
	grow func() []geom.Triangle) error {
	places, err := forest.Placements(seed, side, spacing, 0.9)
	if err != nil {
		return err
	}
	for _, placed := range places {
		turned := maths.About(maths.Up, placed.Turn()).Then(upright)
		things, err := dressed(grow(), made, maths.Subspace{
			Origin: placed.At(), Orientation: turned,
			Scale: placed.Size()})
		if err != nil {
			return err
		}
		for _, one := range things {
			world.Add(one)
		}
	}
	return nil
}
```

`forest.Placements(seed, side, spacing, jitter)` takes a side of one
place or more, a spacing above zero, and a jitter from zero to one.

On the `fixes` branch the side stops at 256; on `trees` it has no top,
and a huge one runs out of memory. A jitter near one hides the grid. The
grid's corner is at the origin, so move the world or the places if your
set is centred.

## A hillside, in layers

Build in layers: ground, ground cover, then taller plants. Give each
layer a separate seed so its placements do not line up with the others.

| layer | piece | spacing | why |
|---|---|---|---|
| ground cover | ankle grass | 0.15 | a tuft is 6 cm wide; closer than this is a lawn |
| the rough | waist grass | 0.6, in patches | long grass grows in clumps, not evenly |
| the shade | fern | 1.2 | under the trees, where the grass thins |
| the edge | shrub | 3 | along the tree line and round rocks |
| the litter | loose leaves | 0.8 | under every tree, in the hollows |
| the wood | whorled conifer | 6 | a tree's crown is 5.5 wide; closer and they merge |

These spacings are visual starting points, not survey data. Adjust
them after inspecting the render.

The cost adds up. A 20 by 20 patch of ankle grass is 400 tufts and 3,200
triangles once one face is kept. forty whorled conifers are 195,000. One
default `Conifer` is 23,648 on its own, all in one material, so prefer
the whorled one.

## How it films

- On `soot`, which sees blue only, green leaves print black. The whole
  hillside goes dark and the sky goes white.
- On `ash`, which sees green and not red, foliage prints light and a
  red sash against it prints black.
- On `lantern` and `pollen` it is green.

doc/filming.md is how to film it. doc/lighting.md is how to light it.

## Issues recorded in this revision

The examples work around the following generator limitations. Their
long-term fixes belong in `manufacture`; check later revisions before
reusing a workaround:

- Grass blades made of two faces on the same points
- Ferns, shrubs and organic trees seeded from the clock
- Organic trees that reach below their base
- `Conifer` giving its trunk and its needles the same material
- No rocks, logs or deadfall
