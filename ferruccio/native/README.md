# Render routing update, 15 November 2026

The clean ccfe456f first-play bundle and its real native proofs are
preserved as the historical direct Game adapter. The external Azazel submit,
observe and cancellation adapter is now implemented and covered by client
and native UI tests. A real scheduler-backed native proof remains pending
operator queue readiness and authorisation. No new render followed the hold.

# Ferruccio on GoGPU

This is the native GoGPU/Metal frontend for Ferruccio, previously Atelier.
It reuses the existing schema-aware draft and real Slayer preview packages
in the parent module. The Fyne prototype remains available as a checkpoint
in the parent directory.

The default Play composition opens the prepared native Larson world and
requests a bounded 640 × 400 preview through the operator Azazel queue.
Move, Turn and Look change the same authored scene; source and construction
are available on request. Observe and Compose retain the existing YAML and
preview workflow.

The external Azazel worker owns render admission and execution. The native
client requires its configured executable and queue path.

The render preserves relative file references, removes job output
destinations from the temporary snapshot, and keeps the original YAML
unchanged. Stop submits cancellation of this preview job; its client does
not own the renderer process group. The UI distinguishes confirmed terminal
outcomes from unconfirmed cancellation and preserves the own-job error and
retained-input diagnostic.

The editor supports multiline Unicode text, selection, clipboard, undo/redo,
keyboard navigation and scrolling. Choose 16, 18, 20 or 24 point type in its
pulldown. It is a basic editor; completion and language-server features are
not connected.

Catalogue previews and simple fixed set cameras support drag-to-orbit,
scroll-to-zoom, arrow keys and +/−; camera edits are undoable YAML changes.
Set gestures require explicit numeric `location.at` and
`orientation.look_at` coordinate arrays on the first camera. Moving,
relative and rigged cameras remain editable as YAML but are outside the
gesture controls. Video planning and music remain future work.

The optional Observe composition shows the source beside the picture.
Compose expands the picture and provides camera buttons; Edit YAML returns
to Observe. Theme and text size controls remain available in both layouts.
Colours come from Chroma's bundled CSS sources, with source pins and role
choices in [theme provenance](internal/appearance/themes/PROVENANCE.md).

From the Slayer root, `game5 run ferruccio-build-all` builds the current
backends and native frontend. `game5 run ferruccio` launches them together.
The native child build uses `CGO_ENABLED=0` and GoGPU's pure-Go macOS
bridge. The parent Ferruccio module retains the earlier Fyne prototype as a
checkpoint.

The public interface uses Puzzlemouth:

From this native module directory, an explicit launch is:

```sh
bin/ferruccio open --examples ../examples \
  --azazel "$AZAZEL" --azazel-queue "$AZAZEL_QUEUE" --agent ferruccio \
  --slayer ../../bin/slayer --catalogue ../../bin/catalogue \
  --evidence "$HOME/mesh/the-vorrh/renders/frames"
bin/ferruccio open /path/to/existing.yaml \
  --azazel "$AZAZEL" --azazel-queue "$AZAZEL_QUEUE" \
  --slayer /path/to/slayer --catalogue /path/to/catalogue \
  --evidence "$HOME/mesh/the-vorrh/renders/frames"
```

The `proof` command opens the actual native Metal window, optionally renders
an existing YAML file through the same visible action, exports that window’s
widget tree, and closes only its own window. Its PNG is labelled as an
offscreen widget export, rather than an OS screenshot.

Run the new queue proof unwrapped, with its shared evidence destination:
`bin/ferruccio proof ... --azazel PATH --azazel-queue PATH --evidence
$SLAYER_RENDERS/frames`. The separate Azazel worker owns admission;
frontend-held Game lanes can block its work.

The earlier ccfe456f lane wrapper is historical evidence. See [the integrated
build guide](../../doc/applications/ferruccio/BUILD.md). `--gesture orbit`, `zoom`, `move`,
`turn` or `undo` dispatches through the actual UI event path after the initial
render. Each needs `--render --file FILE`.

The undo proof uses movement in Play and orbit in Observe, then compares the
restored render's pixels with the initial render. Compose does not support undo
proof. `--composition` and `--theme` choose the same settings as the UI.

Pinned dependencies: gogpu 0.54.1, ui 0.1.55, gg 0.52.7. The separate
upstream editor repository is currently scaffolding, so the YAML widget is
local. Upstream Image currently draws a placeholder; Ferruccio’s viewer
draws actual PNG/JPEG pixels.

gogpu carries patches in `patches/`: on macOS the app can start in the
background without taking focus, and Command shortcuts do not insert their
letters into the focused editor. Option text remains available.
`ferruccio/scripts/patch-gogpu.sh` applies them to gogpu 0.54.1 from the
module cache before a build; the background patch is offered upstream.

Catalogue `settings.yaml` files use their own schema and renderer. Open the
file beside its `render.json` receipt to recover the specimen automatically.

For a standalone plan, choose the specimen above the editor. Render preview
freezes the first listed detail at 640 × 360 and one ray per side; camera
and lights stay as authored. It writes a new catalogue book and displays its
specimen PNG. Original settings, receipts and earlier pictures are not
replaced.

The app bundle includes the matching catalogue executable. A command-line
launch can specify it with `--catalogue PATH`, beside `--slayer` and
`--game`.

Every native preview requires an explicit resolved shared evidence
destination. The command uses --evidence, then FERRUCCIO_EVIDENCE, then the
configured shared renders/frames default. Without a destination, the window
stays editable and reports that its shared render destination is not
configured.

Native requests use UTC YYYY-MM-DD-<normalized-agent>-ferruccio-preview-
folder prefixes; the agent portion is a bounded safe basename. No native
preview silently falls back to an unlisted temporary output folder.
