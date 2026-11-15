# The island needs ground beyond the first window

The island study connects a geographic raster to the spherical frame and
prepares ordinary GPU terrain meshes for the actual camera. Its small Game
command is a staging path for the full ground-to-orbit scene, not completion
of that scene. The first input is [a topology preview](island-source-preview.yaml).

Build and run from the checkout root:

```sh
game build island-binary
game run island -- render --scene doc/island-source-preview.yaml --out out/island-preview
```

The destination must be new and its parent must exist. The example needs the
exact NOAA subset described below. Change its file path for your own data
directory; relative paths resolve beside the YAML, not the current directory.
The command does not fetch or silently substitute a source. A dry run checks
the input schema and reports the request without opening a GPU or output.

## Sources and the chosen world

Every source has an expected SHA-256, geographic CRS, named vertical datum
and explicit height offset. The command verifies the file before decoding it.
Sources are ordered: an absent height may fall through to the next source,
but corruption or a decoding refusal stays an error. If none has a height,
preparation fails. An inland hole cannot become a flat sea by accident.

`height-model: authored-radial-offsets` interprets each source's geographic
angles on the declared sphere and adds its stated height offset. This is an
explicit scenery approximation. It does not transform NAD83 into WGS84,
NAVD88 into mean sea level, or orthometric height into ellipsoid height.
The initial preview uses the MSL coastal model alone; joining the USGS land
detail still needs an accepted common-height policy.

The NOAA Hawaii CRM2023 WCS subset was retained through Celestial-data's
`noaacrm` adapter at `3f26308ac7047e0b8f84c3ef16a7de3d7db8c70e`. The
archive-published endpoint and coverage description supplied the request:

