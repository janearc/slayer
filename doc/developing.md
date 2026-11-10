# Working on Slayer with Game

Run Game from the Slayer checkout. The repository's `.game` names the
project, its source check and the commands you can run. Your Game config
keeps machine paths and remotes outside the repository.

## Toolchain and source shelf

Use the Go version in `go.mod`, currently Go 1.27.3. The Metal commands
and their tests need macOS, the Xcode command-line tools and a usable
Metal device. CPU packages can be used without Metal; a skipped native
test does not establish GPU conformance.

Add a Slayer block to your Game config, normally
`~/.config/game/config`:

```text
project slayer
    set SLAYER_SHELF /absolute/path/to/slayer-shelf
```

With `GAME_HOME` set, Game reads that home's `.config/game/config`
instead. Put the shelf outside every Git checkout. It holds the papers
and datasets pinned in `sources.yaml`; it is separate from Grissom's
survey directory and from rendered output.

Fetch and verify the shelf through the declared bootstrap runs:

```sh
game run sources-fetch
game run sources-check
game run sources-verify
```

These runs can start before Slayer's binaries exist. `sources-check`
holds the cached bytes to their hashes; `sources-verify` also checks the
recorded page or excerpt for each published numeric value. A quoted
number alone does not establish its physical role or units.

## Build and run

```sh
game build
game run hillside
game run smoke-one
game run observatory
```

`game build` verifies the source shelf first and stamps the commands in
`bin/`. The application runs use those binaries, so rebuild after changing
Go code. `bin/` and `out/` are local build and render output.

For application flags, use a Game build with argument forwarding:

```sh
game run hillside -- -night -alight 1 -out out/night.png
game run hillside -- -set doc/fire-woodland.yaml -out out/woodland.png
game run smokecheck -- smoke 9 -out out/smoke-nine
game run grissom -- -help
```

The separator belongs to Game; the words after it belong to the selected
application. Older Game builds discard those words. The fix is tracked in
[Game issue 2](https://github.com/janearc/game/issues/2). Until it is
installed, use the fixed runs above or describe a fixed run in `.game`.

A configured run names its working directory and program. For example:

```text
run my-scene . :: ./bin/hillside -set sets/my-scene.yaml -out out/my-scene.png
```

Then invoke `game run my-scene`. Run entries are executable argument
lists, not shell scripts: shell variable expansion and pipelines do not
happen inside them.

## Checks and resource limits

Start with one test package at a time on a machine sharing its GPU with
other work. This setting permits three Go workers while keeping packages
and parallel tests serial:

```sh
export GOMAXPROCS=3
export GOFLAGS='-p=1 -parallel=1 -timeout=60m'
export TMPDIR=/tmp
game check
game cover
game lint
game sweep
```

The long timeout accommodates the actual smoke renders on a cold cache.
It does not skip a test. `GOMEMLIMIT`, when set, is a soft Go heap target;
it does not bound native Metal allocations. Follow the machine owner's
queue and render policy before starting large tests or renders.

Game check runs formatting, vet and tests. Game cover runs the tests again
with coverage instrumentation and writes `bin/.cover/go.out`. Its report
is per source file across the checkout, including command entry points;
it is not a percentage for only the packages changed by one feature.

Game cover requires every measured file to reach 100%. Its current Go
implementation does not use `.game` omit or exclude entries to narrow that
report. Generated mocks and commands without package tests therefore appear
alongside renderer code, and can make the whole command fail.

Coverage measured in a package's own tests differs from coverage of that
package exercised by another package's integration tests. Request the
latter explicitly when that is the question:

```sh
GOFLAGS="$GOFLAGS -coverpkg=./..." game cover
```

This changes which packages Go instruments; it is not a waiver for their
unexecuted statements. Subprocesses need instrumented executables and their
own collected coverage too. The default command does not supply those.

Report the source revision and measurement mode with a percentage. A
union of several source-matched profiles is useful evidence, but is not
the result of a fresh, single Game cover run. An unexecuted defensive
return remains unexecuted even when its input-domain argument is sound.

The smoke gallery has its own cache, separate from Go's test cache.
`SMOKES_FRESH=1` requests fresh smoke renders. Cache hits validate stored
results; they do not execute the tracing work again. A coverage build can
also have a different executable fingerprint and a cold gallery cache.

## Harness commits and releases

Keep changes under `burning-trousers/` in commits separate from changes
to the renderer and the rest of the project. The history test checks
merge resolutions too. A merge cannot remove an earlier mixed commit
from the release candidate's ancestry.

If a candidate contains such a commit, prepare a clean replacement
history while retaining its file tree, then review the replacement ref.
Do not weaken the history rule to make a candidate pass. The harness's
[rules](../burning-trousers/INSIDE.md) describe the module-pin exception.

Celestial and celestial-data still use private review candidates. A release
needs their published revisions and corresponding pins; see
[review dependencies](review-dependencies.md). Only the owner releases
the dist. Preparing a local branch does not publish it.

See [Coverage](coverage.md) for the per-file gate, command instrumentation,
focused runs, and how to read the remaining gaps.
