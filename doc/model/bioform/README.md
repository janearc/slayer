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

## How a bioform is made

The work is done in passes. A pass states its sources and what they give and
do not give, builds the animal from an empty file, renders matched views and
a short motion proof, and compares them with the sources. Whatever the
comparison finds wrong is written down before the next pass begins, and the
next pass builds the animal again rather than patching the last. Each page
here records what the passes taught, under "Earlier builds".

These rules came out of that work and hold for every animal:

1. Separate what is measured from what is imagined, and choose an explicit
   individual scale and coordinate system. Fossil bone, a related species,
   soft tissue, texture and locomotion are different kinds of evidence. No
   imported animal asset is needed to study a source.
2. Start from recognizable anatomical regions seen from several sides. A
   smooth surface, a connected shell or a matching side outline does not make
   a plausible animal.
3. Keep a true undeformed reference and a stable indexed skin. If a field is
   cut again at each frame, correspondence and painted marks are unproven.
   A deforming mesh does not imply a vertebrate skeleton or a tissue solver.
4. Test the failure modes one by one: connected components, closure, unused
   vertices, Euler characteristic, the largest edge change against rest,
   volume, how far landmarks move, attachment, and paint seen through real ray
   hits. Total volume, a nominal joint control or one good ray cannot certify
   the whole animal.
5. Paint is not geometry. A painted fold is not a tissue fold, and teeth and
   similar details need roots and an orientation frame. Judge pose, motion,
   appearance and evidence separately. Claim no gait, swim, contact or fluid
   force that has no test.
6. A "matches the source" claim names the landmark compared. A skin box is
   not a skull length.

## Drawing a set

Draw a checked set directly from the repository root. The set supplies the
studio, cameras and light; `--camera` selects a named view. `--at` and
`--renderer` are also available, and `--dry-run` reports the planned job
without drawing or staging the scene.

```sh
bin/slayer draw sets/bioforms/bay-seal.yaml --camera side \
  --width 480 --height 300 --rays 1 --out out/bioforms/bay-seal.png
```

Use `sets/bioforms/orca-clean-05.yaml` or `sets/bioforms/walrus-fresh-5.yaml`
for the orca and the walrus. The older `slayer render SET` entry point also
draws a set, using its first camera. It needs `SLAYER_CACHE` set to an
absolute writable folder and takes `-out` for the output path. A direct
preview carries no job source receipt.

## Deforming sets

The seal and the walrus are static specimens. The orca has an authored
three-second deformation loop when sampled over scene time. It is a study, not
a measured swimming gait, and its placement and camera anchors stay fixed: the
loop does not move it through the water.

Deforming sets need CPU rendering and refuse shutters, lens rigs, cached
lighting, close-glow masks, and stand or ecosystem exclusions. Their initial
bounds do not yet give a conservative envelope for every pose. The
scene-clock ocean path does not support regular terrain and things together.
These limits have to be lifted before a coastal swim is claimed.

## Scene defaults

Each named study scene has its own default lens. The tests that pin scene
defaults check every scene individually, require equality inside an explicit
family of studies, and reject duplicate defaults between families or between
unrelated scenes. That is why the walrus studies keep 38 degrees, the
Purussaurus studies use 39 and the pliosaur studies 40, while the review jobs
of each set the lens they were framed with.
