# Author a bioform pose clip

A `pose-clip` recipe specifies named joint turns and optional rigid
attachments. The [adult studio reach][human] uses the native human and
carries a small marker on the left hand. The [orca example][orca] uses
the checked orca profile, bends the tail stock and moves the whole
figure. Both are editable in the set YAML.

```yaml
recipes:
  figure:
    kind: adult-human-neutral
    pose-clip:
      timing: smooth
      keys:
        - at: {value: 0, unit: s}
        - at: {value: 1.4, unit: s}
          turns:
            thorax: [0, 8, 0]
            left-upper-arm: [0, 0, 23]
            left-forearm: [-14, 0, 0]
      attachments:
        marker:
          bone: left-hand
          kind: can
          offset: [0.018, -0.028, 0.030]
          scale: 0.35
```

Place `figure` under `scene.recipes`, and name it with `recipe: figure`
on a thing. A human recipe can use one of the native adult reference
kinds to select construction, pigment and hair. The clip supplies the
whole pose, including for a kind that ordinarily has a built-in stance.
For an orca recipe, replace `kind` with
`animal: {family: whale, file: "@builtin", species: orca}`.
A checked external whale book can replace the built-in book.

- Keys use elapsed scene seconds, start at zero, and must increase.
  A single key supplies a still. Before or after the listed times the
  clip holds its endpoint. Each key is a complete pose. Copy a turn to
  the next key to keep it there. An omitted turn is neutral at that key.
- `turns` maps bone names to `[x, y, z]` angles in degrees. The turns
  operate about the bone's head in the skeleton's rest axes, in X, Y,
  then Z order. A child inherits its parent's motion.
- `root-offset` and `root-turns` move the whole figure within its set
  placement. Offset is a metre triple in figure axes. Root rotation is
  a degree triple about the figure origin, applied before the offset.
- `timing: linear` advances at a constant fraction between keys.
  `timing: smooth` eases away from and into each key. Bone and root
  rotations interpolate along the shortest arc. Add a key to control
  the direction of a turn that would otherwise take the other arc.
- An attachment names a static manufacture kind and a bone. `offset`
  is in rest-axis metres from the bone's head. Optional `turns` gives
  the prop's starting rotation in degrees, with the same axis order.
  Optional `scale` is uniform and defaults to one. The prop follows
  the sampled bone as a rigid body. Light-carrying kinds are refused.

The rig checks the names against the chosen specimen when it prepares
the clip. The human's skin and tissue, eyes, and scalp follicles all
follow the same sample. Paint stays in the original surface chart.
Skin vertices blend several bones near joints. A rigid prop stays with
its bone even if the nearby skin follows several bones.

Draw with the CPU renderer. For a still, choose the time with `--at`
using a duration such as `700ms`. A film job samples the same scene at
its frame times. Live deformation supports one geometry sample per frame.
Sweep clearances, contact, anatomical joint limits, and inverse kinematics
require separate authoring or solvers. The renderer uses initial bounds
for arrangement and does not derive a swept-motion envelope.

[human]: ../../../sets/bioforms/adult-human-reach-clip.yaml
[orca]: ../../../sets/bioforms/orca-authored-pose.yaml
