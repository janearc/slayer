# A compact survey telescope

The `rubin` kind uses two reflective figures on a common backing, a short
truss, an annular secondary and a camera housing. It is a scenery template;
the mirror figures do not implement a calibrated optical prescription.

Rubin's [key numbers](https://rubinobservatory.org/for-scientists/rubin-101/key-numbers)
give the physical 8.4 m blank. Its [mirror description](https://rubinobservatory.org/explore/how-rubin-works/technology/mirrors)
gives the 3.4 m secondary and 5.0 m inner tertiary surface. The physical blank
diameter is distinct from effective collecting aperture.

These three values are pinned in `research/research-derived-data.yml` and
verified against the fetched source bytes. The primary and tertiary have
separate reflective skins over one closed backing volume. A narrow picked
gap keeps their edges distinct for the mesh representation.

The 30 m enclosure, supports, shallow figures, backing thickness, secondary
opening and camera dimensions are authored. Their choices and provenance
are in `manufacture/ground-choices.yaml`. The camera fits through the opening;
neither the opening nor the housing claims a measured instrument dimension.

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

## Checks beside the picture

Every detail has closed finite meshes and the declared physical envelope.
Independent probes check all three mirror spans, the primary and secondary
openings, the tertiary surface, one backing volume and the camera housing.

In the original optical-only revision, the first 208 geometry records
remained unchanged. Its eight survey records had separate producer hashes.

CPU, software Metal and native Metal comparisons cover the enclosed template.
A separate enclosure-free numerical view checks the exposed assembly.
The cutaway uses the same ordinary mesh route and retains native evidence.

The original optical-only revision passed its source, geometry and build
checks. Its 93-case survey/cutaway mutation scope caught 92 faults, with
one unbuildable case and no survivors or timeouts. Those results predate
the facility expansion; broader module gates remain open.

The [feature backlog](feature-backlog.md) keeps those distinct from a rendered
picture or a whole-release approval.

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
The three source-sized mirror surfaces, their backing and camera remain
the same; their geometry checks still apply.

The earlier survey appearance records and pictures describe the old
generic enclosure. Keep their provenance when replacing those eight
records with the intentionally changed building. Older mutation results
above predate this facility expansion.

[facility]:
https://rubinobservatory.org/explore/how-rubin-works/technology/summit-facility
