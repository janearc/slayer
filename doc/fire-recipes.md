# fires you can make

a fire recipe makes flame, optional smoke, and the light they cast without
requiring a burning tree. it can stand beside manufactured logs or burn on bare
ground. fuel geometry is placed separately; the fire does not consume it or
spread to its neighbours.

## describe and place a fire

start with [fire-study.yaml](fire-study.yaml), a single source, or
[fire-cluster.yaml](fire-cluster.yaml), nine sources with different heights.

both use the kilauea DEM pinned in `sources.yaml`. configure and populate
the shelf using the [Game workflow](developing.md), then draw the fixed preset:

```sh
game build
GOMAXPROCS=2 game run fire-cluster
```

`kind: fire` belongs to a named YAML recipe. it is not a static
`manufacture.KindNamed` entry. place it with `recipe: wood fire` under
`scene.set.things`, as the examples do. do not also set `alight: true`: the
recipe already burns.

existing campfire, bonfire and alight recipes keep their existing behaviour.

all configuration passes through Puzzlemouth and the bundled ecosystem schema.
unknown keys, wrong units and invalid models are errors. a flame requires radius
and height in metres, absorption at 550 nm in `m^-1`, and hottest and edge
temperatures in kelvin. these are independent inputs.

omit `layout` for one source. a cluster has `pattern: cluster`, `count` from 2
to 32 and a radius in metres. a line uses `pattern: line` and a full length in
metres; its local axis is x. `height-variation`, from 0 to 0.5, varies source
heights about the requested height.

source positions, variation and seeds are deterministic. cluster placement uses
an area-stratified disk, an authored arrangement rather than measured fuel.

overlapping sources add their absorption and emission coefficients. they share
one flame volume, so patches do not each consume a renderer volume slot. their
combined radiance is absorption-weighted; averaging their colours equally would
lose light.

the shared grid is capped at 192 cells along its longest side. very long fields
with tiny source patches can lose those patches between cells. place widely
separated fires individually to keep their local detail.

smoke is optional. when present, supply heat release, convective fraction, soot
yield, combustion energy, mass extinction, ambient temperature and density,
column height and albedo, with the units in the example. its base diameter
encloses the source layout.

a line still feeds a round plume; this model does not predict the flow above an
elongated fuel bed.

`light: none` removes external illumination and uses a black sky. `light-cells`
sets source quadrature from 3 to 8 cells per source axis. more cells cost more
lighting work; they do not increase flame grid detail. film exposure changes
picture brightness, not combustion temperature.

## what the example numbers mean

the examples are visual studies, not calibrated wood-fire experiments.
dimensions, 1750/900 and 1900/1100 K temperature pairs, optical thickness,
camera and exposure are authored choices.

so are the 40 kW input, 10 m smoke cutoff, 288 K air, 1.2 kg/m^3 air density and
albedo.

[NIST TN
2314](https://nvlpubs.nist.gov/nistpubs/TechnicalNotes/NIST.TN.2314.pdf) (2025),
abstract, reports a mean Douglas-fir radiative fraction of 0.29 and soot yield
of 0.24%, with substantial scatter and a particulate composition caveat.

the example's 0.71 convective fraction assumes the complement of that radiative
share; it was not independently measured.

section 2.6 uses 8.7 m^2/g at 632 nm for post-flame soot, with expanded
uncertainty of 1.1 m^2/g. the example's 10.6 m^2/g at 550 nm is the rounded
conversion `8.7 * (632/550)^1.39`, using Slayer's existing spectral tilt. the
report warns that smoldering aerosols need not share that extinction.

table 3, printed page 16, lists 21 MJ/kg as Douglas fir's net combustion energy,
distinct from the measured effective value of 17.5 MJ/kg. the example uses the
former as a nominal model input. it is not a measured energy balance for this
scene.

the flame emits as hot soot: absorption times black-body radiance, with
wavelength-dependent absorption.

its rise constants come from [B. J. McCaffrey, NBSIR
79-1910](https://nvlpubs.nist.gov/nistpubs/Legacy/IR/nbsir79-1910.pdf) (1979),
table 1, already pinned in Slayer's research data. those are time-averaged
buoyant diffusion-flame correlations.

shape, tongues, occupancy and cool rags are authored visual choices.

applying the mean temperature profile to individual moving tongues is an
approximation.

[Nguyen, Fedkiw and Jensen
(2002)](https://physbam.stanford.edu/papers/stanford2002-02.pdf) describe soot
radiation, the reaction zone and smoke in a flow simulation. Slayer does not
implement their combustion, flow or propagation solver.

smoke assumes a vertical plume in still air. its finite top is an authored
truncation. the width relation in `plume.Radius` remains explicitly recalled
rather than verified against its original publication.

flame radiation is not normalized to the smoke heat input; these parameters do
not close the total combustion energy balance.

## preparing and lighting a frame

`manufacture.Burn` checks a `BurnSpec`; `Place` returns volumes and light source
descriptions at the supplied time and uniform placement scale. the scene
converts those descriptions to emitters.

rebuild those light samples for each animation frame. the fields themselves are
immutable and support `Tick`. shutter samples can move the fields, but the
staged lamps remain fixed at the frame's construction time.

canonical flame density and emission grids are prepared together on Metal. the
GPU uses the same time, seed, rise constants and temperature-to-RGB table as the
CPU model. custom paint or emission keeps CPU preparation.

smoke preparation currently remains on the CPU; tracing both fields runs on the
GPU. this is a grid-based renderer, not a fluid simulation.

each sampled flame cell becomes a finite spherical light proxy, with the cell's
original far-field strength and half its diagonal as radius. this removes the
dark holes caused by treating a source cell like an opaque star with an excluded
interior.

existing `emit.Lit` keeps its surface exclusion.

for distance r and radius R, the scalar falloff is the volume average `F = (1/V)
integral_ball 1/|x-y|^2 dV`. integrating spherical shells gives, with `q = r/R`:

```text
F = 3/(2 R^2) * [1 + (1-q^2)/(2q) * log((1+q)/abs(1-q))]
F(0) = 3/R^2
F(R) = 3/(2 R^2)
F(r >> R) -> 1/r^2
```

these are derived limits, not fitted fire constants. CPU and Metal use near and
far series to avoid cancellation. the near-series relative remainder is below
1.1e-8; the far-series remainder is below 5e-7.

tests compare the result with direct volume quadrature. direction, surface
cosine and visibility still use the source centre, so nearby angular lighting
and occlusion are approximations.

## checks and remaining work

tests cover units, refusal, copied inputs, repeatable time samples, distinct
source phases, additive mixed-temperature overlap, GPU preparation, custom-field
fallback and finite-source lighting. actual Metal grids are compared with the
CPU model; the existing CPU/GPU fire image bounds remain.

wind response, crown fuel consumption, transported smoke and temporal
image-quality validation remain work. a short motion preview does not prove the
720-frame animation budget. keep the forest benchmark separate.

[validation notes](fire-validation.md) record checks and open review gates.

## night scenes and moving fronts

[the night-fire study](fire-night-design.md) adds sourced emission lines,
prescribed glowing firebrands, an opt-in smoke-lighting cache and ignition
fronts over placed stands.

[fire-night.yaml](fire-night.yaml) is the complete
scene, and [its sequence plan](fire-night-sequence.yaml) asks the Pzm sequence
runner for timed PNG frames.

the study separates measured line positions and fuel proxies from authored
strengths, motion and plume aggregation.
