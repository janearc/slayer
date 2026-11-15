# Space observatory recipes

Spitzer gets to come too. These templates give it a cryostat and shield
panel, give Hubble a long optical baffle and solar wings, and give Webb an
open segmented primary on a tower above a layered sunshield, with its bus,
solar panels and antenna under the shield on the sun's side.

The geometry is scenery based on their published layouts. The default
apertures, the Webb shield dimensions and counts, and the place and size of
Webb's secondary come from pinned references. Body proportions, gaps and
layer spacing are authored choices in `manufacture/spacecraft-choices.yaml`.
Display materials are authored too, except Webb's mirrors and sunshield,
which `manufacture/webb-surfaces.yaml` makes of gold and aluminium.

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
lengths of 21.197 m and 14.162 m. Its outline is a kite with its edges
drawn inward, by eye from NASA's picture of it. That outline, the
exaggerated gaps, the finite
membrane thickness and support structure here are visual approximations.
There is no membrane tension or thermal model.

Webb is laid out as [McElwain et al. 2023][ote] lays it out, section 3: the
spacecraft bus and solar array on the sun's side of the shield, the
telescope above it on a tower, and the instrument module on the primary's
back. Its secondary stands 7169.0 mm in front of the primary's vertex and
is 738 mm across, the paper's table 2 (source
`mcelwain-2023-webb-telescope`).

No paper in hand gives the size of the
bus, the tower or the instrument module, so those are authored, and the
module is one closed box: the four instruments are not made. The sun side,
with its two pallets, its booms, its row of five solar panels, its
radiators and its flap, is laid out by eye from NASA's picture of it.

For Spitzer and Hubble, colours and the Fresnel index are picked display
parameters. Webb's mirrors are gold and its sunshield aluminium, each a
conductor by its published optical constants: Johnson and Christy's
measurements for gold, Rakić's fitted model for aluminium. The mirror's
overcoat and the doped silicon on the sun's side of the two hottest shield
layers are not drawn, and Webb's structure keeps a picked dark grey.

[spitzer]: https://www.spitzer.caltech.edu/mission/the-telescope
[bus]: https://www.spitzer.caltech.edu/mission/the-spacecraft
[hubble]: https://science.nasa.gov/mission/hubble/observatory/design/optics/
[shield]: https://science.nasa.gov/mission/webb/webbs-sunshield/
[ote]: https://arxiv.org/abs/2301.01779

The source tool fetches and pins the official pages and the paper.
`game run sources-verify` holds all nine numerical claims to their archived
text before a Game build; the paper's two are held to its page 6.
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
