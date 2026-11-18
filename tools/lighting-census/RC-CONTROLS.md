# Independent construction controls: follow-up study

The first numbered-smoke lighting matrix does not exercise the independent
RC construction controls. The following CPU-only entry points and the
proposed interaction study are separate work. No performance result or
accepted full-scene equivalence is claimed here.

Wait for a coordinated render-free window before running benchmarks. Pin the
revision and clean/dirty source state, build and Go version, exact settings,
scene and asset identities, machine load and memory pressure. Preserve each
result outside shared render output folders. Run through Game's tests lane
with the local two-worker limit. In a shell at the Slayer checkout:

```sh
export GAME_HOME=/path/to/game-home
export GOMAXPROCS=2
export GOFLAGS='-p=1 -parallel=1 -timeout=60m'
```

The existing bounded city construction benchmarks are:

```sh
game5 lane tests -- go test ./manufacture \
  -run '^$' \
  -bench '^(BenchmarkCityBatchPlacementAllocations|BenchmarkClusteredCityPlanAndVacantLandscape)$' \
  -benchtime=1x -count=3
```

The first has legacy nested, current nested and current fused subcases on the
same 192-building fixture. The second compares vacant-lot index on and off
for a seeded green clustered-city plan and its vacant landscape. Each reports
allocations. These are construction microbenchmarks, not full-city frame times.

`-benchtime=1x -count=3` limits each reported subcase to one operation per
repetition; setup and test-framework work also consume time and memory. Actual
wall time and peak memory on the current machine are not known until measured.

The authored anatomy benchmark is also explicit and CPU-only:

```sh
game5 lane tests -- go test ./improv/searching \
  -run '^$' -bench '^BenchmarkAuthoredAnatomySharingConstruction$' \
  -benchtime=1x -count=3
```

Its shared and separate cases use the same synthetic 16-being authored
population. It reports Go construction allocations and retained geometry
storage, excluding storage accounting from its timed section. It does not
render, test audio, or represent the older unique-geometry 3,067-being finale.
The existing `BenchmarkHistoricalMultitudeConstruction` measures that older
constructor; it does **not** select the authored sharing switch.

Before interpreting speed differences, run the exact geometry contracts through
Game. These tests construct scenes but do not benchmark or render them:

```sh
game5 lane tests -- go test ./manufacture \
  -run '^(TestCityBatchAddMatchesNestedPlacementExactly|TestCityAndClusteredCityHonorBatchPlacement|TestClusteredCityVacantLandscapeMatchesFullGridScan)$' -count=1
game5 lane tests -- go test ./improv/searching \
  -run '^(TestAnatomySharingBenchmarkUsesTheSameValidPopulation|TestAuthoredMultitudeCanDisablePrototypeSharingWithoutChangingScene)$' -count=1
```

The city batch tests compare points, faces, material chunks and placements in
small fixtures. The vacant-lot test compares building selection, vacant-lot
order and landscape geometry. The authored-sharing test compares geometry,
materials, local poses, storage and three animation times in a small fixture.

They do not prove output equality for every full-size job. More expensive
catalogue set and job-provenance tests are in `set/city_batch_placement_test.go`
and `set/city_vacant_lot_index_test.go`.

## Interactions still to implement

1. For each eligible green and night clustered-city recipe, construct the
   four explicit combinations of `batch-placement: fused|nested` and
   `vacant-lot-index: on|off` using the same seed, detail and scene inputs.

   Eight configurations across the two recipes are a proposed minimum, not
   results. Add a joint fixture and test that the two controls actually reach
   construction and preserve geometry, materials and placements; the current
   individual benchmarks do not measure their interaction.
2. Measure authored anatomy sharing on and off using the same valid population
   at representative scene times. The small benchmark supplies construction
   timings and allocations; add isolated real frames and inspect RGB, coverage,
   depth and motion before claiming a render outcome. It does not combine
   with city controls merely because both have two choices.
3. Only for an actual job that composes eligible city and authored anatomy
   content, expose and validate all three settings in that job and construct the
   valid 2 × 2 × 2 cases. Keep unsupported combinations as explicit refusals.
   Add no imagined cross-product when the authoring path cannot express it.
4. Compare production-quality, unchanged geometry on exact settings first;
   separately compare each lighting mode to its matched reference. Record cold
   and retained paths, source keys, complete outputs and musical partials where
   a film has music. Do not describe a different resolution, missing light,
   simpler geometry or silent preview as a speedup of the same work.

Pair or alternate case order across independent runs to reduce warm-cache and
background-load bias. Report wall time, sampled CPU time, allocations, host
and device memory as distinct quantities, and GPU stage times where available;
none is a guaranteed peak unless measured as one. Use pilots to estimate a
real schedule and leave the GPU lane available to existing jobs.
