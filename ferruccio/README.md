# Ferruccio

A separate Go application for authoring Slayer scenes. The current frontend is
[GoGPU on macOS Metal](native/README.md). It opens an existing Slayer YAML file,
lets you edit and check it, saves your work, draws a real still preview through
Slayer and displays the resulting PNG beside the source.

Start with [the current notes](native/MORNING.md) and
[the complete native validation](native/VALIDATION.md).
The [Ferruccio/Oakland guide](FERRUCCIO-OAKLAND.md) explains the architecture
and the imported Larson study. [Build provenance](native/BUILD-PROVENANCE.md)
records the exact packaged binaries and external scene resources.

The workspace reads its colours from Chroma, with Vaporwave selected initially.
The footer has theme and type-size controls. Type is adjustable from
16 to 24 points. Venus and Titan imagery is labelled as a credited mission
reference. Actual Slayer results receive a different caption.

The current bundle target is `bin/Ferruccio.app`. Earlier checkpoints remain in
`bin/Atelier.app` and
`bin/Atelier Prototype.app`; it is a separate older application. Nothing is
installed on the host, merged into a production branch or released.

## Use the workspace

Open your YAML with **Open…**, or choose Larson + visitor, Subaru, First picture,
Webb at L2 or Fizz.
Examples open as independent editable copies. Saving an example asks for a
location so your work does not remain only in a temporary folder. **Check**
shows syntax/schema diagnostics while preserving invalid work for repair.

**Render preview** freezes the current draft, then starts the bundled existing
Slayer itself, with no Game lane in between. **Stop** cancels that preview’s owned
process group. The resulting 640 × 360 still appears in the image pane. Logs
and diagnostics remain available; the app never invents a completion percentage.
A result for an older draft cannot replace the current document’s view.

The editor supports multiline Unicode text, selection, clipboard, undo/redo,
cursor navigation and scrolling. There is no completion or language server.

Catalogue previews and supported fixed set cameras accept drag-to-orbit and
scroll-to-zoom. Each gesture edits
the YAML with undo support, then queues the latest view after input settles.
**Observe** keeps the YAML beside the picture; **Compose** gives the picture
the working area and camera buttons. Film planning, movie playback, music,
terrain acquisition and object transform handles remain future work.

Set placement is editable in the YAML.
Existing camera, timing and sound fields
stay in the source. A movie job is refused by the still-preview action.

## Development

Use Game5 build 511afb9 and the Go toolchain in go.mod. Build the native frontend
from `native/` with `CGO_ENABLED=0 game5 build`. The nested module reuses the
parent’s draft, schema and preview packages without importing Fyne or Slayer’s
private source dependencies. GoGPU’s macOS bridge requires Cgo to be disabled.

From the parent directory, `game5 run bundle_native` packages the already-built
native binary into `bin/Ferruccio.app`. Its launcher supplies the explicit
Puzzlemouth verb and bundled renderer locations.

Stage the existing clean Slayer in `bin/engine/slayer`, requested Game5 in
`bin/engine/game5` and processor-limited Game configuration in
`bin/engine/gamehome` before packaging. Catalogue navigation requires the
matching producer in `bin/engine/catalogue`; its pin is recorded in
[the catalogue guide](native/CATALOGUE.md).

The older root module continues to build the Fyne checkpoint independently.
See [native validation](native/VALIDATION.md) for current checks and limits;
[the earlier validation](VALIDATION.md) records the previous iterations.
All saved visual iterations are retained in the shared renders directory.
Reviews remain deferred until the owner has read the work.

The native frontend can expose its process metrics with
`SLAYER_METRICS_ADDR=127.0.0.1:9466` and local Go profiles with
`SLAYER_PPROF_ADDR=127.0.0.1:6066`. Use separate ports for each process;
see [observability](../doc/observability.md) for details.

The workspace
records initialization and UI tick durations, plus preview preparation
and request round-trip duration. The round trip includes queueing and
waiting for the separate renderer, so it is not GPU execution time.

Catalogue specimen settings are supported too. Opening `settings.yaml` beside
its `render.json` selects the recorded specimen; standalone plans have a specimen
chooser. The app uses the bundled catalogue renderer for those files and keeps
the original YAML and previous images intact.
