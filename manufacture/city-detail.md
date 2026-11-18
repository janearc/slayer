# City buildings at street level

The city's buildings have three levels. `hull` is a box and a roof.
`quick` adds at most three glazing bands on two faces. `demo`, new on
2026-11-16, models each building to be looked at from the street:
floor-slab edges, recessed windows with sills and mullions on all four
faces, setbacks, cornices and parapets, rooftop plant, and an entrance
on the ground floor.

A set file asks for it per thing with `complexity: demo`. `hull` and
`quick` are unchanged to the bit; the test
`TestCityExistingLevelsKeepTheirGeometry` holds their hashes.

## What was measured first

Measured on next at d0f70ff8 with `TestCityDetailMeasures`, seed 31.

- Faces per building, mean of 400 seeds: rowhouse 20 hull, 79 quick;
  apartment 24 and 96; tower 48 and 120.
- The clustered night city at `quick` pools 1,745,532 faces in 35
  meshes, one per material chunk of 60,000 faces. No building shares a
  mesh with another: every face is unique, and the GPU holds about
  168 MB of triangles and vertices for it (96 bytes a face: a
  three-float4 Triangle and three float4 Metal vertices).
- The night city lights about 710,000 windows (estimated from the
  night faces less the day glazing; the new level counts them).
- The renderer does instance. `shape.MeshGroup` keeps a mesh used by
  several members once (`StoredFaces`), `gpu/instances.go` pools each
  mesh once by pointer, and `gpu/native.go` builds one Metal primitive
  structure per pooled mesh and one instance per member.

  A member costs a 112-byte kernel Instance, a 68-byte Metal instance
  descriptor and its share of two hierarchies: a few hundred bytes,
  against 96 bytes a face for unique geometry before any hierarchy.

## The choice

Unique detailed buildings do not fit. A window with its reveal, sill
and mullion is about 34 faces; 710,000 of them is 24 million faces,
2.3 GB of triangles before either hierarchy.

A bounded set of whole designs fits easily, but every lot was planned
from its own seeded width, depth, storey and floor count, and its
envelope bounds the lot, the trees and the night glow. Replacing it
with one of a few dozen designs breaks that, and repeats visibly.

So the city uses shared modules, chosen because the numbers favour
them:

- A facade strip is one storey of one face: a wall with `n` window
  openings at a fixed bay pitch, their reveals, sills and mullions.
  It is modelled once per family and bay count, a few dozen meshes in
  all, and placed once per face per floor. The strip has the wall's
  material, so one mesh serves all four wall colours.
- Rooftop plant (lift overruns, cooling units, tanks, masts) and
  chimneys are modelled once and placed.
- What differs per lot stays unique and small: the core, one slab
  band per floor (its height takes up the lot's own storey), corner
  piers that take up the face's remainder, cornices, parapets and the
  glass. Each pane keeps the seeded lit or unlit draw the night city
  already uses, so night windows do not change their pattern.

Distance-chosen detail was the third option. The city is baked once
without a camera, so it would need a camera in manufacture or a second
city thing per ring; a scene can already choose its level per thing.
It stays possible later.

## The envelope

Every part stays inside the envelope planning gave the lot. Walls are
set in by the facade projection so slab edges, sills and canopies end
at the old box faces. A flat roof's top storey is the plant storey:
its deck carries a parapet and the plant, all below the old roof top.
A tower's crown is its plant room. A rowhouse's chimney stands on its
gable's slope, below the ridge.

## Cost

The new level's numbers, measured on the same seed, are in the
branch's report beside its pictures:
the `REPORT.md` in the 2026-11-16 city-detail folder under `renders/frames`.
