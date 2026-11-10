# The keyhole shot: what the scene format needs

2026-11-08. A plan, unreviewed; each piece goes to review before it
merges.

## The shot

One scene, in YAML. The camera starts in the cloud sea at the summit
observatory, with its guide-star beams up through the mist.

It rises out of the mist, through the sky, into orbit, where a
Keyhole-inspired reconnaissance satellite sits. Then the view turns to
look back down through the satellite's optics, at the cloud sea and the
dome.

Everything the shot needs must be said in the scene's YAML. Whatever is
not exposed there is a gap to close, not a flag or a constant to set by
hand.

## Who does what

The split agreed with the observatory work's owner on 2026-11-08 is
written in full in her coordination notes; in short:

- this branch, `keyhole`, from `feat/observatory-sites` at d270f3c:
  schema and reader parity, a look per camera move, the instrument eye,
  a spacecraft's orbit and attitude, the date, and weather beyond a
  terrain patch.
- hers: OWL and the other factory assemblies, vegetation, the factory
  kinds and catalogue, and the existing guide-star pointing and
  launchers.
- shared, in small separate commits announced first: `set/recipes.go`,
  `set/world.go`, `set/scene.go`, `cmd/hillside/scene.go`, the source
  manifests and these docs. New kind names in the schema's enums are
  hers; parity changes are this branch's, kept apart.
- the GPU's field transport is not touched without its owner.

## The gaps, in order

1. **The schema behind its reader.** Done on this branch, as schema
   only: `set/ecosystem.schema.json` now takes every key the reader
   takes, and its parity test omits nothing.
2. **What the schema took, and where.** A film's print and rig and a
   camera's eye (6acd458); a stand's `within`, a set's `bounded_box`,
   `assets` and `texts`, and the `hud` (36db504); the `world` (aee376f);
   the `planet` (e08a6e9).
3. **What parity holds.** Key names are held equal; value limits,
   required keys and cross-references are the reader's, and whether a
   scene renders is the renderer's.
4. **A look per move.** Done on this branch (6a1a76a): a move may say
   its own look, a target that travels from one point to another as the
   camera does. Looks relative to the scene's objects, not only absolute
   metres, are still to come.
5. **The frame.** Agreed with the observatory work's owner before orbit
   and weather. The scene map is east, north, up; the maths space is
   east, up, south; terrain staging rebases to its west, north and lowest
   point.
6. **One frame across the climb.** A climb from metres to hundreds of
   kilometres needs one explicit common frame across that boundary.
7. **What an eye is.** An instrument eye looks through a spacecraft's
   aperture and focal length. The factory's mirrors and lenses are
   display stand-ins, so the eye must say what it models (a pupil and a
   focal length, its diffraction limit), not trace a compound telescope.
8. **A spacecraft in orbit.** The designs are Hubble, Webb and Spitzer,
   placed on the terrain by east, north, height and a compass heading.
9. **What a spacecraft needs.** An orbit, or a place above the planet,
   and an attitude, so it can point down. A Keyhole-inspired design is
   the Hubble-sized bus with its instrument looking down.
10. **The world, not a patch.** See the design note below. Weather is
    `prescribed-mist` layers whose extent is the terrain patch, and the
    terrain is a cropped patch of an elevation model. From orbit both
    end at the crop's edge.
11. **The date.** A scene has a time of day and a place but no date, so
    the sun and moon are placed by hand. A date would place them, while
    explicit authored directions and tint stay as they are.

One gap is not the schema's: a single move from metres to hundreds of
kilometres in 32-bit floats on the GPU needs rendering relative to the
camera, or the far end shakes. It is named here so it is not forgotten.

## Design note: the world, rendered by what can be seen

2026-11-08: there should be no 1.6 km limit. A scene knows how big
its world is, and renders only what can be seen, at the detail that can
be seen.

At the ground every branch shows. At 100 m the trees still have their
detail. At 10 km a tree is a dot, so the same budget draws tens of
thousands of them and much more terrain. Higher still, a forest is a
pattern.

As the view widens, each thing in it is smaller, so less detail is
needed for each; that is not cheating, it is what the picture holds.

For the scene format that means:

