# ReSTIR direct light for many lamps

ReSTIR is an optional direct-light approximation for scenes with
many positioned lamps. select `shortcuts: {restir: on}`. it runs on
native Metal with a pinhole and a closed shutter. it requires at least
one positioned lamp and refuses floods, suns, beams, polar lamps,
gravity and participating media. disable it for a raw-light reference.

the regular path already uses a light tree for large groups of
positioned lamps. ReSTIR adds reservoir resampling at primary hits:
it proposes lamps from a light tree, reuses candidates from nearby
samples and the previous frame, and casts a final shadow ray for
the chosen lamp. the estimator follows ReSTIR DI (Bitterli et al.,
SIGGRAPH 2020).

## Where it plugs in

Geometry is untouched. Slayer already builds Metal acceleration structures
(`metal/acceleration.go`: primitive structures, instance structures, one
world instance structure) and queries them from `trace.metal`
(`native_shadow`, `native_first`). This code builds no second structure for
geometry. Its shadow rays are the kernel's own `blocked` and `unhidden`.

What is new:

1. A light tree over the lamps (CPU build, `gpu/restir_tree.go`).
2. Reservoirs and four kernels: `restir_gbuffer`, `restir_resample`,
   `restir_spatial`, and `trace` reading the chosen lamp.
3. A traversal mode, `restirTraversal`: native primary and shadow queries,
   as `nativeTraversal`, plus the passes above.

The kernels live in `gpu/restir.metal`. `gpu/trace.metal` is not edited:
in the restir mode only, `gpu/restir_source.go` puts four small changes
and the new passes into the kernel text, each at a place that must occur
exactly once or the draw is refused. Every other mode compiles the text it
compiled before; a test holds the legacy text to its hash.

## The light tree

A binary tree over the lamps that light by position: a lamp that is not a
flood, a sun, a beam or a polar lamp. A world with any other kind of lamp
is refused by name; those lights have no position to put in a tree.
Built top down on the CPU by splitting at the median of the longest axis of
the lamps' centres. One lamp a leaf. A node holds its box, the sum of its
lamps' luminance (its power), and its two children.

Choosing a lamp for a surface point x with normal n: start at the root; at
each node weigh each child by `power / max(d2, floor)`, where d2 is the
squared distance from x to the child's box; a child whose whole box lies
behind the surface's plane (every corner has `dot(n, corner - x) <= 0`) has
weight nought. Take a child with probability its weight over the sum; the
product of the probabilities is the pdf of the lamp.

A point with both
children at nought has no candidate. The weight is a proposal, not an
estimate: it only has to be above nought wherever the lamp could light the
point, so the weights stay unbiased whatever they guess.

Depth is about log2 of the lamps, 17 for 100,000.

## The reservoir

Four numbers a sample, 16 bytes: the chosen lamp's index (as a float, whole
numbers are exact to 2^24), `W` the unbiased contribution weight, `M` the
count of candidates it stands for, and a spare. The target function
`phat` is the lamp's light at the point times the cosine, as its
luminance, without the shadow: `luminance(lamp_light * max(0, n . toward))`.

A candidate has resampling weight `phat / pdf`. A reservoir keeps one
candidate by weighted reservoir sampling and ends with
`W = wsum / (M * phat(chosen))`.

A G-buffer of eight numbers a sample goes beside it: the shading point
(already lifted off the surface), its normal, the index of the sphere it
leaves, and its depth. Reuse reads it to decide whether two samples are
alike.

## Per frame

1. `restir_gbuffer` records each primary hit.
2. `restir_resample` takes four candidates from the tree,
   then, if `Visibility` is set (it is off by default), one shadow ray on
   the chosen one, an occluded choice having `W = 0`.

   Then temporal reuse: the point is projected into the last frame's camera;
   if the pixel found there has a depth within ten percent and a normal
   within about 25 degrees, its reservoir joins, its `M` capped at twenty
   times the current one's.
3. `restir_spatial`: three neighbours drawn from within sixteen sample positions, kept
   if alike by the same test, joined by their `phat` at this point.
4. `trace`: the megakernel, as ever, except that at the primary hit
   `falling` takes the reservoir's lamp in place of `asked`, casts one real
   shadow ray, and adds `light * visible * W / pi`. Sky, mirrors and every
   later bounce are lit as before, by the exact rules. Where the world has
   no sky the sixteen sky looks are skipped under restir; they sum to
   nought there.

Merging a neighbour's reservoir: its candidate is weighed
`phat_here(candidate) * W * M`, M adds, and `W` is recomputed.

## What is approximate

- Reuse does not test visibility for the borrowed candidate. A neighbour's
  lamp that is hidden here is kept and then shot at in step 4: the pixel
  gets nought from it instead of a better candidate. This darkens near
  shadow edges. It is the biased variant of the paper, named as such.
- temporal reuse can lag behind moving lights. the reservoir count
  cap limits the weight of reused samples.
- The estimate is noisy at one candidate set a pixel; neighbours' sharing
  is what smooths it.
- Lamps with a cone are proposed as if they lit every way. They are
  weighed correctly at the end; only the proposal is blunt.

## How a job turns it on

choose `shortcuts: {restir: on}` in the job or `--restir on` on the
command line. `fastlight: on` and `vibes: on` request it too.
ReSTIR takes priority over the lamp store and shared sky looks.
set `shortcuts: {restir: off}` to use those instead. see
[lighting flags](flags.md) for precedence.

the job records which shortcuts it requests. the GPU profile reports
when ReSTIR was applied, and a retained film drawer can reuse the
preceding frame’s reservoirs.

## Not done here

- The passes run through the host between dispatches: each pass reads
  reservoirs through host storage. a device-resident stream is a
  possible improvement.
- No visibility on spatial reuse, no unbiased (MIS) combination.
- Lamps that move between frames rebuild the tree each frame.
