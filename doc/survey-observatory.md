# A compact survey telescope

The `rubin` kind uses two reflective figures on a common backing, a short
triangulated frame, a solid convex secondary and a camera between the mirrors. It is a scenery template;
the mirror figures do not implement a calibrated optical prescription.

Rubin's [key numbers](https://rubinobservatory.org/for-scientists/rubin-101/key-numbers)
give the physical 8.4 m blank. Its [mirror description](https://rubinobservatory.org/explore/how-rubin-works/technology/mirrors)
gives the 3.4 m secondary and 5.0 m inner tertiary surface. The physical blank
diameter is distinct from effective collecting aperture.

These three values are pinned in `research/research-derived-data.yml` and
verified against the fetched source bytes. The primary and tertiary have
separate reflective skins over one closed backing volume. A narrow picked
gap keeps their edges distinct for the mesh representation.

The 30 m enclosure, supports, shallow figures, backing thickness and camera
dimensions are authored. Their choices and provenance are in
`manufacture/ground-choices.yaml`. The camera sits below the solid secondary;
its position and housing dimensions remain photographic approximations.

## Make a specimen

Use `kind: rubin` in a scene's recipe or manufactured thing. Placement and
site weather follow the other ground templates. Its default optical pose
is fixed; the scene rotates the complete object at placement.

The specimen book works without terrain:

```sh
game run catalogue -- render rubin --out out/rubin
```

To inspect the optical structure, copy the catalogue plan and set
`cutaway: true`. The output explicitly labels the hidden enclosure and
records the filtered view. The source kind still supplies the complete
object to ordinary scene construction.

The exterior and cutaway have the same physical bounds across details.
A narrower field of view and shorter viewing distance can make a close
study; keep the mount in the picture when reviewing the whole instrument.

## The building has a job too

The expanded facility replaces the generic dome with faceted upper walls,
a sloping roof and the observing slit. A long service wing steps down
beside the enclosure, with roof tiers and separate loading and personnel
doors. Its footprint now extends east of the telescope.

Rubin's [summit-facility description][facility] supplies the functional cues:
maintenance and mirror coating, clean rooms, stepped wind-guiding surfaces
and separation between the main and service buildings.

The 30 m tower, 50 m lower wing and all other building dimensions remain
picked scenery proportions. They are not dimensions measured from the
reference photograph or a surveyed Rubin drawing. The joint gap is a visual clearance,
not an earthquake-isolation calculation.

The cutaway hides the cladding and service building with the enclosure.
The source-sized mirror diameters remain unchanged.

[facility]:
https://rubinobservatory.org/explore/how-rubin-works/technology/summit-facility

## Photographic model

The owner's front view and Rubin's [mount description][mount] guide the
broad double upper ring, paired diagonal stays and suspended camera.
The [mirror layout][mirrors] places the camera between M1/M3 and M2.
M2 has a closed convex visual face; the camera has independent supports.

The upper enclosure is narrower than the white fixed base, with faceted
cladding, vent rows and slit-edge housings. The long white service wing
has sloping sides and an inset outer roof edge on each tier. These are
procedural photographic approximations, not engineering CAD.

The overall mount is a simplified fork; articulation, working shutter
mechanisms and a calibrated optical prescription are not modeled.

[mount]: https://rubinobservatory.org/explore/how-rubin-works/technology/mount
[mirrors]: https://rubinobservatory.org/explore/how-rubin-works/technology/mirrors
