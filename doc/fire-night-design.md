# fire in a dark grove

This is a dated development record. Its timings and check results belong to
the revisions named below. Use the [Game workflow](developing.md) for current
setup and commands, and [the committed 4K review](review-4k-2026-11-05.md)
for the later dense-forest measurement.

## scope and sequence

first make a large fire visibly light volcanic ground and nearby trees under
the existing Celestial-backed full-moon model. inspect stills before choosing
long-sequence settings. add repeatable glowing firebrands, then chemical
emission controls and a prescribed front through a large stand.

work through stills and short motion studies. measure 4K frame costs before
committing the remaining overnight budget to a minute, extending toward two
if measured runtime permits. the target remains 24 fps; resolution and sample
count must be recorded, and a preview cannot prove the 4K animation target.

## firebrands

use a seeded particle emitter with explicit source extent, emission rate,
lifetime, size, starting velocity, wind, loft velocity and drag time.
give particles stable birth identifiers. position and temperature at a time
must be independent of render order or skipped frames.

with prescribed constant air velocity u, gravity g and linear drag time tau,
solve dv/dt = (u-v)/tau + g analytically:
v_inf = u + tau*g
p(t) = p0 + v_inf*t + tau*(v0-v_inf)*(1-exp(-t/tau)).

this is an authored transport model, not a measured firebrand trajectory fit
or a combustion/spot-ignition solver. temperatures cool between explicit
endpoints over the prescribed lifetime. black-body radiance determines glow.

visible particles use shared small meshes and native GPU instances; avoid
adding a lamp and volume for each particle.

## composition

retain the existing thermal-soot default and temperature controls. distinguish
fuel heat/soot/smoke parameters from spectral-emission inputs. species
wavelengths require primary sources, with air/vacuum convention and units
retained.

line strengths or template mixtures must be identified as authored inputs
unless a population/excitation model actually derives them.

the GPU already takes a temperature-to-RGB table, so prepare composition
there without introducing a separate physical approximation in the shader.
copy immutable profile data; do not make Fire non-comparable by adding
mutable slices. no full chemistry/equilibrium solver is promised.

## prescribed fronts

define a moving ignition band over placed tree centres, not a new volumetric
fire per tree. batch nearby burning patches into bounded composite fields,
preserving their distinct seed and ignition age. the current GPU world limit
is 32 volumes.

optional smoke should aggregate sources at stand/front scale rather than
silently overflowing it.

a front is authored kinematics with speed and burn duration, not a prediction
of wildfire spread or a coupled wind/fuel solver. inactive trees remain
visible. source light and dynamic volumes must use the same frame time.

## checks

tests precede the implementation. check deterministic jumps in time,
birth/death boundaries, bounded particle counts, stable identifiers, thermal
endpoints, invalid units/values and preserved input ownership. compare actual
GPU instances and composition grids with the CPU model.

check front timing, independent ages and grouping without losing absorption
or emission. measure renderer limits and memory, inspect moon/fire contrast,
sparks, smoke, nearby tree light and shadows. keep the previous independent
physics/image bounds unchanged.

primary reading:

