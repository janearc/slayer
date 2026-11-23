# Building Slayer

For the automated path, run `./bootstrap.sh --check` first, then
`./bootstrap.sh`. See [Bootstrap](#bootstrap) below for exactly what changes.

Slayer builds through **Game**, the project runner used across these
repositories. Game reads the committed `.game` file for source requirements,
build steps, named runs and checks. Your own configuration supplies paths
and remotes. Run Game from the Slayer checkout root.

## Why we use Game

Slayer needs to know which code produced an image and whether the published
data behind its physical values are available and verified. Game makes
those requirements part of ordinary work:

- The `pre-build` target verifies the source shelf before compilation.
- Ordinary builds stamp binaries with revision and builder information.
- Named runs provide shared entry points instead of private scripts.
- Checks, coverage and lint enforce project gates.
- Lanes coordinate expensive tests and GPU work on a shared machine.

Game runs the work; Slayer renders and `ffmpeg` encodes movies. The `.game`
file is readable text: inspect it to see what each target actually does.

## Tools and inputs

| Requirement | Purpose |
|---|---|
| Git | checkout and exact source revision |
| Go version in `go.mod` | this checkout requests Go 1.27.3 |
| compatible Game v0.5.0 or newer | reads this checkout's `.game` |
| macOS, Xcode command-line tools, usable Metal device | Metal rendering and GPU tests |
| `ffmpeg` on PATH | movie encoding and audio decoding |
| source shelf | normal pre-build verification |

CPU-only rendering does not require Metal. Metal kernels compile at runtime;
a successful Go build does not establish GPU conformance. Native mesh
traversal needs the supported Metal ray-tracing API too.

Install [Game](https://github.com/janearc/game) using its distribution
instructions, or use its source checkout:
`sh bootstrap.sh` builds and checks a local `bin/game`. Adding `install`
changes your installation; it is optional. Some development installations
name the executable `game5`; substitute that name in these examples.

```sh
command -v game
game --age
go version
```

Use a current Game with argument forwarding. In `game run NAME -- ARGS`,
the words after `--` reach the application. Older builds discarded them.

## Configure the source shelf

The shelf holds the papers and datasets pinned in `sources.yaml`. A
release does not include it, so name a folder of your own, outside any
Git checkout, in your shell:

```sh
export SLAYER_SHELF="$HOME/slayer/sources"
game run sources-fetch
game run sources-check
game run sources-verify
```

Put the `export` in your shell's startup file so later builds find the
shelf. Fetch downloads about 2 GB, most of it one elevation model of
Kilauea, and keeps what it already has. The shelf is separate from
renders and Grissom survey data.

You can name the shelf in your Game config instead of the shell. Game
passes a `set` value to its runs only for a variable the repository
declares, and Slayer declares that it wants `SLAYER_SHELF`:

```
set SLAYER_SHELF /absolute/path/to/slayer/sources
```

That reaches what Game starts: `game check`, `game build` and `game run`.
A command you start yourself, such as `go test` or `slayer draw`, reads
your shell.

With `SLAYER_SHELF` unset, verify, check and path read `shelf/` in the
checkout, which only a working tree of the road has.

These bootstrap runs work before binaries exist. Fetch needs source access;
preserve the downloaded shelf for offline use. Check holds bytes to hashes.
Verify also checks source pages or excerpts behind published numeric values.

## Beach surveys

The measured beach surveys are too large for the tree, and a release does
not include them. The `scene/beachsurvey` package reads them from the
`beachsurvey` folder inside the folder `SLAYER_DATA` names: one `.json`
page and one `.bin.gz` height grid for each survey.

```sh
export SLAYER_DATA="$HOME/slayer/data"
```

With `SLAYER_DATA` unset, a survey is refused with a message that names
the variable, and the tests that read a survey are skipped. Slayer
declares that it wants this one too, so a `set SLAYER_DATA` line in your
Game config reaches what Game starts.

## Dependencies

Slayer's dependencies come through the public Go proxy.

Set `GOWORK=off` when checking a standalone checkout so an enclosing
workspace cannot silently supply different modules.

## Build and run

Build, then check the binary and a dry run:

```sh
game build
bin/slayer --age
game run slayer -- --help
game run slayer -- draw doc/examples/jobs/first-picture.yaml --dry-run
```

`game build` verifies the shelf and builds stamped commands into `bin/`.
Application runs use those binaries: rebuild after changing Go code.

A named target such as `game build camera-binary` executes the custom
recipe in `.game`. Its raw `go build` does not automatically receive the
ordinary build's provenance stamp. Use the ordinary build for binaries
retained as render evidence, and check their `--age`.

Named runs are argument lists, not shell scripts: pipelines and shell
variable expansion do not happen inside them. The separator in
`game run hillside -- -help` belongs to Game; `-help` belongs to Hillside.

Continue with [your first picture](getting-started.md).

## Check at the appropriate scale

Use focused tests while developing:

```sh
game test -- -count=1 ./job ./beats ./cuesheet
```

Game coordinates tests and GPU access. Select a real GPU check with
`-tags=draw`, a precise `-run` and its package. Do not wrap `game test`
in an outer GPU lane: it obtains lanes itself. An application render uses:

```sh
game lane gpu -- game run hillside -- -set doc/observatory-ecosystem.yaml -out out/observatory.png
```

Confirm the shared queue is available before substantial rendering.

| Command | Scope |
|---|---|
| `game check` | formatting, vet and non-draw tests |
| `game run test-hot` | host tests of selected hot-path packages |
| `game run test-draw` | all draw-only tests, serial packages |
| `game run test-all` | every test, including real draws |
| `game run check-tiers` | test inventory consistency |
| `game cover` | instrumented tests and per-file coverage gate |
| `game lint --look` | configured lint findings |
| `game sweep` | distribution checks of the committed tree |

Full draw and coverage runs can be expensive; choose them deliberately.
Cache hits and skipped Metal tests do not establish fresh GPU execution.

## Common setup failures

| Symptom | Check |
|---|---|
| unknown `.game` version or `gomaxprocs` key | selected Game executable and `--age` |
| source verification fails | shelf path, pinned bytes and source fetch |
| missing dependency symbol | revision, `go.mod` and `GOWORK` |
| Metal unavailable at render time | Mac tools, device or a supported CPU alternative |
| movie encoding fails | `ffmpeg` on PATH |
| flags ignored | current Game and `--` before application arguments |

Keep the exact error and revision with a report. Repair the setup rather
than disabling source verification or deleting useful dependency caches.

## Bootstrap

The executable `bootstrap.sh` gives this checkout a repeatable setup path:

```sh
./bootstrap.sh --check
./bootstrap.sh
```

Check mode performs prerequisite inspection without downloads, builds or
writes. It prints the requested Go/Game versions, local Go version and
chosen shelf/setup paths. It catches missing tools and the current missing
local Celestial replacement. It does not prove module availability,
source hashes, toolchain compatibility or a usable Metal device.

Setup requires Git, Go and a C compiler already installed. On macOS,
install the Xcode command-line tools with `xcode-select --install` if
needed. Install `ffmpeg` for movie and audio workflows; pictures can be
built without it. The script does not invoke a system package manager.

It chooses Game from `GAME`, then `game`, `game5`, or its own cached copy.
If none exists, it clones Game from its official repository into its
setup directory and runs Game's bootstrap, including Game's own checks.
A present but incompatible Game is an error to fix, not silently replaced.

The script creates its own Game home and configuration, downloads module
inputs through a Game test lane, fetches/verifies sources, then performs
an ordinary Game build. It limits Go to two workers with serial package
builds. It does not run Slayer's draw suites, create a release or render
an example. Network downloads and Game's own bootstrap can take time.

By default its state is under `~/.cache/slayer-bootstrap` (or XDG cache
home), outside the checkout. Override `SLAYER_BOOTSTRAP_DIR` or
`SLAYER_SHELF` with absolute paths. It does not edit personal Game config
or `go.mod`. It prints the environment exports for later Game commands;
use those in your shell to keep selecting the same home, shelf and runner.

Slayer's dependencies are public releases, so setup needs no private input.
A local `replace` in `go.mod` whose directory is missing is refused before
network work.
