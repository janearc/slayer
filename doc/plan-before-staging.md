# Planning a set before it is staged

This is a design. Nothing in it is built yet.

## The problem

A set file is small. `sets/futian-shore.yaml` is 130 KB, 237 kinds on
1,364 lines. Staged, it is 108,083 things, 703,667 instances, 4.3 billion
faces referenced, 5,161 lamps, 72 bent plumes, and a process of 14 GB.
Staging takes two and a half minutes before the first ray, and that does
not change with the size of the picture: drawn at 16 by 9, the whole run
is 2 minutes 48 seconds, of which the 144 pixels are a few seconds.

The cause is the order of `staged` in `session/scene.go`. It places
everything the set describes (`stage.place`, line 202): terrain, light,
every stand and every thing in it, built and with its search tree. Only
then does it work out the camera (`stage.eye`, line 235). Nothing can be
decided by what the picture needs, because the picture is not known
until everything has been made.

So the size of the process is the size of the set, not of the picture. A
set ten times larger does not fit, and cutting the frame into tiles does
not help: each tile stages the whole set again. Dark City cut into 2 by 2
up to 16 by 16 tiles peaked at 13.4 to 14.2 GB at every size.

## What changes

The camera and the picture size are read first. Then a plan is made from
the set file and the camera alone, before anything is manufactured. The
plan says, for each thing the set asks for, one of three things:

- **drop.** It is outside every ray the picture can cast, or it is
  smaller than a pixel and too faint to change one.
- **average.** It is many small things that together cover a few
  pixels: 30,358 mangroves along a shore, 5,161 lamps in windows. It is
  stood in for by one thing that carries what they carry together: a
  canopy with their mean height, cover and colour; a few clustered lamps
  with their summed light, placed at their light-weighted middle.
- **build.** It covers enough pixels to be seen. It is made at the level
  of detail its pixels need, as the leveller already chooses for towers.

Staging then makes only what the plan builds, and the stand-ins for what
it averages. What it drops is never read past the line that names it.

## What the plan needs from each kind

For each entry in a set, without manufacturing it:

- **a box** that holds everything it could make, from its recipe: a
  tower's footprint and height, a stand's region and its tallest design,
  a plume's reach and lid. A box may be too large; it must never be too
  small, or a ray misses a thing that is there.
- **a count**, for a stand: how many members its rule places, from the
  rule, not by placing them.
- **what it gives off**, for anything with lamps: how many, and their
  summed power, from the recipe.
- **what an average of it is**: the stand-in a kind offers when many of
  it share a pixel. A kind with none is built.

Today no kind can say any of this without being made. A kind is a
`maker` (`manufacture/kinds.go:82`), and a maker has one method,
`made(from seed) (good, error)`: the only way to learn a kind's size is
to build it. `Bounds` exists on what was made (`manufacture/burner.go:72`
is one), not on the recipe. So the base maker changes.

Every maker answers, beside `made`, how big
what it would make is, and whether a sensor sees it, from its recipe's
numbers and without making it:

```go
type maker interface {
	made(from seed) (good, error)
	// extent is the box that holds everything made would make from
	// seed, in the thing's own frame. It may be too large, never too
	// small.
	extent(from seed) (maths.Box, error)
	// seen is how a sensor at a place sees what made would make from
	// seed, standing at a place: whether any pixel's ray can meet it,
	// and how many pixels across it spans.
	seen(from seed, at maths.Subspace, by Eye) (Seen, error)
}
```

`Eye` is the sensor as the plan needs it: where it stands, how wide it
sees and the picture's size in pixels. It exists already as
`cities.Eye` (`manufacture/cities/level.go`), with `Sees` and `span`,
built from the set's shot before staging (`session/levelled.go`,
`seen`); only towers use it today. It moves out of `cities` so every
kind can take it.

`seen` has one answer for every kind, from `extent`: the box placed and
projected through the eye. A kind whose box is a poor guide to what it
shows (a mast, a cable, a lamp brighter than its size) answers for
itself. `extent` has no such default: a kind that cannot say how big it
is without making itself is the greedy step this design removes, so
every maker says.

There are 84 makers. Futian asks for 25 kinds; those come first. Until
every maker answers, one that does not is made once, at its coarsest
level, to measure it, and the plan says so on its line, so the
remaining greedy steps are listed and counted, not hidden.

Light is a third question, asked of what is seen or averaged: how many
lamps it would carry and their summed power. It comes after extent and
seen.

## Every object is manufactured

For the plan to ask every object, every object has to come out of the
manufactory as the one base type. Today these do not; the session makes
them itself while staging, and each moves into manufacture as a maker,
behind the same `made`, `extent` and `seen`:

- the sea
- plumes
- text
- ecosystems
- lasers
- terrain
- the weather the set prescribes: rain, mist, cloud, soot, rust
- the rigs: `GroundRig`, `SealRig`, `LivingLight`

