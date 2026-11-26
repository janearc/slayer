# Building Ferruccio

From the Slayer checkout, `game build ferruccio` builds the frontend and
staging helper, Slayer and the Azazel client, then assembles
`bin/Ferruccio.app`. `game build ferruccio-dmg` performs that build and makes
`bin/Ferruccio.dmg`, containing the app and an Applications shortcut. Builds
replace only a previous app made by this target or a disk image whose recorded
checksum still matches.

The app and its engine executables are bundled; a
shared Azazel queue and any external scene assets still need configuration.
No rendering is started by building. The app launcher honours the existing
Game environment and does not install or start an Azazel worker.

The descriptions below record older build and package paths and may describe
historical launch defaults. The two commands above are the new entry points.

## populated opening candidate

The native app defaults to `--preview-transport slayer`. It displays the
supplied starter before any fresh preview and uses the external shared
Slayer through Game for finite CPU work. The old Azazel transport remains
available explicitly; the opening launcher has no retired queue dependency.
Catalogue rendering is left unconfigured until a shared producer exists.

```sh
export FERRUCCIO_SLAYER=/absolute/path/to/slayer
export FERRUCCIO_STARTER=/absolute/prepared-starter
export FERRUCCIO_BUNDLE=bin/Ferruccio-Opening.app
game5 run ferruccio-build
game5 run ferruccio-package
```

The starter directory must contain `source.yaml`, `preview.png` and
`receipt.json`. Packaging copies these into `Contents/Resources/starter`
and passes `--starter` explicitly. It captures the shared renderer path;
it does not copy a checkout Slayer or catalogue binary. The candidate
uses a separate bundle identity and refuses to overwrite an existing
output, including the running user app. Deployment is a separate decision.

## source and tools

Ferruccio submits preview requests through the configured operator Azazel
client.

The active frontend is `ferruccio/native/cmd/ferruccio`, in module
`github.com/janearc/slayer/ferruccio/native`. Its parent draft and preview
module is `github.com/janearc/slayer/ferruccio`. Local replacements point
to the parent with `..` and Slayer with `../..`. The Fyne commands in
`ferruccio/cmd/` remain historical checkpoints.

Configure the existing operator executable and queue; the application
creates neither a private worker nor an alternative queue:

```sh
export AZAZEL=/absolute/operator/azazel
export AZAZEL_QUEUE=/absolute/operator/queue
export FERRUCCIO_AGENT=ferruccio
export FERRUCCIO_EVIDENCE=/absolute/evidence/folder
```

The public Puzzlemouth flags are `--azazel`, `--azazel-queue`, `--agent`
and `--evidence`. The default agent is `ferruccio`; an explicit identity
is retained unchanged for owned cancellation. Azazel, queue and evidence
paths become absolute before document-relative work. Game flags remain
available for build metadata; the queue client does not require Game and
cannot fall back to direct Game rendering.

The direct CLI evidence precedence is `--evidence`, then
`FERRUCCIO_EVIDENCE`, then `mesh/the-vorrh/renders/frames` under the current
user home.

Launchers pass their environment-selected destination
explicitly. A missing home and absent explicit setting leave rendering
unconfigured. A destination must be an existing parent. The UI retains
editing when configuration is missing; it does not fall back to temporary
preview output. Each request creates a fresh labelled child directory.

## integrated build

Run from the Slayer root with a scoped `GAME_HOME` naming the shared
Slayer road and source shelf. Set `GAME_ROAD` when the config needs it.
`GAME_BIN` selects Game5; scripts otherwise resolve `game5` from `PATH`.

Use Go 1.27.3 and make the pinned Chroma module available in the builder
cache or configured proxy. gogpu carries one patch, applied by
`ferruccio/scripts/patch-gogpu.sh` before the build; see
`ferruccio/native/README.md`.

```sh
export GOTOOLCHAIN=go1.27.3 GOMAXPROCS=2 GOWORK=off
export GOFLAGS='-p=1 -parallel=1'
game5 run ferruccio-build-all
game5 run ferruccio -- --dry-run
```

The wrapper finishes the root Game build with `CGO_ENABLED=1`, including
source verification and root commands, then runs Game in the native
module with `CGO_ENABLED=0`. It copies the stamped native executable to
root `bin/ferruccio`. GoGPU uses its pure-Go macOS bridge. The frontend-only
entry point is `game5 run ferruccio-build`; its explicit target is
`game5 build ferruccio-binary`.

Plain root `game5 build` discovers root `cmd/*`, not nested modules. The
current Game implementation runs pre-build before `buildBinaries` can
perform sticky synchronization. Building a nested frontend in pre-build
could therefore stamp earlier sources before a root synchronization.
The sequential wrapper avoids that ordering problem without a recursive
pre-build or a Game fork. No pairing command is required for these builds.

A normal `game5 run ferruccio` opens the native window and requests its
configured queue preview. The dry-run above opens no window, starts no preview and does not verify
operator queue readiness.

## package and provenance

After the integrated build, `game5 run ferruccio-package` packages current
root `bin/ferruccio`, `bin/slayer`, `bin/catalogue` and the selected Game.
`game5 build ferruccio-package` is the corresponding explicit target and
also runs source verification. Missing executable or operator settings
are refused before the bundle is created.

The result is `bin/Ferruccio.app`, identifier
`com.janearc.slayer.ferruccio`, with `Contents/MacOS/launch`. Packaging
captures the operator and evidence defaults as safely quoted launcher
values; runtime environment settings can override them.

It does not copy
an Azazel executable into the app. The launcher passes public CLI flags.
The bundle script quotes each value for POSIX sh while building;
the generated POSIX shell launcher needs neither Ruby nor Python.

`Contents/Resources/builds.txt` records actual executable `--age` output;
`SHA256SUMS` records the packaged hashes. Packaging preserves stamps and
does not turn a dirty build into release provenance. Matching package
inputs, launcher syntax, plist and dry-runs are separate from queue
runtime acceptance.

## checks

```sh
game5 run ferruccio-check-command
game5 run ferruccio-check-contracts
game5 run ferruccio-check-preview
game5 run ferruccio-check-native
game5 run ferruccio-check-launcher
```

The launcher check requires a built native CLI and exercises hostile
quotes, spaces, dollar signs, backticks and newlines through real CLI
dry-runs. It checks captured defaults, environment overrides and the
shared evidence destination for both launchers. It starts no queue client.

The contract check compares embedded schemas and the catalogue menu to
their current Slayer producers; recopy changed contracts and update
`ferruccio/internal/assets/SHA256SUMS` before building.

## running with a queue

Run the frontend unwrapped. The external worker owns resource admission and
does not inherit frontend-held `GAME_LANES`; wrapping the frontend in Game
render lanes can block worker admission.
