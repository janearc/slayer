# Working on Slayer with Game

New contributors: start with [Building Slayer](building.md), which explains
Game, why the project uses it, prerequisites and the first build. This
guide covers the development gates and their evidence in more detail.

Run Game from the Slayer checkout. The repository's `.game` names the
project, its source check and the commands you can run. Your Game config
keeps machine paths and remotes outside the repository.

## Toolchain and source shelf

Use the Go version in `go.mod`, currently Go 1.27.3. The Metal commands
and their tests need macOS, the Xcode command-line tools and a usable
Metal device. CPU packages can be used without Metal; a skipped native
test does not establish GPU conformance.

Verification and checks read `shelf/` in the tree, and need no setup. To
fetch, or to keep your shelf elsewhere, add a Slayer block to your Game
config, normally `~/.config/game/config`:

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

`game build` verifies the sources, from `shelf/` in the tree unless
`SLAYER_SHELF` names another, first and stamps the commands in
`bin/`. The application runs use those binaries, so rebuild after changing
Go code. `bin/` and `out/` are local build and render output.

For application flags, use a Game build with argument forwarding:

```sh
game run hillside -- -night -alight 1 -out out/night.png
game run hillside -- -set doc/fire-woodland.yaml -out out/woodland.png
game run slayer -- assess --smokes 9 --out out/smoke-nine
game run grissom -- --help
```

The separator belongs to Game; the words after it belong to the selected
application. Older Game builds discard those words; with one of those, use
the fixed runs above or describe a fixed run in `.game`.

A configured run names its working directory and program. For example:

```text
run my-scene . :: ./bin/hillside -set sets/my-scene.yaml -out out/my-scene.png
```

Then invoke `game run my-scene`. Run entries are executable argument
lists, not shell scripts: shell variable expansion and pipelines do not
happen inside them.

## Checks and resource limits

Start with one test package at a time. This setting permits two Go workers
while keeping packages and parallel tests serial:

```sh
export GOMAXPROCS=2
export GOFLAGS='-p=1 -parallel=1 -timeout=60m'
export TMPDIR=/tmp
game check
game cover
game lint
game sweep
```

The long timeout accommodates the actual smoke renders on a cold cache.
It does not skip a test. `GOMEMLIMIT`, when set, is a soft Go heap target;
it does not bound native Metal allocations.

Game check runs formatting, vet and tests.

The smoke gallery has its own cache, separate from Go's test cache.
`SMOKES_FRESH=1` requests fresh smoke renders. Cache hits validate stored
results; they do not execute the tracing work again.

Slayer's own dependencies are released versions and come through the public
Go proxy.