Once they are makers, staging has one path: the plan asks each object
in the set, and builds, averages or drops it. The session keeps no maker of
its own.

A stand's count is already in the set (`set.Stand.Count`), and where its
members stand is chosen by its seed over the ground
(`session/scene.go`, `scattered`) without making them. So a stand's
places can be planned from the set and the terrain, and its members'
boxes from their kind's answer.

Which kinds Futian asks for, and what each must give, is the first
thing to write down.

## How a verdict is made

The box is projected through the camera. Its width in pixels decides:

- no pixel of the picture sees it, and nothing it lights or shadows is
  seen: drop.
- it covers less than a pixel, and so do its neighbours of the same
  stand: average them together, by stand, so a stand of mangroves is one
  canopy at a distance and its trees nearer than that are built.
- it covers more: build, at the level for that many pixels.

Light is judged the same way, by what it adds to a pixel rather than by
where it is. A lamp whose light at the nearest point it could reach is
under a set fraction of the brightest light in the picture is averaged
into a cluster with its neighbours.

Shadows and reflections reach outside the frame: a tower behind the
camera can shade the street in front of it, and the sea mirrors the sky
over the camera. The plan widens the view it judges by along the sun's
direction and by the sea's mirror before it drops anything; what lies
only in those widened views is averaged, not built, since it is seen
only through a shadow or a reflection.

## Occlusion cascades

Size alone is not enough. The first plan of Futian (`slayer assess
sets/futian-shore.yaml --plan`, 21 November) asks for 30,995 things:

- at 16 by 9: 182 built, 14,744 averaged, 16,069 dropped;
- at 480 by 270: 14,865 built, 61 averaged, 16,069 dropped.

The dropped are those out of view. At any real size nearly everything in
view covers more than a pixel, so size keeps them all. What keeps them
out is what is in front of them: a camera facing a brick wall needs none
of the 150,000 houses behind it.

So the plan walks from the camera outward, as radiance cascades do, but
carrying what blocks light rather than the light itself:

- **The set is the tree.** Its things and stands are the top nodes, each
  with a box from `extent`. A node's children (a stand's members, a
  superblock's buildings) exist only when a pass opens it. Members are
  placed by their own numbers (`session/members.go`), so one region of a
  stand opens without placing the rest.
- **Cascades by depth.** Cascade 0 is a thin shell before the camera; each
  next one reaches four times as far, and its cells are four times as
  wide, so a cell of any cascade covers about one pixel. A cell keeps one
  number: how much light still passes it.
- **Front to back.** A pixel's openness is the product of its cells, near
  to far. A cascade is filled only for pixels still open; a pixel under a
  256th open is closed, since a picture cannot show what is behind it.
- **What a node gives a cell.** A node larger than the cell it falls in
  is opened, or built if it has no children. A node smaller adds its
  coverage at that cell's size to the cell's average. This is the third
  question each object answers, beside `extent` and `seen`: how much of a
  cell of a given width it blocks.

  A wall fills its cells; a crown fills
  part of them; a mast blocks almost nothing once its cell is wider than
  it is thick. It must err toward blocking less, never more, or a thing
  that is seen is lost.
- **What reaches past it.** A thing hidden from the camera may still
  shade or light what is seen. The same cascades are built from each
  strong light, the sun first; a hidden node that the light's cascades
  find between a seen surface and that light is averaged as a
  shadow-caster, not dropped.

The code is `improv/cascades/flatland`, the two-dimensional proof of
radiance cascades: its `interval` already carries transmittance beside
light, its `merge` joins far intervals to near ones, and its `Plan` lays
the cascades out four times further each.

It moves out of `improv` and
into three dimensions as the one cascades library: the camera's
occlusion first, and then fast light's lamp store (`gpu/reach.go`),
which today fills a uniform grid by tracing every cell against the whole
resident scene.

## What the plan is

A file, written before staging and read by a person: each entry of the
set, its box, how many pixels it covers, its count, its verdict and the
reason. `slayer draw` stages from it. Running the plan on its own costs
seconds, so the same set can be planned at 16 by 9 and at 3840 by 2160
and the two plans read side by side.

## What this is not

- It does not change a set file. The set still says everything it says;
  the plan decides what a given picture needs of it.
- It does not lose anything that a pixel would show. The thresholds are
  numbers, stated in the plan file with each verdict, and a draw with
  every verdict set to build is the reference the planned draw is
  compared against.
- It is not a cache of built geometry across draws. That may come later;
  the plan is what makes it possible, since it says what each draw needs.

## Order of work

1. For each kind Futian asks for: can it give a box, a count and its
   light from its recipe alone? The answers go in this file.
2. The plan, printed and not used: run on Futian at 16 by 9 and at 480
   by 270, and read.
3. Stand-ins: the canopy for a stand, the lamp cluster.
4. Staging from the plan.
5. The planned draw against the every-build reference, at both sizes.
