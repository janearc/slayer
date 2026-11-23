# Atelier: open, render, view

Open the app bundle. It starts with an editable copy of Slayer's existing
first-picture job, with its original YAML intact.

1. Click **Render preview**. Slayer draws the scene and Atelier displays its PNG.
2. Edit YAML or change a setting, then render again.
3. Use **Open** for an existing job or scene set. Relative input paths keep their
   original meaning. The original source file and its output destinations are
   untouched by previews.
4. **Examples** also opens the existing Webb at L2 set. Some scenes require the
   renderer's external source shelf; failures are shown in the status and Output.
5. The text-size selector offers 16, 18, 20 and 24. It changes the widgets and
   editor, without changing the document. Vaporwave colour provenance is in colour.md.
6. Stop cancels the current request. Closing waits for the owned renderer to stop.

The preview is a still at 640 by 360 with one ray per side, using the document's
renderer selection. It starts Slayer itself, with no Game lane, with CPU use held
to two processors unless Settings choose otherwise. The activity indicator is indeterminate; Output shows what
Game and Slayer actually report. A changed document cannot accept an older result.

Previews snapshot the current draft to a new temporary file beside its source.
That preserves relative input paths without rewriting them. Job output fields
are omitted only in that temporary snapshot; the preview image is written to a
new directory. The temporary input is removed when the process ends. A source
directory that cannot accept a temporary file is refused with the filesystem error.

The real files shipped as examples are unchanged copies of Slayer's
doc/examples/jobs/first-picture.yaml and sets/webb-at-l2.yaml. Example editing uses a separate
copy, so updating the app cannot overwrite a saved example draft. Use Save and
choose a durable location to keep it beyond temporary-file cleanup.

Camera movement, movie rendering/playback, shot planning, music editing, terrain
acquisition and scene placement are still future work. Film jobs are refused by
this still-preview action. References to Venus and Titan are labelled mission
images; choosing them while a real file is open leaves that file unchanged.

The native smoke has exercised opening first-picture.yaml, the actual Render
action, Slayer execution and displaying its output before capturing the window.
Tests cover source preservation, output redirection, schemas, refusal cases,
widget state, palette, text sizing and command dry runs. This remains development
work on its own road branch; no release or independent acceptance is claimed.
