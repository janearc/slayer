# Space observatory recipes

Spitzer gets to come too. These templates give it a cryostat and shield
panel, give Hubble a long optical baffle and solar wings, and give Webb an
open segmented primary above a layered sunshield.

The geometry is scenery based on their published layouts. The default
apertures and the Webb shield dimensions and counts come from pinned
references. Body proportions, gaps, layer spacing and display materials are
authored choices in `manufacture/spacecraft-choices.yaml`.

Use `spitzer`, `hubble` or `webb` as a named kind for the default blueprint.
A custom aperture scales the chosen design:

```yaml
recipes:
  infrared telescope:
    kind: space-observatory
    spacecraft:
      design: spitzer
      aperture: {value: 0.85, unit: m}
set:
  things:
    telescope:
      recipe: infrared telescope
      seed: 5
      east: 263850
      north: 2148720
      height: {value: 40, unit: m}
      complexity: quick
```

`height` lifts an individual object above its terrain point. Cameras and
named launch offsets refer to that lifted placement. It is a scene pose,
not an orbit model. Omit it to place an object on the ground as before.

The seed retains the current placement-yaw rule. These blueprints have fixed
structural dimensions rather than growing a different telescope for each
seed. All four complexity levels share the same structural envelope and
change circular tessellation.

After building and fetching the source shelf, render a specimen with
`game run space-spitzer`, `game run space-hubble` or `game run space-webb`.
The PNG goes into `out/`. Each run uses the corresponding YAML in `doc/`
and asks Hillside for a black sky.

Spitzer and Hubble have their optical axes along local +y. Webb has its
primary face toward local -z above an XZ shield. These are construction
frames for placement and viewing; they do not implement tracking or attitude
control.

## Reference values and visual choices

[Caltech/IPAC describes Spitzer][spitzer] with an 85 cm primary. Its
[spacecraft description][bus] identifies the octagonal bus and the solar
panel assembly that also shields the cryogenic telescope.

[NASA gives Hubble a 2.4 m primary][hubble]. The tube, bus proportions and
solar-panel grid in this recipe are authored; the grid does not predict
power production.

NASA's Webb overview (source `webb-telescope-overview` in the
[source catalogue](../sources.yaml)) lists a nominal 6.6 m primary and 18
segments around an empty centre. This recipe fits that declared height;
it does not reconstruct the optical figure, poses or collecting area.

[The Webb sunshield reference][shield] gives five layers and deployed
lengths of 21.197 m and 14.162 m. Its outline, exaggerated gaps, finite
membrane thickness and support structure here are visual approximations.
There is no membrane tension or thermal model.

Colours and the Fresnel index are picked display parameters. A gold-looking
mirror is not a measured gold optical-constant model.

[spitzer]: https://www.spitzer.caltech.edu/mission/the-telescope
[bus]: https://www.spitzer.caltech.edu/mission/the-spacecraft
[hubble]: https://science.nasa.gov/mission/hubble/observatory/design/optics/
[shield]: https://science.nasa.gov/mission/webb/webbs-sunshield/

The source tool fetches and pins the official pages.
`game run sources-verify` holds all seven numerical claims to their archived
text before a Game build.
The page bytes stay on the local source shelf and are not redistributed.

## Geometry and GPU checks

All three templates use the existing triangle-mesh route. Public geometry
checks cover closed meshes, finite bounds, nominal apertures and silhouette
probes at all four details. Webb's empty primary centre is checked separately.

The [Metal test](../gpu/spacecraft_test.go) compares small CPU reference views
with software and native GPU traversal at `quick` detail. It requires actual
native closest-hit replies and reports fallback counts. The image budget and
authored views are in its [fixture](../gpu/testdata/spacecraft.yaml).

The separate spacecraft compatibility file holds authored geometry and
materials at two seeds and four details. It supplements the 144 unchanged
historical records; it does not establish physical accuracy.

Inspected specimen PNGs, mutation testing and the broader Game gates remain
part of this open OBS-2 work order. The full ground-based optical family,
radio arrays and radomes are still separate work.
