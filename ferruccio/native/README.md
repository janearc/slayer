# Ferruccio native frontend

Build and install the app using the [build guide](../../doc/applications/ferruccio/BUILD.md).
For development, `game build ferruccio-binary` from the Slayer checkout
builds the native command and staging helper into `bin/`.
`ferruccio/scripts/patch-gogpu.sh` applies the pinned GoGPU patches before
the build. The native module uses the parent draft and preview packages.

## Run the command

From the Slayer checkout, point the native CLI at the producer and an
existing Azazel queue. Keep the frontend outside Game render lanes so Azazel
can admit its work.

```sh
bin/ferruccio open /path/to/scene.yaml \
  --slayer /path/to/slayer \
  --azazel /path/to/azazel --azazel-queue /path/to/queue \
  --evidence /path/to/existing-output-folder
```

Add `--dry-run` to inspect the launch without opening a window.
`bin/ferruccio open --help` lists flags and
[the queue guide](cmd/ferruccio/PORTABLE-QUEUE.md) describes precedence.
The unpackaged CLI expects an existing worker. App bundles can start one
on demand.

Catalogue `settings.yaml` documents use their own producer. Supply
`--catalogue /path/to/catalogue` for an unpackaged launch. A sibling
`render.json` identifies the recorded specimen. For a standalone plan, use
the specimen chooser. The app build bundles the Slayer still producer and
Azazel client. Catalogue rendering needs a separately supplied executable.

## Native proof

`bin/ferruccio proof --out /path/to/new.png --file /path/to/scene.yaml` opens
a native Metal window, exports its widgets to a PNG and closes its own window.
This is a widget export. To exercise an actual queued still, add `--render`
and the runtime flags above. `--full` goes through the full still action.
`proof --help` lists gesture and capture options. See the
[regression runner](../../doc/applications/ferruccio/REGRESSION.md) for
serial evidence runs.

The [older review bundle](BUILD-PROVENANCE.md) and
[validation notes](VALIDATION.md) retain dated build and test evidence.
