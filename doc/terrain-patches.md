# Source patches selected by a view

The source and patch cache now have focused host test evidence, recorded
in [the verification notes](verification-2026-11-09.md). Mesh, view-adapter
and renderer acceptance remain separate. This does not yet replace the
existing Hillside terrain stage or establish whole-world rendering.

## Identity and bounded reads

RootPatch covers a raster's complete native sample lattice. A Patch is
its source-local integer rectangle and subdivision level; source identity
belongs alongside that key. Changing a render crop does not change it.

Children splits cells while sharing endpoint samples exactly. A narrow
strip can split along its remaining nonterminal axis. A source containing
only one cell in each direction cannot subdivide further.

ReadPatch chooses native indices, keeping both endpoints, at a bounded
sample count per side. It reads through the existing byte-bounded tile
cache and retains height availability. Its output byte limit accounts
for both native-index axes, heights and the availability mask.

Index selection uses wide integer arithmetic, without overflowing an
index-times-source-span product. Sampling centres are those of the native
source cells; float coordinate binning is not used to recover their IDs.

## Reuse without a camera key

PatchCache now drafts a second bounded layer above the decoded tile cache.
It retains immutable sampled grids by native patch and actual axis counts.
Two oversized side requests that resolve to the same native grid share
one entry. Camera positions, timestamps and render crops are not keys.

The caller supplies byte and entry limits explicitly. A cache hit must still
fit that call's output byte allowance. Misses evict least recently used grids
before allocating a retained replacement; an oversized result may be read
within its separate output allowance without being retained.

Failed reads retain no partial grid. Usage distinguishes hits, attempted
misses, retained entries and payload bytes.

The cache borrows one open, immutable raster; it does not own the file.
Closing that raster refuses subsequent reads, including cached hits. Clear
releases the cache's own roots. Read-only grids already held by a consumer
remain valid and consume memory until that consumer releases them.

Closing the source does not automatically clear a separately held cache; the
world manager must clear it or release it too.

Sample payloads, decoded tiles, meshes, GPU inputs and consumer-held grids
are separate allocations. This cache does not establish an aggregate RSS
cap. It is not yet wired into moving-view staging, nor does it cache a
transformed mesh under an unspecified projection or height datum.

The cache witnesses now exercise reuse and eviction, a stricter allowance
on a hit, zero retention, closed sources and consumer-held grids after
Clear. Failed patch decoding after eviction and mesh/view integration
still need their own cases.

`terrain/testdata/patch-cache-contract.yaml` records independent payload
arithmetic and reuse order for the existing small TIFF fixture.

## Selection and budgets

SelectPatches starts at a source root and refines the highest-scoring
visible patch. The caller supplies conservative visibility and a finite
score, such as projected sample spacing for the actual current lens.

Patch count, view evaluations and subdivision depth have separate limits.
When children cannot fit, their coarse parent remains. Limited explicitly
reports that the requested target was not attained; visible coverage is
not silently discarded to meet a count.

The selector chooses no altitude threshold and no arbitrary kilometre
crop. A narrow instrument field can request finer distant data through
its own projected score. Equal priorities and final patch ordering are
deterministic.

A skipped-sample grid does not bound unobserved peaks or no-data holes
between its samples. It is not a filtered overview or a geometric-error
estimate. Native overview/error metadata and an actual view adapter are
still required before claiming such guarantees.

The [projected-detail draft](projected-detail.md) now supplies a separate
pinhole/frustum assessment for a caller's world bounds and displacement.
It is not yet a source projection/datum adapter or an enabled terrain stage.

SelectProjectedPatches now connects that assessment to the bounded source
selector. It takes the actual camera pose, field of view and raster through
footprint.Perspective. Its target is projected displacement in pixels. A
distant view can select fewer patches over the same source footprint; a
narrow lens can ask for finer distant patches again.

The source adapter supplies PatchGeometry: full world bounds and a bound
on the displacement introduced by the chosen representation. Unsampled
peaks must remain inside those bounds. This helper does not estimate error
from a handful of DEM samples. A retained leaf touching the near-plane
uncertainty reports Limited, as does a detail request that exceeds budgets.

The technique is view-dependent quadtree level of detail (LOD). Geometry
clipmaps are another terrain LOD organization: nested grids centered on the
viewer, with progressively coarser levels farther away. The primary
[GPU Gems chapter by Asirvatham and Hoppe](https://developer.nvidia.com/gpugems/gpugems2/part-i-geometric-complexity/chapter-2-terrain-rendering-using-gpu-based-geometry)
describes that approach. This selector uses a quadtree rather than clipmap
rings. Neither organization manufactures terrain beyond source coverage.

Moving renderer residency, source overview/error metadata, seams between
unequal patch resolutions and explicit fallback coverage are still open.
The current Hillside scene continues to stage its authored terrain crop.

## World mesh boundary

PatchGrid.Mesh takes a count budget and a projected-source-to-world
mapping callback. The mapper must own projection and height-datum
conversion into the fixed frame. This slice does not invent that mapping.

Missing sampled corners suppress triangles that reference them. An empty
mesh is reported as absence, while corrupt reads, invalid mapped points
and construction failures remain errors. A world manager can select
another source or an explicit reference-surface fallback.

Source patch meshes are ordinary mesh geometry for the existing GPU path.
The current helper uses flat faces and the legacy terrain diagonal. It
does not stitch unequal-detail neighbours, filter normals across edges,
manage visibility residency or apply the renderer's future rebasing.

Those seams, the projection/datum adapter, planetary fallback and full
staging integration remain open. This reader/selector must not be
presented as a completed whole-world renderer.

## Shared source edges

PatchGrid.StitchTo now supplies bounded source-space edge matching for an
affine world mapping. A fine patch follows its coarser neighbour's linear
edge segments. Coarse knots inside the overlap must also be fine knots;
otherwise the helper refuses the pair rather than leaving a hidden bend.

The helper returns independent height/availability payloads, retaining the
cached source grid. Missing heights in either grid remain absent. Its byte
allowance covers the copied payload; axis arrays are immutable and shared.
The maximum changed sample height is returned so callers can account for
the extra representation displacement.

Focused CPU witnesses cover all four shared sides, exact edge interpolation,
cache immutability, no-data propagation, byte refusal and a missing coarse
knot. This does not yet stitch curved-world mesh edges after projection or
establish an entire resident patch set. Callers must supply compatible
source identity, placement and resolution ordering.
