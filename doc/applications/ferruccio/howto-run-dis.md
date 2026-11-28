# Install Ferruccio

Ferruccio targets Apple silicon Macs and declares macOS 13.0 as its minimum.
Some Slayer render modes have additional hardware and macOS requirements.

1. Open `Ferruccio.dmg`.
2. Drag **Ferruccio** onto **Applications**.
3. Open **Ferruccio** from **Applications**.

If macOS blocks a copy you trust, open **System Settings** and check
**Privacy & Security** for **Open Anyway**.

## Open a scene

Ferruccio opens the **Larson playground** scene. Use the **Examples** menu
to choose **Oakland Larson study**, or **Open…** to choose your own Slayer
set YAML. Use **Save** to keep your changes. The disk image's
`Ferruccio Quick Start.pdf` walks through the first edit and render.

## Render your scene

Choose **Preview** or **Render fully** when you are ready for a Slayer image.
The app reuses a configured or discovered Azazel queue. If neither is
available, the installed app initializes a private queue on first render
and starts Azazel dispatch as needed. You can also choose an existing queue
folder containing `queue.db` under **Settings** → **Render queue**.

Choose **Show in Finder** below the image to select the finished PNG. Its
`receipt.json` is in the same folder. **Settings** → **Render output** shows
where new render folders go. The per-user fallback is
`~/Library/Application Support/Ferruccio/Renders`.

## Open a scene that uses external data

If the working view cannot stage, open **View details…** to see the
producer's error, including the dataset and path it looked for when a
pinned file is missing. In **Settings…**, choose **Source shelf** for
datasets pinned in `sources.yaml`, or **Scene data** for scene resources
such as the `beachsurvey` folder. Settings shows the active value and
where it came from. A command-line flag overrides the environment, and
the environment overrides the saved setting.

If you need to populate a shelf, use the original scene's data-provider
instructions and keep its manifest and pins. A scene copied away from
its original tree needs the corresponding `sources.yaml` beside it or in
an ancestor folder. Once the required files are available, choose
**View details…** → **Retry** to stage the same draft again.
If scene detail fails, Ferruccio keeps the previous view and retries the
same detail after you change the source shelf or scene data folder.

## Verify the download

If you received `Ferruccio.dmg.sha256` beside the image, compare its digest
with the first field printed by `shasum`:

```sh
shasum -a 256 Ferruccio.dmg
cat Ferruccio.dmg.sha256
```
