# Atelier on GoGPU

A separate native GoGPU/Metal frontend reuses the existing draft and Slayer
preview packages from its parent module. The older Fyne prototype remains a
working checkpoint while this port is verified. No engine or scene file changes.

The first workspace opens existing YAML, edits it without dropping comments,
validates with the pinned schema, saves atomically and renders a frozen still
through Game. Rendering stays asynchronous, cancellable and bounded. Worker
results return to the UI thread; stale results cannot replace a newer document.

A small multiline YAML widget supports text entry, cursor movement, selection,
clipboard and undo. It is a basic text editor, not an IDE. Typography is adjustable.
Vaporwave colours come from the same source used for the current prototype.
Buttons have visible outlines and a single pink primary action.

Video, camera planning and music remain reserved; a still preview refuses movies.
Real imagery is labelled as either a mission reference or an actual Slayer output.
Every exported view is labelled with how its pixels were captured.

Catalogue settings are a separate document format, validated with the catalogue
plan schema pinned to the bundled producer. They go to `catalogue render`, not
`slayer draw`. A specimen is selected explicitly or read from the adjacent
`render.json` receipt; filenames never choose a specimen.

A standalone plan
without that context asks for a specimen. Preview freezes the first detail in
the plan at 640 by 360 and one ray per side, retaining lighting and camera values.

The original settings and previous images remain untouched. Output goes into a
fresh catalogue book, and the selected specimen's actual PNG is displayed.
