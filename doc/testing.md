# Testing Slayer

Start with the smallest check that exercises your change. Slayer includes
quick host tests, real image/audio/device tests, an independent physics
harness and geometry records. Running every package with every draw enabled
is a deliberate expensive gate, not the first command for a contributor.

## Before the first test

Complete [build setup](building.md), including the private review dependency
pin in this development branch. Use compatible Game from the checkout root.
On the shared development machine:

```sh
export GOMAXPROCS=2
export GOFLAGS=-p=1
export GOWORK=off
```

Do not set `-tags=draw` globally. It enables expensive tests for every Go
invocation that inherits it. Keep cached dependencies and source data;
deleting them makes a cold run slower and can prevent offline work.
`GOMEMLIMIT` is a soft Go heap target, not a cap on Metal memory.

## Choose your first command

| What changed | A useful first scope |
|---|---|
| job format or clocks | `game test -- -count=1 ./job ./cuesheet ./beats` |
| a procedural model | selected construction tests in `./manufacture` |
| set parsing | `game test -- -count=1 ./set` |
| a pure ray/volume calculation | selected tests in `./trace` or `./maths/...` |
| a GPU kernel | one named real-device test with `-tags=draw` |
| Azazel logic | selected private package tests; see [Azazel](azazel.md) |
| bootstrap | `game run check-bootstrap` |
| prose | links, examples and `game lint --look` |

For example, these exercise the Larson's construction without rendering:

```sh
game test -- -count=1 ./manufacture -run 'TestTheLandedLarsonStandsOnItsGear|TestTheFlyingLarsonCarriesNoGear'
```

Use `-v` when selecting names: a green command saying “no tests to run”
establishes nothing. `-count=1` bypasses Go's test-result cache, not the
separate smoke gallery or rendered-frame caches.

## The tiers

| Tier | Command | Meaning |
|---|---|---|
| focused host | `game test -- ...` | selected packages/names, without draw tag |
| hot path | `game run test-hot` | host packages listed in `tools/tiers/hot.txt` |
| all host | `game check` | formatting, vet and non-draw tests |
| selected draw | `game test -- -tags=draw ... -run REGEX` | only the requested real draws |
| draw-only suite | `game run test-draw` | names absent from the plain build, serial packages |
| every test | `game run test-all` | host and draw tests together |

Draw tests open devices, stage scenes, render images/audio/movies, use the
source shelf or perform other slow work. Files use `//go:build draw`,
combined with platform tags where needed. Some package-wide GPU suites
have no tests in their plain build. The detailed
[tier contract](test-tiers.md) records placement rules and exceptions.

Plain tests avoid the general render suites; this does not remove compiler,
platform, dependency or small example-fixture requirements. The current
[integration record](integration-2026-11-14.md) gives a dated inventory;
use the inventory tool for the current checkout, not a fixed test count.

## Run a real GPU sample

Metal needs macOS, the Xcode command-line tools and a usable device.
Native queries need their supported ray-tracing API too. This small
example exercises both software and native compositor passes:

```sh
game test -- -tags=draw -count=1 -parallel=1 -timeout=5m -v ./gpu -run '^TestCompositePassTransport$'
```

Game coordinates its test and GPU lanes. Do not wrap `game test` inside
an outer GPU lane. For application renders, use `game lane gpu -- COMMAND`.
Named `.game` runs are explicit commands; follow the machine's scheduling
policy when invoking broad runs that execute Go directly.

A skipped Metal test is not GPU conformance. Record device unavailability
as a limitation. Avoid parallel rendering packages: native lifetime tests
and memory observations can fail or mislead under unrelated GPU work.
Full draw runs have hour-scale per-package timeouts for a reason.

## What makes a useful assertion

For geometry, test dimensions, orientation, closed/outward surfaces, physical
bounds, attached lights and deterministic variation. For transport, compare
against an independently derived integral or controlled CPU reference.
For an optimization, prove its invalidation and result contract rather
than merely checking that a cache says it hit.

An anti-test makes a nearby wrong case distinguishable. Examples include
a stair rail intruding into a hull, a needle detached from its twig,
or a cache key missing a changed camera. Keep its marker on a line alone:

```go
// anti-test of TestTheValidShape
func TestTheInvalidShapeIsRefused(t *testing.T) {
    // Exercise the same contract with a meaningful invalid case.
}
```

This is marker syntax, not a complete test to copy. Inspect existing tests
before adding duplicate negatives. `tools/antitests` counts named relations;
`tools/antiproof` tests a retained deliberate break and restores sources.
Use a clean isolated tree for mutation work, a precise scope and a timeout
based on the unchanged test. Broad mutation sweeps are not an iteration gate.

## Physics and geometry references

`burning-trousers` independently checks renderer exports against physics
and authored expectations. Keep its commits separate from the renderer;
the history rule is part of the test. Selected-pixel Fermat replay is a
different check: deterministic replay does not establish physical completeness.

Manufacture signatures on Darwin/ARM64 include geometry, placement and
material properties. A material rename can fail them. Read
[record provenance](../manufacture/testdata/README.md) before changing
references. `STAGE_REFERENCE_OUT` enables reference production; do not
set it during ordinary verification. Never update a digest to conceal an
unexplained geometric change.

## Coverage and caches

`game cover` reruns instrumented tests and enforces per-file coverage.
It is a separate, potentially broad run. A repository average, selected
package result or merged profile is not a fresh whole-tree passing gate.
Go coverage measures host Go statements, not Metal shader branches.

`-coverpkg=...` changes what Go instruments. Cross-package calls and
instrumented subprocesses need matching scopes to contribute. Whole-program
instrumentation can change conifer vertex bits in this checkout; do not
rewrite geometry references to conceal that distinction. Read
[coverage](coverage.md) for command children and remaining gaps.

The smoke gallery has its own executable/input cache. `SMOKES_FRESH=1`
requests real smoke work; do this deliberately for expensive scenes.
Film resume and section reuse also save work independently of Go's cache.
Inspect actual trace counters when measuring a speedup. See [caches](caches.md).

## Add tests without losing the old ones

```sh
game lane tests -- game run check-tiers
```

This lists tests with and without the draw tag; it does not run their
draws. A new test initially appears as an addition. Preserve the old
`tools/tiers/before.txt`, snapshot to another file through Game, compare
lost/added names, then replace the saved inventory only for intended changes.
The exact tool options are in [test tiers](test-tiers.md).

## Report what actually ran

Record the commit, toolchain, command, platform/device, sampling and flags,
passes/failures/skips and cache state. For a picture comparison, keep raw
light and developed error separate. Inspect motion for temporal claims.
List checks you could not run and their specific reason.

For a small change, focused tests plus a meaningful GPU sample can be the
right iteration evidence. Full host/draw/release gates still have their
place; agree on their cost and machine availability before starting them.
An existing lint or history failure must be reported, not waived or
described as a passing whole-tree check.
