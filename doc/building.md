# Building Slayer

Slayer builds through **Game**, the project runner used across these
repositories. Game reads the committed `.game5/` configuration for source requirements,
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

Game runs the work; Slayer renders and encodes movies. The `.game5/`
configuration is readable text: inspect it to see what each target actually does.

## Tools and inputs

| Requirement | Purpose |
|---|---|
| Git | checkout and exact source revision |
| Go version in `go.mod` | this checkout requests Go 1.27.4 |
| compatible Game v0.5.0 or newer | reads this checkout's `.game5/` configuration |
| macOS, Xcode command-line tools, usable Metal device | Metal rendering and GPU tests |
| macOS with cgo | Slayer's AVFoundation movie encoding and audio decoding |
| source shelf | normal pre-build verification |

Some separate experimental tools use `ffmpeg`; their own guides describe
that requirement.

CPU-only rendering does not require Metal. Metal kernels compile at runtime;
a successful Go build does not establish GPU conformance. Native mesh
traversal needs the supported Metal ray-tracing API too.

Install [Game](https://github.com/janearc/game) with Go:

```sh
go install github.com/janearc/game/cmd/game@latest
export PATH="$(go env GOPATH)/bin:$PATH"
```

If `GOBIN` is set, add that directory to `PATH` instead. The current Game
release may fail to install because it references an unavailable
`libtheme-css` revision; check for an updated Game release if Go reports
that error. Some development installations name the executable `game5`;
substitute that name in these examples.

```sh
command -v game
game --age
go version
```

Use a current Game with argument forwarding. In `game run NAME -- ARGS`,
the words after `--` reach the application. Older builds discarded them.

## Configure the cache and source shelf

Rendering requires an absolute cache directory for reusable data. The
source shelf holds the papers and datasets pinned in `sources.yaml`. Name
folders of your own outside the Git checkout. Game and commands you start
directly inherit these shell exports.

```sh
export SLAYER_CACHE="$HOME/slayer/cache"
export SLAYER_SHELF="$HOME/slayer/sources"
mkdir -p "$SLAYER_CACHE"
```

Put the exports in your shell's startup file so later commands find these
folders. The shelf is separate from renders and Grissom survey data.

You can name both folders in your Game config instead of the shell.

```
set SLAYER_SHELF /absolute/path/to/slayer/sources
set SLAYER_CACHE /absolute/path/to/slayer/cache
```

A Game `set` value takes precedence over the shell for commands Game
starts. For a command you start directly, such as `slayer draw`, export
the path in your shell.

With `SLAYER_SHELF` unset, verify, check and path read `shelf/` in the
checkout, which only a working tree of the road has.

## Fetch the build citations

Before compiling, Game verifies the sources cited by the numbers in
`research/research-derived-data.yml`. Those 61 files total about 116 MB
in this checkout. Install Poppler so `pdftotext` is on your path.

From the Slayer checkout, set the shell variable below to the same shelf
path as in your Game configuration. Copy the excerpt shipped in the
checkout, then fetch the rest of the citations:

```sh
export SLAYER_SHELF="$HOME/slayer/sources"
mkdir -p "$SLAYER_SHELF"
cp -n shelf/prima-ipac-observatory "$SLAYER_SHELF/"
game run sources-fetch -- --research
game run sources-verify
```

Fetch keeps files that match their recorded SHA-256 hashes. If a publisher
changes a file, fetch refuses the new bytes. Use the original pinned copy
from an existing shelf. Verify reads the local files and checks the
quoted numbers. It prints nothing on success.

Run `game run sources-fetch -- KEY` to fetch specific datasets as a scene
needs them. Without arguments, `sources-fetch` attempts the entire source
list, which records about 3 GB of source files. The Kilauea elevation model
alone is a 1.82 GB archive and takes another 1.93 GB once unpacked.
`sources-check` checks hashes for the entire list, including files used
only by particular scenes or research.

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

A named target such as `game build azazel-binary` executes the custom
recipe in `.game5/`. Its raw `go build` does not automatically receive the
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
in an outer GPU lane: it obtains lanes itself. For rendering on a shared
machine, submit work through [Azazel](azazel.md).

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
| unknown Game configuration version or `gomaxprocs` key | selected Game executable and `--age` |
| source verification fails | shelf path, pinned bytes and source fetch |
| missing dependency symbol | revision, `go.mod` and `GOWORK` |
| Metal unavailable at render time | Mac tools, device or a supported CPU alternative |
| movie encoding fails | macOS, cgo and the requested movie dimensions and rate |
| flags ignored | current Game and `--` before application arguments |

Keep the exact error and revision with a report. Repair the setup rather
than disabling source verification or deleting useful dependency caches.