[WRF-Fire's guide](https://www2.mmm.ucar.edu/wrf/users/wrf_users_guide/build/html/fire.html)
separates firebrand generation, transport, physics and landing. its coupled
surface-fire and spotting solver is not implemented here.

[NIST's sodium table](https://physics.nist.gov/PhysRefData/Handbook/Tables/sodiumtable2.htm)
gives Na I wavelengths in air. handbook intensities are not fuel abundances
or absolute powers in a wood fire.

## first measured cache study

the opt-in `scene.volume-lighting: cached` setting requires `renderer:
gpu-native`. it caches the isotropic scattered source term on a separate grid
with 64 cells along the longest axis. it keeps the original density, opacity,
emission and ray march.

every frame rebuilds the light grid against that frame's lamps, volumes and
native mesh hierarchy.

only a canonical plume.Column with no paint or emission texture is eligible.
custom media and textures retain direct evaluation. the original direct mode
remains the default. interpolation softens sub-cell shadow detail; the cache
is an explicit approximation, not an exact replacement.

the 960x600, four-sample night study at 3 seconds had 180 introduced redwoods,
480 visible firebrands and 98 lamps. the zero-scattering fast path reduced
trace GPU time from 5.204 to 4.973 seconds with identical raw RGB.

caching then reduced trace GPU time to 0.482 seconds; the whole render call
took 2.579 seconds, including encoding, preparation and readback. these are
single-frame measurements, not a 4K animation result.

the cached image's sum of absolute raw RGB differences, divided by reference
RGB energy, was 0.0005215. this is not a perceptual maximum-error bound. the
smoke-only test uses a black floor and a moving coloured lamp, so flame or
ground brightness cannot hide a missing smoke source.

it measures about 0.001 relative RGB error at both tested times, below its
predeclared 0.02 limit.

## authored flame appearance

`intermittency-start` places the occupancy border as a height fraction.
`temperature-profile: hot-tongues` keeps occupied tongues hotter instead of
applying the mean axial cooling profile. both are authored choices.

zero Intermittency in Go and omitted YAML inputs preserve the old 0.4
occupancy border and mean temperature law. McCaffrey's measured mean regions
remain the source for rise speeds; they do not measure instantaneous hot
sheets.

## spectral source function

`flame.composition` supplies a continuum weight and explicit spectral lines.
each line keeps its species label, wavelength, air/vacuum convention, source
and integrated radiance in W/m^2/sr at the flame's hottest temperature.
below that temperature, an authored Planck ratio scales its power.

the continuum weight is not a soot mass fraction. the labels do not derive
excitation, abundance or equilibrium. opacity and soot yield still have their
own inputs. this source function is an artistic control with sourced line
positions, not a combustion chemistry solver or energy-closed fuel model.

RGB integration uses the supplied wavelength coordinate directly; it is not a
high-resolution air/vacuum spectroscopy calculation.

NIST's sodium strong-line table gives Na I at 5889.950 and 5895.924 angstrom
in air (588.9950 and 589.5924 nm), reference R56. the example's line strengths
are authored and do not use the handbook's relative intensities as powers.

## running a sequence

the complete example is fire-night.yaml. it places 500 introduced coast
redwoods on the Kilauea DEM, a stationary 24-patch fire and a front over the
grove. its physical moon direction stays fixed during the shot.

the offscreen moon has disc:false: moonlight and sky stay present while the
distant sphere is omitted. the later [woodland study](fire-woodland-result.md) fixed the general
near-shadow bias caused by mixing lunar and terrestrial geometry scales.

build through Game, then run:

    game run hillside -- sequence --plan doc/fire-night-sequence.yaml --dry-run
    game run hillside -- sequence --plan doc/fire-night-sequence.yaml

the plan asks for 2,880 distinct frames at 3840x2160, 24 fps, one ray per
pixel, starting at zero and lasting 120 seconds. this is a requested render,
not evidence that the long run has finished. paths in the plan are relative
to its directory. output goes to out/fire-night at the repository root.

the runner creates each PNG through a temporary file and writes sequence.json
only after the PNG has been renamed. --resume verifies recorded PNG hashes and
exact timestamps. it refuses changed plan or scene bytes, renderer build
stamps, executable bytes, source catalogue or the unpacked terrain file.

keep the same renderer executable when resuming. other external assets do not
yet have an exhaustive dependency manifest.

static scenery and thermal palettes stay in memory across frames. the native
hierarchy, scene encoding, density grids and lighting grids are still prepared
per frame. this is not the completed cross-frame resource cache in the caching
brief. new invocations rebuild scenery.

existing legacy campfire, bonfire and alight lamps are refused by the sequence
path because their light was sampled at construction; use volumetric fire
recipes instead.

## prescribed front controls

fire-fronts names a stand, one single-patch fire recipe and a placed object on
the starting plane. bearing is the direction the plane moves, clockwise from
north. speed is in m/s, starts and burn-duration in seconds, and
ignition-height is the flame base above each target's ground position.

group-width is an engineering choice in metres, not a measured fuel property.

fade-time is an explicit smoothstep interval for ignition and burnout, at most
half the burn duration. opacity and emitted coefficients fade together. each
patch retains its own ignition time and seed; skipping or reversing frame
order does not change the answer.

patches are grouped spatially in fields of at most 64. construction
conservatively refuses more than 32 simultaneous volumes or a scene budget
above 8,192 living firebrands.

firebrands stop being born when their source's burning window ends. airborne
particles continue their prescribed trajectories and cooling until their
lifetimes end. sources share one mesh and thermal palette within the render.

a front recipe's optional smoke is one aggregate plume. smoke-radius gives its
authored footprint; heat-release is the aggregate reference power, not power
multiplied by the number of targets.

its centroid is weighted by the active fuel's fade envelope, and its soot
yield scales by active weight over the plan's maximum simultaneous target
count. equal target fuel weights and the unchanging reference plume geometry
are modelling choices.

that shared column has no transported burn history. the front does not
predict wind-driven spread, fuel depletion, crown-fire transitions, char,
secondary ignition or tree collapse. tree geometry stays in place. the shot
is a rendering workload with prescribed motion, not a wildfire forecast.

## further measured checks

in the recorded 960x540 pair at 20 seconds, 0.13464506% of pixels moved
more than one OKLab JND under the smoke cache; the worst moved 2.28086897
JND. this compares cached and direct native lighting. the earlier
960x600 / 0.086458% statement had no matching record and is withdrawn.

it does not add up every approximation in the renderer or establish a temporal
error budget. the recorded still budget is at most 2% of pixels beyond one
JND.

the earlier manual zero-cache experiment has no preserved replay patch.
it is not acceptance evidence; the committed smoke-light tests are the
checks a reviewer can rerun.

scheduled-field tests compare actual GPU absorption-weighted emission with the
CPU at distinct ignition ages and fades, under the original 1e-4 relative grid
bound. custom smoke paint/emission retains direct evaluation.

the first 500-tree front study used 51 active fires and 1,128 sparks at 20
seconds. its one-ray 4K frame took 6.984 seconds including PNG output; GPU
tracing took 1.940 seconds.

a later, unobstructed four-ray 4K study had 47 active fires and 1,076 sparks
and took 12.677 seconds including output, with 6.365 seconds of tracing. these
single-frame studies preceded the final sodium profile and smoothly weighted
smoke.

the full Game functional suite passed after the first night slice. its overall
result still fails the previously recorded history rule at 03d6b0a. front
checks and final long-run evidence are recorded separately.

lint, mutation, prose and independent review gates are not all clean; these
local development commits are not a claim of merge readiness.
## completed sequence

[the run report](fire-night-result.md) records the completed two-minute
4K/24fps sequence, with frame hashes, measured times and output checks.
the earlier studies above remain the record of how the settings were chosen.
