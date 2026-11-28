# Bioforms

A bioform is a living thing the manufactory makes: an animal built as
geometry, with a skeleton or a set of controls that moves it. Each animal has
one page that says what it is, what it was built from, how it is put
together, how to use it in a set, and what is known to be wrong.

| Animal | Page | In one line |
| --- | --- | --- |
| Allosaurus | [allosaurus](allosaurus/README.md) | A walking theropod of 7.4 m, kinematic, used by the cornice gargoyles. |
| Bay seal | [bay-seal](bay-seal/README.md) | A harbor seal at its bottling pose, a static profile study. |
| Hatzegopteryx | [hatzegopteryx](hatzegopteryx/README.md) | A giant azhdarchid of 11 m span; the stone gargoyle kind and a motion study. |
| Humboldt squid | [humboldt-squid](humboldt-squid/README.md) | A 0.80 m mantle squid with eight arms and two tentacles, used as a gargoyle. |
| Orca | [orca](orca/README.md) | An adult cow bound to a skeleton: swim loop, bones alone, spy-hop. |
| Pliosaur | [pliosaur](pliosaur/README.md) | A Pliosaurus kevani composite with four flippers; skull only is fossil. |
| Purussaurus | [purussaurus](purussaurus/README.md) | A giant caiman after UFAC 1403, an authored Go construction. |
| Walrus | [walrus](walrus/README.md) | A bull on land, hide over a skeleton, with static poses and representations. |

[Authored anatomy](authored-anatomy.md) is a separate page: the opt-in
anatomy of an individual stage and the small authored populations of heard
scenes.

## how to read the models

the model pages distinguish measured landmarks, related-species references
and authored soft tissue. they describe geometry and motion separately.
a static profile, a posed skeleton or a repeated swim loop makes different
claims. see [authored anatomy](authored-anatomy.md) for the constructor
and each model page for comparisons, source limits and earlier builds.

## Binding and posing a surface

Use `bioforms/skeleton` for the hierarchy, bone shapes and joint turns.
Add bones parent first, in rest-space metres. The pose gives each named
joint its turn in rest axes. A child inherits its parent's motion and
turns about its own head. `Joints` can apply authored angular limits.

`Skeleton.BindSkin` takes rest vertices and a rule assigning weights to
bone indices. The maker can select different bones or reaches along a
trunk, limb, wing or fin. `Holders` and `Nearness` offer weights from
bone centre lines. The binding is reusable across poses and holds a
snapshot of the hierarchy, points and weights.

`Skin.Pose` gives the posed surface and named rigid frames.
`SkinPose.Points` preserves the original vertex order. Use the same
face indices and the original coordinates for paint, even where the
geometry has separate relief. Surface attachments can use the same face
indices and barycentric weights. Rigid attachments can use
`SkinPose.Frame`.

The orca uses `LinearSkin`, the existing linear blend of bone positions.
`RigidSkin` instead blends unit dual quaternions. Each blended
point transform is rigid, which avoids the shrinkage from averaging
rotated points at a twisting joint. Nearby vertices can have different
weights, so the surface can still stretch, bulge or intersect itself.
This blend gives no guarantee of whole-body volume or joint contact.
Species-specific weight placement, soft-tissue movement, collision and
anatomical validation remain the maker's responsibility.

## Drawing a set

draw a checked set from the checkout root after `game build slayer`. The set supplies the
studio, cameras and light; `--camera` selects a named view. `--at` and
`--renderer` are also available, and `--dry-run` reports the planned job
without drawing or staging the scene.

```sh
game run slayer -- draw sets/bioforms/bay-seal.yaml --camera side \
  --width 480 --height 300 --rays 1 --out out/bioforms/bay-seal.png
```

Use `sets/bioforms/orca-clean-05.yaml` or `sets/bioforms/walrus-fresh-5.yaml`
for the orca and the walrus. use a [job](../../../job/README.md) to save the camera, scene time and
output settings together.

## Deforming sets

[Authored pose clips](pose-clips.md) let a set specify the native human or
orca's named joint turns, whole-body motion, and bone-carried props.

The seal and the walrus are static specimens. The orca has an authored
three-second deformation loop when sampled over scene time. It is a study, not
a measured swimming gait, and its placement and camera anchors stay fixed: the
loop does not move it through the water.

Deforming sets need CPU rendering and refuse shutters, lens rigs, cached
lighting, close-glow masks, and stand or ecosystem exclusions. Their initial
bounds do not yet give a conservative envelope for every pose. The
scene-clock ocean path does not support regular terrain and things together.
These limits have to be lifted before a coastal swim is claimed.
