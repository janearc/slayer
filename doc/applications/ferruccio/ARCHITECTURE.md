# Ferruccio inside Slayer

Ferruccio is Slayer’s separate native authoring executable. It is built from
this checkout, and its application bundle contains this checkout’s Slayer
and catalogue executables. The main native command is
`ferruccio/native/cmd/ferruccio`; root Game build/launch/package verbs are
specified in BUILD.md. No private renderer road or checkpoint is required.

The root Go module owns sensors, native geometry, materials, manufacture,
scene staging and rendering. `session/placement` is the pure Go canonical
space-placement primitive: renderer and picking share seed mixing and
transforms. Native `stage.Add` supplies world surfaces for pick occlusion.
`film.Look` generates selection rays. No Ferruccio rasterizer exists.

The nested `github.com/janearc/slayer/ferruccio` module owns source
documents, producer contracts, exact embedded recipes and renderer process
adaptation. `internal/draft` retains YAML as the authority, checks pinned
producer schemas and applies atomic camera/placement changes without
discarding comments.

Placement identities are source paths, independent of
position. `internal/preview` freezes a request beside the source for
relative inputs, submits through the configured operator Azazel client,
observes that private job and verifies completed output. Optional evidence
destinations receive outputs directly. It does not start a worker, use a
private queue or bypass queue ownership with a direct Game render.

The native module uses GoGPU for its macOS Metal window and GoGPU UI for
layout, event dispatch, focus, capture, clipboard and controls. Its custom
image leaf paints actual completed Slayer pixels. `internal/editor` owns
bounded text, scrolling, selection and undo; the same tested editor supports
read-only producer recipes. `internal/picking` prepares immutable native
surface geometry for an accepted source and image aspect.

`workspace` owns state on the UI thread. A worker runs the frozen preview
and constructs picking geometry off that thread. Its RunObserved callback
sends immutable own-job state, held reason and sequence values through a
bounded channel. Tick applies these observations on the UI thread only for
the current generation and the same bound durable job sequence. Sequence is
job identity, not an event counter.

Same-job state and held-reason changes
are applied in callback order; alien job identities are refused. Queued and
Running are queue observations, not image results. Previous pixels and
source selection identity remain retained until actual completed output
passes the existing acceptance guard.

Results carry generation and source identity; stale work cannot replace
current pixels or selection. ProofOutput additionally requires the exact
accepted generation, path and text. Normal UI Output retains the previous
accepted image while a newer request waits. Missing queue configuration
keeps the source editor open with a calm status, and automatic startup does
not display a configuration-error modal.

Cancellation is scoped to this
window's context and the client's own private job. Pointer capture retains a
drag outside the initial hit region. Release applies one source edit through
the editor’s history and queues one draw. Wheel and keyboard share the
camera/pose paths. Undo restores source and renders it, rather than moving
or reverting a bitmap.

The visual language starts with the world, then a local invitation to grab
what the pointer actually hits. Move, Turn and Look express separate
intentions. Exact units and computation state remain visible. Construction
and source are temporary disclosures; Escape returns to the same world.

Chroma supplies palette tokens, including checked readable pairs. Type size
and colourway are configurable. Colour, labels and icons do not stand in for
working capabilities. Immediate guides are labelled intentions; new pixels
only appear after Slayer finishes.

The first playable scene uses native mcrn-larson-landed and human-scale-2m.
The older Oakland glTF mannequin is retained as an import test fixture, not
the user-facing scale figure. Video, camera planning, music, terrain
editing, object creation and component isolation remain future extensions.

Existing source workflows remain available for scenes outside first-play
picking. The earlier Fyne frontend is preserved history, not the active app
entry.

The external worker owns resource admission; the interactive frontend holds
no client-side Game lane while awaiting it.
