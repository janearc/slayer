# AGENTS.md

Guidance for agents working in this repository. Slayer is a Go ray tracer with
procedural geometry, volumetric light, astronomical scenes, a Metal renderer
for macOS, and a photographic darkroom. There may have been ninjas involved in
the darkroom. The documents below are the authoritative map; this file covers
only what they do not put up front.

## Read first

- OPERATION.md: the rendering pipeline (scene, trace, develop, output).
- PACKAGES.md: package responsibilities and import boundaries.
- doc/developing.md: the Game workflow this file summarises.
- EXAMPLES.md: scenes and application flags.
- DEBT.md: the remaining style migration.

`doc/` mixes implemented interfaces with historical design notes; proposed
syntax and archived commands there are not live API. Package exports and
current command help define the live interface.

## Work through Game, not raw Go

Development runs through Game, driven by the `.game` file in the repo root:

- `game build` builds and stamps binaries into `bin/`. It runs a pre-build
  source verification (`go run ./cmd/sources verify`), which reads the
  committed `shelf/` unless `SLAYER_SHELF` names another.
- `game run <name>` runs the named runs declared in `.game` (hillside,
  smoke-one, observatory, check-film, ...). Application runs use the `bin/`
  binaries: rebuild after changing Go code.
- `game check` runs formatting, vet and tests. `game lint`, `game cover`
  and `game sweep` are the other gates.
- Application flags follow `--` (for example `game run hillside -- -night`),
  which needs a Game build with the argument-forwarding fix (Game issue 2).
  Older Game builds silently discard those words.

The source shelf holds the papers and datasets pinned in `sources.yaml`.
It is `shelf/` in the tree unless `SLAYER_SHELF` (an absolute path, set in
the Game config under the Slayer project block) names another; fetch
needs it named:

    game run sources-fetch
    game run sources-check
    game run sources-verify

Use the Go version in `go.mod`. Slayer's own dependencies are released
versions and come through the public Go proxy; a fresh clone needs no
preparation.

## Test settings and gates

On machines sharing a GPU with other work, keep packages and parallel tests
serial:

    export GOMAXPROCS=2
    export GOFLAGS='-p=1 -parallel=1 -timeout=60m'
    export TMPDIR=/tmp

- The 60m timeout accommodates cold-cache smoke renders; it skips nothing.
- `game cover` requires every measured file to reach 100%, across the whole
  checkout. `.game` omit/exclude entries do NOT narrow the report; generated
  mocks and commands without package tests can fail the whole run. Report
  any coverage percentage with the source revision and measurement mode.
- Metal tests need macOS, the Xcode command-line tools and a usable Metal
  device. A skipped native test does not establish GPU conformance.
- Some tests need external data (for example terrain's Kilauea lidar) and
  skip without it.

## Byte-stability machinery (do not break casually)

- `manufacture/b_geometry_reference_contracts_test.go` (darwin/arm64) holds
  authored geometry to sha256 digests committed in
  `manufacture/testdata/*-reference-arm64.txt`. The digest hashes bounds,
  poses, mesh faces, and each thing's material name plus Colour/Glow RGB,
  Gloss and Index, so renaming a material or touching
  `manufacture/ground-choices.yaml` breaks the reference tests.
- Stage reference rows regenerate only when `STAGE_REFERENCE_OUT` is set.
- The smoke gallery has its own cache, separate from Go's test cache;
  `SMOKES_FRESH=1` requests fresh renders.
- Smoke scenes are recorded rendering milestones; their numbers stay stable,
  and a change to an existing smoke is a separate commit.

## Renderer map

`trace` is the CPU tracer used by `slayer`, `ring` and `swarm`. The old
`render/trace` is gone. `gpu` supplies Metal software/native traversal,
retained frames and explicit lighting shortcuts. `hillside` connects jobs
to scene sessions; Omnipositor also exposes separate surface/volume passes.
Read [the renderer guide](doc/renderer.md) and
[optimization catalogue](doc/optimizations.md) before changing these paths.

## Harness and commits

- Changes under `burning-trousers/` (the independent physics harness) go in
  commits separate from changes to the renderer. The history test checks
  merge resolutions too, and the rule is never weakened to make a candidate
  pass.
- Mocks are generated with `go tool mockgen` (pinned in go.mod as a tool)
  through the `mocks` runs in `.game`; the generated files are committed.