- [NOAA WCS capabilities](https://www.ngdc.noaa.gov/thredds/wcs/crm/cudem/crm_vol10_2023.nc?service=WCS&version=1.0.0&request=GetCapabilities)
- [Coverage z description](https://www.ngdc.noaa.gov/thredds/wcs/crm/cudem/crm_vol10_2023.nc?service=WCS&version=1.0.0&request=DescribeCoverage&coverage=z)
- [NOAA coastal relief model documentation](https://www.ncei.noaa.gov/products/coastal-relief-model)

The request box was west -156.2, south 18.8, east -154.7, north 20.4,
in CRS84, with GeoTIFF_Float output and no resampling arguments. The retained
file is 124,507,189 bytes, SHA-256
`37c3cf3b48849d599950f255e64fed2bed7eb744ef18bf73d2375e9f0fc0f830`.

It has 5401 by 5761 big-endian float32 samples, uncompressed one-row strips,
WGS84 geographic coordinates and PixelIsArea registration. Its two angular
scales differ slightly in the delivered bytes; the reader retains both.
The first sample centre is near longitude -156.19986124556166 and latitude
20.399861183961665. The example region stays inside the interpolation domain.

All 31,115,161 delivered samples were independently scanned. There were no
NaNs, infinities or missing values; the measured stored range was
-5648.2431640625 to 4197.93603515625. The example's round height envelope
encloses that full range, rather than estimating peaks from sparse queries.

The delivered TIFF's no-data text is -5649.243, not the NetCDF's -9999;
comparison uses its float32 representation. No vertical CRS or unit GeoKeys
are present. Metres, positive up and MSL EPSG:5714 come from NOAA's retained
product guide and NCML. The retained inspection also records nonfatal TIFF
metadata warnings; a complete independent libtiff decode exited successfully.

The contemporary FETCHED.md, exact request, source documents, scan manifest
and warnings are kept beside the local raster under the research adapter's
`retained/hawaii-crm-2023-big-island` directory. Keep them with a copied source.
Grid spacing is not a promise of equally fine measured coastal detail.

## Geometry follows the view

`scene/island.Terrain` keeps one frame and source region across views.
The camera, field of view and output dimensions drive bounded quadtree
selection. Each selected patch has a regular source-coordinate lattice;
one sampled location has the same identity wherever it is shared.

`cell-pixels` is a projected cell-span target. It is not a bound on omitted
terrain relief. Complete source-height bounds enclose unsampled peaks for
visibility; the representation can still alias fine relief. Budgets retain
coarse coverage and record `cell-target-limited` when the target cannot fit.

Unequal-resolution shared edges follow the coarser patch's actual segments
after spherical projection. This avoids treating linear source heights as
linear world-space edges. No missing source height is filled by stitching.
The source samples stay unchanged; the view's representation changes.

Terrain vertices, the camera and ocean centre all subtract the same render
origin before GPU encoding. The directional lamp is unchanged by translation.
This command does not yet rebase existing Hillside weather or spacecraft.

Source cache payloads share one declared total allowance; tile and metadata
limits are separate. Selected patch counts, selection work, lattice depth
and sampled point records have their own explicit limits. They do not form
a whole-process RSS guarantee, and source access remains serial.

## What the first picture means

The original topology input uses a single authored ground material, a uniform
background and a smooth spherical ocean proxy through the existing reflection
path. The water's index and colours are picked preview inputs, not a sourced
spectral ocean model. This view contains no wave simulation or ecosystem yet.

## A site on the sourced ground

The [observatory input](island-observatory-preview.yaml) adds an authored
site to the same world. Its fixed east/south chart maps locations along the
sphere, samples source heights and places equipment and trees with radial up.
The introduced redwood/sequoia mixture is scenery, not a Hawaiian flora survey.
Equipment clearings keep roots away from the manufactured buildings.

Root identity and prototype variety are independent of the camera. Projected
specimen size chooses aggregate appearance, hull, clumps, quick or demo detail.
An authored canopy colour blends into the ground at large ray footprints;
this is not a physical canopy or a proved seamless transition. Cell, candidate,
prototype-face and material-group instance budgets refuse excess work.

Optional height indices scan every native source sample into bounded min/max
blocks. These tighten terrain refinement priority without claiming a relief
error bound. Whole-region bounds still decide terrain visibility: a stitched
leaf edge can follow a coarser neighbour outside that leaf's native envelope.
Off-screen coarse coverage is retained rather than culled on an unsafe bound.

Prescribed mist uses existing weather media in a local tangent frame, with
immutable sampled terrain floors. Floor projection has an explicit tolerance
and iteration limit; nonconvergence is an error. These finite regional decks
are authored weather, not tree transpiration or a planetary atmosphere.
The four example beams share one boresight and visible launcher geometry;
finite beacon heights, asterism offsets and refraction remain separate work.

`secondary-ray-reference-metres` optionally selects the base length in the
existing ray-origin offset heuristic. Zero-value worlds retain their previous
geometry-derived choice. The point-magnitude contribution and multiplier stay
unchanged on CPU and GPU. Camera-relative planetary scenes can avoid a globe's
diameter setting metre-scale offsets at small trees. This numerical control
does not prove a floating-point intersection-error bound.

The first native site studies now include a close 720p view and a 15 km
overview. The close view shows the dome, source-grounded trees, four beams and
regional mist. The overview exposed a rectangular mist edge. Optional
`edge-fade-metres` now gives a bank elliptical horizontal support and a smooth
outer transition; zero retains rectangular support. This is an authored shape,
not a condensation model. Moving-view transitions still need inspection.

Optional `patch-scale-metres` separates the size of seeded weather patches
from a layer's vertical depth. Omission keeps the original depth-based noise.
Wind advects the same field at either scale; coverage remains an authored
threshold, not a measured cloud fraction.

Weather footprints can extend up to 200 km on each local coordinate axis,
independently of the smaller forest limits. The terrain source must cover the
entire projected floor; unsupported regions fail instead of becoming sea.
Fixed floor-grid and density-grid allowances still bound preparation.

These remain decks in the site's tangent frame. A broad regional deck can
place weather away from the observatory, but is not a curved planetary
atmosphere, a forecast or a condensation simulation.

For weather that follows the sphere and continues offshore, use the separate
[planetary cloud layers](island-clouds.md). Those fields use direct CPU/Metal
evaluation and split occupied shell crossings; they do not enlarge a local
floor grid into a planetary weather approximation.

The original plain island stills below do not verify the added site.

The site stills are retained locally under:

- `2026-11-09-sourced-observatory-320`: the first, distant placement study.
- `2026-11-09-sourced-observatory-close-320-v2`: a closer north-side view.
- `2026-11-09-sourced-observatory-close-720`: 1280 by 720, four samples per
  pixel, 1,256 drawn trees and four visible beams; 12.68 s application time.
- `2026-11-09-sourced-observatory-rise-320`: the original 15 km overview,
  retained with its visible rectangular mist-bank termination.
- `2026-11-09-sourced-observatory-rise-faded-320`: matched geometry and
  camera with the edge fade; the bank's termination is now rounded and soft.
- `2026-11-09-sourced-observatory-orbit-320`: 300 km camera height, whole
  island coverage and aggregate forest appearance, with no root enumeration.

These are actual inspected native-Metal stills with exact settings, PNG and
producer hashes, source records and resource receipts. The initial close
attempt refused its face budget before rendering; the successful input uses
one prototype variety within an explicit two-million stored-face allowance.
The 720p run sampled about 529 MiB process-tree RSS under the watchdog; this
is not a guaranteed peak or a whole-system/GPU-memory measurement.

All terrain selections still report a limited cell-span target. Faceting,
thin-foliage aliasing at four samples, the uniform background and local weather
support remain visible approximation limits. The zero-fade 720p study predates
the boundary correction. None of these stills demonstrates temporal continuity,
wave water or the CORONA/Keyhole instrument view.

The output keeps `island.png`, the exact `scene.yaml` and a `render.json`
with source identities, executable hash and build info, image hash, origin,
selected patches, target status and measured GPU costs. A failed invocation
may leave partial artifacts; only a successful run and inspected image are
a render result. A dry run or a successful compile is not a rendered proof.

## A continuous geographic ascent

Optional `detail.normal-step` supplies longitude/latitude stencil half-spans in
source degrees. Four neighbour heights estimate each shading normal in the fixed
sphere frame; geometry and normal construction share coarse-edge knots. The
example uses the delivered NOAA sample intervals, including their slight axis
difference. A complete source halo is required and absence remains an error.

Normal estimates have a separate cache bounded by the point allowance; their
stencil and height-query counts are recorded separately from geometry. This
shades existing triangles smoothly; it does not add terrain detail or certify
relief, shadow-ray error or temporal normal continuity. Omission keeps flat
shading. Stencil steps are explicit sampling choices, not a resolution or
accuracy claim about the archive.

`landcover-transition-metres` is an optional authored half-width for smooth
blending between adjacent height colours. Overlapping transition ranges are
refused. Zero preserves the old steps; the last cutoff still returns the base
colour. This is an appearance control, not a measured climate/vegetation map.
The revised example uses that transition for authored high-elevation snow
appearance. It is not fetched weather, snow depth, accumulation or tree loading.

The first inspected normal/snow study,
`2026-11-09-sourced-island-normal-snow-320`, used 261,313 bounded normal
stencils and 1,045,252 source-height queries. It completed under the two-worker
guard; this is not a controlled timing comparison or a physical snow validation.

`forest.adaptive-regions` enables hierarchical support selection before cell
enumeration. It first uses the declared full specimen envelope; a wholly
subpixel region needs no individual roots. Nearer views measure all permitted
variant/detail bounds before using a tighter, rotation-safe envelope. Native
source extrema tighten regional height bounds.

Region assessments and actual cell queries share the explicit cell allowance;
candidates and instances keep their own limits. This does not make off-screen
shadow influence complete. The pixel thresholds apply to the conservative
projection criterion, which can exceed a tree's literal image height. The first
snow/woodland movie aggregated that criterion below 40 pixels. The current
example lowers it to 8; both are authored quality/work choices.

`minimum-geometry-detail: clumps` replaces the hull tier with branch clumps,
while preserving the aggregate cutoff. This avoids coarse conical silhouettes;
omission retains the hull tier. The current example uses fitted crowns below
the clump tier, as described later in this note.

Only supports large enough to need individual trees enumerate candidates. The
authored canopy appearance follows the same fixed spherical chart and blends in
as projected trees become small. Optional forest `edge-fade-metres` thins
candidate density and aggregate colour toward the region boundary.
`height-fade-metres` similarly thins the elevation limits, letting high summit
snow show through instead of painting those peaks with aggregate forest colour.

The wider example spans 40 km by 40 km at 20 candidates per hectare before
elevation/clearance/edge filtering: an expectation of 3.2 million candidates,
not a request to construct them all. Its introduced species and weather remain
authored scenery. The first wider-forest still renders 1,639 trees after 437
region assessments, 1,940 cell visits and 63,566 generated candidates.

At 300 km the whole support aggregates, with zero cell visits and zero generated
candidates. The first full sixty-frame snow/woodland ascent completed under the
guard in 94.97 s, with 726 MiB sampled process-tree RSS. Its contact sheet
exposed coarse hull cones. A following clump-only study removed that tier but
kept the abrupt aggregate transition. The later fitted-crown study is described
below; changed settings do not establish an isolated speedup over this ascent.

The [ascent plan](island-ascent-preview.yaml) supplies geographic camera and
look-target keyframes, horizontal field of view, outgoing-segment easing and
scene timestamps. It points to a scene beside the plan. Run it through Game:

```sh
game build island-binary
game run island -- sequence --plan doc/island-ascent-preview.yaml --out out/island-ascent --dry-run
game run island -- sequence --plan doc/island-ascent-preview.yaml --out out/island-ascent
```

The destination must be new. Fix the scene's local raster path first. The
example asks for 60 frames at six playback frames per second, spanning an hour
of scene time: ten seconds of playback at 360 times the scene clock. Both
camera endpoints and clock endpoints are sampled exactly. Wind and attached
lights receive scene time; the authored Sun direction remains fixed.

Longitude, latitude and radial height interpolate independently, with no
antimeridian crossing. Height interpolation avoids a Cartesian chord through the
planet, but does not avoid hills, buildings or trees. Sampled cameras are
checked against available source terrain and the ocean before GPU work. This
does not prove clearance between frames. Radial up is rebuilt for each view.

The path is authored choreography, not constant-speed physical travel or an
orbit integration. Zoom reuses the checked angular interpolation in `camera`.

Per-leg `smooth` easing stops coordinate motion at every waypoint. The original
six-fps ascent also changes its look target along the climb. Its repeated
slowdowns and changing framing are path choices, not evidence of dropped frames.

The [continuous ascent](island-ascent-continuous.yaml) instead requests
`interpolation: continuous`, linear key easing and 24 fps, with a fixed target.
Coordinate velocities agree at shared waypoints, using cubic interpolation and
the slower adjacent rate when both legs continue in the same direction.
Reversals meet at zero rate. Coordinates stay between each leg's endpoints.

This removes per-leg velocity resets; it does not enforce constant geographic
speed, constant ascent rate or a fixed viewing angle. Camera, target and lens
have continuous coordinate rates. Terrain clearance between frames, detail
transitions and temporal sampling remain separate checks. The full continuous
example is an input plan, not a claim that its 720 frames have been rendered.

Source hashes, decoding caches and native min/max indices are prepared once.
One bounded prototype cache is shared across frames; its face allowance applies
to all retained details together. View-dependent terrain meshes, placements,
density fields and GPU acceleration structures are still rebuilt. The cache
does not establish persistent GPU residency or a measured speedup by itself.

Every numbered frame directory retains `island.png`, its effective `scene.yaml`
and `render.json`. Root input files preserve the original scene and track;
`sequence.json` records their hashes, the producer, clock, preparation time and
completed frame hashes. Its completion flag stays false on a partial run. After
all frames and source closes succeed, the root poster and settings copy the
first effective frame, and root `sequence.yaml` resolves to those settings.

The command currently writes frames for an external movie encoder and refuses
existing destinations; sequence resume is not implemented in this study path.

The first native sequence is retained as `2026-11-09-sourced-island-ascent-320`:
60 actual 320 by 180 frames with four samples per pixel, encoded to a
ten-second, six-fps MP4. Its first camera is at 3,740 m radial height near the
observatory; the last is at 300,000 m. These are heights above the authored
reference sphere, not ground clearances. The one-hour scene clock reaches both
endpoints exactly.

Decode/count verification found all 60 movie frames; all PNG and
effective-setting hashes were checked.

Source/index preparation took 0.622 s once. The guarded run took 177.07 s;
sampled process-tree RSS reached 663 MiB. This is not GPU-memory accounting or
a controlled speed comparison. Across the records, 1,513 distinct tree IDs
had 69,697 repeated identity/position comparisons with no mismatch. The final
twelve frames aggregated the forest without root enumeration. Retained
prototype geometry grew from 500,132 to 593,492 stored faces within its budget.

The contact sheet shows the intended continuous widening from the dome to
the island, but intermediate terrain remains visibly faceted and the authored
height-colour boundaries are hard. Every frame reports a limited cell target.
Identity checks do not establish flicker-free LOD transitions or physical water.
The ocean remains smooth, the Sun fixed and the CORONA/Keyhole view unfinished.

Refining this terrain also exposed a preparation cost: each vertex's seam lookup
scanned every selected leaf. A new per-view edge index retains four references
per leaf and considers only matching lattice lines, preserving coarsest-level
and original-order tie rules. The command's explicit ceilings now permit 1,024
patches, 16,384 assessments and 524,288 sampled points for denser studies.

These are ceilings, not defaults or a quality guarantee; the target still
reports a budget limit. A matched render of ascent frame 33 after the index
change has exactly the same PNG bytes, patch list, source-point count and GPU
input bytes. That compares one actual view, not every seam case. Its
cold-command timing cannot be compared directly with a warm sequence frame.

The denser 320p study selected 1,024 patches and 261,313 source points. It
reduces the largest intermediate facets, but still reports a limited target.
Its guarded command took 4.85 s and sampled 567 MiB process-tree RSS. Both
studies retain their producer/source records and are linked from the local
hero gallery. Source-height shading normals and smoother authored landcover
remain useful next steps; more triangles alone do not remove every artifact.

## First local evidence, 2026-11-09

Later screenshots exposed unresolved appearance faults in the
snow/woodland study. Whole-tree hulls become conspicuous cones, and abrupt
geometry-to-flat-colour replacement removes stands while they remain visible.
The branch-clump floor removes the cone tier, but does not by itself solve
temporal transitions or supply aggregate canopy geometry. Further long movies
are deferred until matched views hold those transitions.

The 180-frame clump rerender finished before the attempted stop; its frames
remain diagnostic.

The dark ocean rings also reproduce with vegetation and mist removed. A matched
diagnostic changed only the secondary-ray reference length: greater separation
removes the rings. Planet-scale float32 self-intersections are implicated; these
are not simulated waves. A larger global offset also alters land shadows and is
not the proposed repair. The shader now tracks the originating sphere and
excludes only that sphere from outward secondary queries, using convexity.

Other occluders and inward rays remain tested. A matched native 320-pixel
comparison retains the small offset and removes the bands. In the recorded water
rectangle 2,967 of 14,400 pixels change; an interior land rectangle keeps all
1,456 pixels unchanged. A separate negative-sun-altitude control remains dark.
The image, exact settings and comparison record are retained in the local render
gallery.

These controls support this specific repair, rather than a general proof of
planet-scale intersection accuracy.

The island adapter provides optional `minimum-geometry-detail: crown`.
It reduces a seeded specimen's clump bounds into eight height slices and four
horizontal quadrants per material. Each occupied bin becomes an octahedron. Its original
material and height distribution survive; fine branch gaps do not. These are
ordinary shadow-casting GPU meshes shared by instances, not screen-facing cards
or a botany model. Empty planar bins contribute no volume.

The first crown builder visited every expanded triangle in a branch group. A
native-process stack sample identified that work during the long placement
phase; doubling the soft Go heap target did not materially shorten the frame.
The revised builder visits eight bounding-box corners per clump, without
expanding shared triangle payloads. The bounds intentionally fill some branch
gaps.

Crowns are retained and measured with the other detail prototypes before
candidate placement. Overhead-area calibration still intersects the original
clumps, rather than these simplified envelopes.

Optional `match-crown-material-area` requires the crown floor and an explicit
`crown-area-samples-side`. It probes each material separately in both the
original clumps and the coarse crown, using the same specimen-local rectangle.
Materials retain their original colour; names and colour channels do not
decide which geometry to reduce.

If a material has excess sampled overhead area, its crown is narrowed
horizontally about the root axis by the square root of the source/crown area
ratio. Exact continuous vertical projected area scales by that factor squared.
Finite grid probes only estimate those areas. Height and the conservative
visibility envelope remain unchanged; fit resolution belongs in the cache key.

Zero sampled area is unresolved, and a target needing expansion exceeds this
construction's remit. Both cases retain the original material geometry and
record the limitation. Render records retain each material's source, before
and after area, chosen scale, probe bounds and newly performed ray work.

These separate material areas overlap; their sum is not union cover or a
visible colour fraction. Narrowing does not match oblique silhouettes, gaps,
directional shadows or physical canopy reflectance. The aggregate-cover sampling
side remains independent of this fit's sampling side.

Four matched native 320 by 180 stills, at sixteen samples per pixel, compare
original clumps, unfitted crowns and two fit resolutions. All retain the same
23,704 tree positions. Against the clump reference, whole-image mean absolute
RGB code difference falls from 7.15 to 3.84 with the 32-side fit, and to 3.90
with the 64-side fit. These are display-code differences in one view, not a
physical or perceptual acceptance threshold.

The local material comparison retains all images, effective scenes, source
records and probe areas. The two fit resolutions give similar images here;
their ordering does not prove convergence. The woodland example requests the
32-side fit. Oblique silhouettes and moving transitions remain under review.

The fitted ascent study, `2026-11-09-island-material-ascent-320`, contains
180 actual native frames, four samples per pixel and a decoded thirty-second
MP4 at six fps. Its scene clock spans one hour; the camera rises from 3,740 m
to 300,000 m above the authored sphere. The guarded run took 389.52 s and
sampled 766 MiB of process-tree RSS. It used the material-fit producer before
the mist-floor cache was implemented.

Frame PNGs, effective settings and source bytes were checked by hash. Across
79,379 distinct root IDs, 1,328,747 repeated identity/position checks found
no mismatch. Two crown fits and aggregate-area estimates remain unchanged
across frames; all their probe work occurs in the first frame.

The sampled PNGs still show the small mist bank and limited intermediate
terrain detail. Stable metadata does not
certify smooth transitions, and the ocean remains a smooth proxy.

The revised woodland example lowers its aggregate cutoff from 40 to 8 in the
conservative projected-displacement criterion and uses crowns below the clump
tier. This criterion is an upper bound, not a measured tree height in pixels.
The work allowance is explicit: two million generated candidates, including
repeated boundary work. Selection batches at most sixteen cells per region axis,
reducing repeated halo visits.

Cutoffs of 2 and 4 exhausted the declared instance allowance in the first
comparison; those failed attempts are retained. At 8, the matched view retains
23,704 crown/clump trees where the previous cutoff drew none. This is stronger
appearance evidence, not a temporal pass.

`aggregate-area-samples-side` optionally samples each seeded clump specimen
from above on a fixed square grid. A ray counts once when any material is hit;
the estimated footprint area is bounding-rectangle area times the hit share.
Results are cached with the prototype, and the render record gives mean area,
estimated cover and new sampling work. Omission retains the old full-cover
colour blend. The example uses 32 by 32 rays per specimen.

For mean projected area A and roots per square metre rho, the distant cover
approximation is `1 - exp(-rho * A)`. With N independent uniformly placed crowns
over area S, a point misses every crown with probability `(1 - A/S)^N`; the
expression follows as S and N grow at fixed N/S. Slayer's bounded seeded cell
population is not that limiting Poisson process.

This is an explicit appearance approximation, using sampled authored geometry
rather than a field measurement or a fitted ecological constant. Finite
quadrature, grazing directions, branch gaps, directional shadows and transport
remain limitations; the result is not a radiometric match certificate.

Candidate preflight reuses each selected region's source-height envelope and the
full rotated specimen radius. Roots whose entire enclosure is invisible or
already below the cutoff need no individual height query. Such candidates are
counted separately; no invented ground point enters the root records. Retained
candidates still use the ordinary source-grounded placement path.

Canopy blend distances use the same declared envelope at every camera height,
including the whole-region aggregate path.

Render records separate total forest preparation, region selection (including
any newly loaded detail prototypes) and candidate placement. These are elapsed
wall times; selection and placement are parts of total preparation. Reduced
height-read counts alone do not demonstrate a render-time improvement.

An optional site budget, `mist-floor-cache-bytes`, retains source-projected
floor grids across layers and frames. Its key includes the fixed world pose,
horizontal bounds, grid side, projection tolerance and iteration limit. Camera
translation and weather time are excluded because they do not alter that floor.
The allowance covers retained float64 height payloads, not map metadata or grids
still referenced by the current frame.

A grid larger than the allowance is refused before allocation. Otherwise the
least recently used entries are evicted as needed. Zero preserves uncached
construction. Each frame records grid builds, hits, evictions, retained payload
bytes, source queries and floor preparation time. Closing the session releases
its retained floor entries.

Only the ground projection is reused. Weather still advances with scene time,
and its density grids and GPU inputs are rebuilt for every frame. This is a
bounded preparation cache, not cached fog illumination or persistent GPU
residency; a reduced query count alone is not a speed claim.

Paired three-frame native controls retain identical PNG bytes with the cache
disabled or enabled, while their weather images change over ten minutes of
scene time. The shared 65 by 65 floor retains 33,800 bytes. After the first
frame, both layers reuse it with zero new floor queries, instead of 22,202
source-height queries and about 4.5 ms of floor preparation per frame.

A second pair uses different grid sides under a 64 KiB allowance and forces
eviction. Its PNGs also remain identical to the uncached reference. A grid
larger than the allowance is refused by the parser. These controls cover
reuse, eviction and budget refusal, rather than whole-render acceleration;
the measured complete-frame times did not materially improve.

The committed scene path at ecbc487 rendered actual native-Metal 320 by 180
and 1280 by 720 studies on the M3 Max, with four samples per pixel. The wider
view shows the whole sourced island and its relief at an authored 300 km
camera height. It selected 256 patches and 16,641 source points; the cell-span
target remained limited. That status is preserved in its render record.

The 720p profile exposed a host-path mistake: the checked pinhole constructor
returns a pointer, while the GPU encoder only recognized a pinhole value.
It generated and uploaded 88,473,600 bytes of CPU rays unnecessarily. The
host fix at 08eb2ac recognizes the pointer and uses GPU ray generation.

Matched 720p inputs, sources and terrain selection were unchanged. Recorded
input bytes fell from 90,896,784 to 2,423,184. Six of 921,600 decoded pixels
changed, with a maximum RGB channel difference of 23/255; three pixels changed
by more than one channel code. This comparison used macOS sips to decode both
PNGs to bitmaps. It is a byte-difference observation, not a JND or temporal
acceptance result. Both source PNGs were inspected and retained.

One complete command measured 0.900 s before and 0.710 s after, including
source checks and image writing. A single pair does not establish throughput;
GPU trace time itself rose from about 28 to 57 ms. The established reduction
is the unnecessary ray input, not a universal GPU speedup.

The local render shelf keeps these under 2026-11-09-sourced-island-topology-320,
2026-11-09-sourced-island-topology-720 and 2026-11-09-sourced-island-pinhole-720,
with settings, source provenance and producer identities. The hero-stills page
links the studies. Focused Game builds passed; no Slayer unit tests were run
for this slice while other work was under way.

The full task still needs richer regional ecosystem appearance, validated
moving detail transitions, spherical sky/atmosphere, wave filtering and water
lighting, accepted source/datum joins, GPU residency and the CORONA/Keyhole
lookdown. The first source-grounded site and ascent are shared groundwork for
that complete shot.
