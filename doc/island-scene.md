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

The stored range is -5648.2431640625 to 4197.93603515625. The example's round
height envelope encloses that full range, rather than estimating peaks from
sparse queries.

The delivered TIFF's no-data text is -5649.243, not the NetCDF's -9999;
comparison uses its float32 representation. No vertical CRS or unit GeoKeys
are present. Metres, positive up and MSL EPSG:5714 come from NOAA's retained
product guide and NCML. Grid spacing is not a promise of equally fine measured coastal detail.

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

Optional `edge-fade-metres` gives a bank elliptical horizontal support and a
smooth outer transition; zero retains rectangular support. This is an
authored shape, not a condensation model.

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

Faceting, thin-foliage aliasing at four samples, the uniform background and
local weather support are visible approximation limits.

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

`forest.adaptive-regions` enables hierarchical support selection before cell
enumeration. It first uses the declared full specimen envelope; a wholly
subpixel region needs no individual roots. Nearer views measure all permitted
variant/detail bounds before using a tighter, rotation-safe envelope. Native
source extrema tighten regional height bounds.

Region assessments and actual cell queries share the explicit cell allowance;
candidates and instances keep their own limits. This does not make off-screen
shadow influence complete. The pixel thresholds apply to the conservative
projection criterion, which can exceed a tree's literal image height. The
example uses 8, an authored quality/work choice.

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
authored scenery.

At 300 km the whole support aggregates, with zero cell visits and zero generated
candidates.

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
transitions and temporal sampling remain separate checks.

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

The command's explicit ceilings permit 1,024 patches, 16,384 assessments and
524,288 sampled points for denser studies. These are ceilings, not defaults or
a quality guarantee; the target still reports a budget limit.

The shader tracks the originating sphere and excludes only that sphere from
outward secondary queries, using convexity. Other occluders and inward rays
are still tested. This does not prove a general bound on planet-scale
intersection accuracy.

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

The woodland example requests the 32-side fit. Oblique silhouettes and moving
transitions are not matched by it.

The woodland example sets its aggregate cutoff to 8 in the
conservative projected-displacement criterion and uses crowns below the clump
tier. This criterion is an upper bound, not a measured tree height in pixels.
The work allowance is explicit: two million generated candidates, including
repeated boundary work. Selection batches at most sixteen cells per region axis,
reducing repeated halo visits.

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
