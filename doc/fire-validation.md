# fire recipe validation

This is a dated development record. Its timings and check results belong to
the revisions named below. Use the [Game workflow](developing.md) for current
setup and commands, and [the committed 4K review](review-4k-2026-11-05.md)
for the later dense-forest measurement.

state: 2026-11-03. this is a local development checkpoint, not a release or an
assertion that the full acceptance bar has passed. the implementation and its
source boundaries are in [fire recipes](fire-recipes.md).

## functional checks

Game builds the commands. the full Game check passes every functional package,
including the GPU, hillside, set, flame and manufactory tests. it still fails
the existing harness-history rule on commit `03d6b0a`. the independent harness
and that rule were not changed.

the first full run caught three regressions, fixed before this checkpoint: a
forbidden emitter import in the manufactory, a missing distant-lamp buffer
field, and an expected error message missing the new light mode. source
descriptions now cross the manufactory boundary; staging makes lamps.

focused tests cover the published YAML examples, invalid units and models,
copied inputs, deterministic source layouts, animated fields, coefficient
addition at overlaps, finite-source lighting and custom-texture fallback. a
rapid property test checks additive transport across 50 seeded samples.

actual Metal grid tests compare density and emitted coefficients against the CPU
model at several seeds and times, including a 64-bit seed. tested mean relative
errors are below one part per million, against a predeclared 1e-4 bound.

the existing CPU/GPU fire image test retains its original limits: mean relative
differences are 0.0027128312 at 96 cells and 0.00091899794 at 192 cells. the
latter remains below 1.5e-3 and half the former.

## tests of the tests

the four earlier manual fault trials have no preserved replay patches. their
claimed outcomes are withdrawn as acceptance evidence. tests of overlap, source
scaling, GPU fields and finite-light quadrature remain committed checks,
distinct from those unrecorded experiments.

a narrow Game mutation run for `emit/volume.go:volumeFalloff` ran 79 mutants: 76
killed, three surviving, none timed out or failing to build. the survivors
change numerical branch cutoffs: `<= .25` to `< .25`, `>= 4` to `> 4`, and `>=
4` to `>= 5`.

these choose the closed form over a numerical series at a boundary or over a
short interval. they have not been classified in `.mutants`; the run is not a
zero-survivor pass. it exposed missing radius-scaling, near-integral and
far-cancellation checks, added without changing the falloff implementation.

coverage was measured with the focused fire tests, not the whole suite. new
constructors and most transport methods have high statement coverage;
parser/staging guard and GPU error paths remain uncovered. whole-project 100%
coverage and the full new-code mutation sweep have not been established.

## visible results and remaining work

actual GPU stills and a short 24 fps sequence were saved outside the repository.
the preview varies source heights, soot detail and lighting with time. checking
a few images is not a temporal error measurement.

the broader 1920 by 1200 study, four samples per pixel, took 7.989 seconds
including construction and PNG output on an M3 Max; GPU tracing was 5.345
seconds. this is a small fire scene, not the 4K forest animation benchmark.

flame detail remains soft. wind response, GPU smoke preparation and more
convincing fuel interaction remain work. smoke is traced on the GPU but its grid
is still prepared on the CPU. no propagation solver was added.

style and review gates remain separate from functional correctness. repository
lint, sweep, shuriken findings and final review remain open. local Game claiming
needs a project block with the road and dist.

shuriken found an open live light description; it is now private and copied
through `Point`. the remaining findings include public input descriptions,
literal construction of existing math/scene types and naming suggestions. they
are recorded for review, not declared resolved.

checks use the Grissom workspace and its companion modules, with the verified
source shelf in `SLAYER_SHELF`. no dependency versions were changed.

the motion preview contains 24 frames at 960 by 600, 24 fps, one second. serial
rendering and encoding took 101.326 seconds. PNG frames were kept; two inspected
time samples visibly differ. this is not temporal validation.

full emit/flame package coverage measured 99.1% and 97.0%, respectively. all
finite-light functions and Body sampling methods were covered. that does not
establish whole-project 100% coverage.

## 2026-11-04 continuation

the [completed night run](fire-night-result.md) contains 2,880 distinct 4K
frames at 24 fps, two minutes of animation. it adds prescribed fronts,
firebrands, sourced spectral controls and an optional smoke-light cache. the
report records settings, wall times, image differences and limitations.

the front's CPU/GPU checks retain distinct ignition ages and smooth fades.
custom paint and emission retain direct evaluation. the manual zero-cache fault
trial has no preserved replay patch and is not acceptance evidence. actual
same-input 4K checks confirm the faster PNG path preserves every decoded pixel.

every saved frame hash and timestamp was verified, and both videos report 2,880
decoded frames and 120 seconds. this establishes the delivered sequence, not a
complete temporal error budget or a clean review gate.
