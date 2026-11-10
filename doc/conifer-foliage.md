# Branches with needles

The first observatory renders used closed green sleeves. Increasing image
resolution exposed their shape more clearly. Coast redwood and giant sequoia now
attach recursively branched shoots carrying individual leaves. The existing
seeded trunk and major branch scaffold is retained.

## What the sources support

[Prusinkiewicz and Lindenmayer, The Algorithmic Beauty of Plants, 1990, chapter
2](https://algorithmicbotany.org/papers/abop/abop-ch2.pdf), figure 2.6 and
printed pages 56-57, describes continuing axes that issue lateral axes, length
contraction, alternating lateral directions, and local branch frames.

The implementation executes a parametric recursive rule directly. It does not
need to construct and interpret an L-system string. The source key is
`abop-tree-models-ch2` in sources.yaml.

[Weber and Penn, Creation and Rendering of Realistic Trees, SIGGRAPH
1995](https://doi.org/10.1145/218380.218427), sections 4.3-4.5, separates
recursive stem orders, child dimensions and foliage. Section 4.4 gives:

```text
child radius = parent radius * (child length / parent length)^RatioPower
```

The new shoot generator uses that radius relationship. Its exponent is our
recorded choice, not a measurement taken from the paper. This is an adaptation
of selected rules, not a complete Weber-Penn implementation. The source key is
`weber-penn-1995`.

Neither reference supplies a calibrated redwood or sequoia recipe. The major
crown proportions, lateral counts, branching angles and spacing remain authored
choices. No physiological growth, bud competition or water balance is simulated
here.

LIGNUM was considered as background reading. It is not an implementation
source for this generator.

## Species foliage

[Oregon State's coast redwood
description](https://landscapeplants.oregonstate.edu/plants/sequoia-sempervirens)
describes flat, two-ranked needles 1-2 cm long. The model uses that flat spray
form, with individually generated lengths inside the described range.

Redwood foliage is dimorphic; this first implementation does not distinguish the
scale-like leader foliage from the flat needle form. The pinned source key is
`osu-coast-redwood-foliage`.

[Oregon State's giant sequoia
description](https://landscapeplants.oregonstate.edu/plants/sequoiadendron-giganteum)
describes awl-shaped, overlapping leaves in three longitudinal rows and gives
3-6 mm lengths on leaders. The model uses three rows and that range.

Applying those leader dimensions to every modeled shoot is our approximation; it
is not a documented measurement of every foliage type. The pinned source key is
`osu-giant-sequoia-foliage`.

The four length endpoints live in research/research-derived-data.yml. Their
source excerpts locate the ranges, so a small numeric value cannot be verified
against an unrelated number elsewhere on the page. Source hashes and exact
excerpts are checked before Game builds the commands.

The PDFs and pages stay on the local source shelf, outside the repository.

Widths, thicknesses, leaf spacing, inclinations, radius exponent, branchlet
counts and reach, spray roll and recursion limits are in
manufacture/needled-shoot.yml, read through Puzzlemouth's numbers reader.

Each is marked as picked, with who chose it and when: rendering choices, not
field data.

## Structure and detail

A terminal shoot's continuing axis produces contracting lateral axes. Redwood
laterals alternate in a plane; sequoia laterals turn around the parent. Needles
attach to the woody axis at their bases. Growing or binning a shoot does not
scale up its needles.

The four complexity names remain:

- hull encloses the crown coarsely.
- clumps uses an open diamond for each needle or scale.
- quick closes each leaf as a thin tetrahedron.
- demo uses a thin, closed diamond with eight faces.

All three foliage levels use the same needle positions and physical lengths.
They change surface detail, not the scaffold. The two original spruce recipes
retain their legacy preview geometry for compatibility; this replacement
currently applies to coast redwood and giant sequoia.

The named `coast-redwood-shoot` and `giant-sequoia-shoot` specimens expose the
same shoot prototypes for close views. They are not separately drawn
illustrations or special geometry used only for demonstration.

## What reaches the GPU

Small, fixed-size shoot meshes are reused along the scaffold. Their sizes are
chosen from bins; their needles retain physical dimensions. Two seeded
microforms provide local variation. This prototype library is an explicit
rendering approximation.

The branchlet density choices currently serve both species. Neighboring
sprays can roll about their axes, so an entire branch does not collapse
into one flat sheet. The largest shoot cell is 40 cm; leaf lengths stay
fixed.

A material's MeshGroup stores placements and a local bounding hierarchy. The GPU
keeps that hierarchy once per specimen and instances the specimen through the
forest. Traversal is world instances, local shoot instances, then triangles.

Groups accept flat meshes and cannot contain groups; there is no recursive Metal
function or unbounded nesting.

Stored face budgets count unique prototypes. Scene face counts include their
repetitions. CPU rays use the same placed meshes through a local hierarchy.
Group tests compare against explicitly transformed triangles, and GPU tests
compare both images and storage counts.

CPU shading retains the member hit by each ray. The GPU advances rays near each
shoot before applying its local rotation, using fused operations for instance
transforms. The pinhole camera centers pixel coordinates before scaling, then
normalizes once after rotation.

These conditioning changes preserve the ray and avoid losing needle widths to
distant-origin arithmetic.

Construction tests also hold leaf lengths to the published ranges, check needle
roots against twig surfaces, check recursive attachment and radius contraction,
and shoot rays through the gaps in a spray.

Whole-crown silhouette checks cover coast redwood at `quick`,
seeds 5 and 11, from three sides. Their floors are visual regression choices,
not field measurements. The foliage-only GPU check covers both species at
`quick`, seed 5, without bark or shadows.

The lit CPU/GPU fixture uses a separate authored regression budget: summed
absolute RGB difference divided by CPU RGB energy must stay below `2e-3`,
with at most four pixels disagreeing in surface coverage. These are fixture
limits, not a general error bound for thin foliage.

The restored-geometry run associated with `db2823d` measured a worst relative
RGB difference of `0.00132864`, for redwood at `demo`; the other tree levels,
observatory and both loose shoots were below it. The same bound is retained.
The camera exception for horizontal loose shoots does not change tree views.

Images stay outside the repository so the geometry can be inspected directly.

## Draft spruce extension, 8 November 2026

The named spruce and Norway-spruce factory entries now have a draft
wrapper with radial needle shoots. The original conifer rows remain
legacy references. [Spruce construction and limits](spruce-foliage.md)
records its picked dimensions and qualitative botanical source.

The earlier captures and test results in this document concern redwood
and sequoia. They do not validate the new spruce construction. Builds,
appearance captures and CPU/Metal checks are deferred to the review break.
