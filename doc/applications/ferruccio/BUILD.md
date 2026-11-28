# Build Ferruccio

From the Slayer checkout on Apple silicon, using the Go version in
`go.mod` and [the project's Game setup](../../developing.md):

```sh
game build ferruccio
game build ferruccio-dmg
```

The app target builds `bin/Ferruccio.app`. The disk image target rebuilds
that app and puts it in `bin/Ferruccio.dmg` beside an Applications shortcut,
the quick-start PDF and installation notes. It writes the image digest to
`bin/Ferruccio.dmg.sha256`. A new build replaces an app marked as built by
this target or a disk image whose recorded digest still matches.

The bundle includes the native frontend, staging helper, Slayer, Azazel, the
Game version pinned in `go.mod`, and example files. The GoGPU frontend builds
without Cgo. The app declares macOS 13.0 and the build sets the same deployment
target for Cgo. The linked minimum does not establish runtime compatibility.
[Metal native ray tracing](../../renderer.md#backends) has its own requirements.

The launcher selects Azazel and the bundled Slayer. See the
[queue defaults](../../../ferruccio/native/cmd/ferruccio/PORTABLE-QUEUE.md)
and [installation instructions](howto-run-dis.md) for runtime settings.
Some scenes need external assets and source data.

## Checks

```sh
game run ferruccio-check
game run ferruccio-check-native
game run ferruccio-check-launcher
```

The parent suite covers the draft, producer contracts and render client.
The native suite covers the editor, working view and launch paths. The
launcher run selects only the launcher tests.

From a clean checkout, build the image and check it with
`sh ferruccio/scripts/check-dmg.sh bin/Ferruccio.dmg MOUNTPOINT`. Supply an
unused mountpoint. The checker mounts the image read-only, verifies the image
digest and bundle checksums, checks the app, helper, Slayer and Azazel for
the clean checkout revision, and verifies the Game pin and arm64 macOS
deployment metadata. It also compares the packaged guide and Oakland study
with their source files.

The manual [GitHub Actions workflow](../../../.github/workflows/ferruccio-dmg.yml)
builds and checks an image and uploads it as an Actions artifact.