- terrain names its source and the world's extent, not a crop; the
  renderer streams it, finer near the camera and coarser with distance,
  from elevation data we have or can get.
- a forest is a density over a region, not a count in a square; the
  renderer draws branches, then trees, then dots, then texture, by each
  tree's size on the picture.
- a detail budget the renderer spends by size on the picture.

Terrain and vegetation are the observatory work's owner's or shared, so
this note goes to her as a design proposal before anything is built.

## Proposal: the frame and the eye

Agreed with the observatory work's owner on 2026-11-08, amended by her
review of b57fecd (her notes, slayer-world-frame-notes.txt). These are
meanings, not a promise to build every effect now; the first slice is
the smallest useful one. Nothing here is built yet.

**One anchor, stable for the whole scene.** The world is placed by one
declared anchor, not by a render crop's origin.

The staged west, north and lowest depend on which elevation window is
loaded, and recomputing the planet from the visible window would move
the world as the camera moves.

The anchor states matching projected and geographic coordinates, its
physical elevation, and the reference radius and height datum. Its
centre and basis do not change through a sequence.

The existing staged coordinates are adapted into that frame by an
explicit adapter, so old scenes read as they did. `location` alone does
not say which raster sample has that latitude and longitude, and
`lowest` is not the site's elevation; neither is used as if it were.

A spherical planet of an authored radius is a valid first model, when
its radius and its vertical datum are stated. An elevation model's
orthometric heights are not ellipsoid heights, and nothing here claims
survey geodesy.

**Altitude and nadir are radial.** Altitude above the reference surface
is along the position's radial normal, not the scene's y. Nadir points
from the craft to the planet's centre.

Above the anchor these agree with local up and down; after horizontal
travel they do not. An azimuth and an elevation are taken in the
tangent basis at the craft's subpoint, with roll declared.

A craft held above a site is a positioned spacecraft, not an orbit: it
is called an orbit only when its motion in time is modelled.

**Every position is rebased together.** Before anything is made 32-bit,
one render origin is subtracted from every position: mesh and group
placements, actors and cameras, finite lights and beams, planet centres
and weather and density queries.

Directions and sizes are not translated. Moving only the camera's matrix
does not rebase what the shaders hold.

Rendering relative to the camera is necessary but proves nothing alone:
a globe's radius less its height, and grazing shadow intersections, keep
their stable predicates and their budgets.

A planet-sized shape says whether its local terms need split precision.
The GPU's field transport is not touched without its owner.

**The eye is an optical port and a named approximation.** An instrument
eye looks out of a named pupil on the instrument, with an optical axis,
carried by the instrument's whole pose; the craft's origin need not be
its pupil.

The eye says how it treats the instrument's own housing, so it does not
see its own stand-in objective, rather than taking the craft out of the
world.

Its blur is a named point-spread approximation, with its wavelength
handling and its conversion from angle to pixels said.

1.22 wavelength over diameter is the angular radius of the first dark
ring of an unobscured circular pupil; it is not a Gaussian's sigma, a
full width at half maximum or a blur width.

An obscured telescope's spread is not an unobscured pupil's;
obscuration, aberrations, the atmosphere, detector sampling and
vignette do not follow from the diameter.

The focal length and the frame are in agreeing units, and the eye's
field drives both the view's rays and the detail chosen: a narrow field
needs finer distant terrain than a wide one at the same height.
Vignette is sourced geometry or a display filter picked and named as one.

**The spacecraft.** A thing of a spacecraft recipe may say `altitude`,
radial above the reference surface, instead of a height on the terrain,
and `points` (`nadir`, or an azimuth, an elevation and a roll in its
subpoint's tangent basis) instead of a compass heading.

**Streaming stays apart from the crop.** The world's extent and a
source's coverage are different: a source can run out while the planet
and a chosen coarse fallback go on.

Detail is chosen by the lens, the projected error and conservative
bounds, not altitude alone. A tree is known by its spatial cell and
seed, not its place in the visible list. The world caps all live sources
and decoded bytes together.

## How it is checked

Each piece: the schema and reader agree (the parity test); a scene that
uses it is read; a scene that misuses it is refused, by the schema where
it can say so and by the reader where only it can.

Small scoped tests, two workers, no mutation campaign; runs held
whenever the machine's owner has called a hold.
