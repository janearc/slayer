# Reach into the world

Ferruccio opens on an editable native Larson and model maintainer’s two-metre visitor.
The real rendered world takes the window. Move, Turn and Look are three
separate physical intentions, with labelled controls and keyboard access.
There is no required YAML task before the first experiment.

A click selects native geometry; a drag on a selected object in Move or Turn
changes its authored placement. A background drag examines the view. Move
uses a ray intersection with the horizontal plane through the object; Turn
uses horizontal pointer travel in degrees. The temporary guide shows an
intention, never a transformed image presented as computed geometry. Release
creates one undoable source change and queues one real Slayer draw.

Previous
pixels remain explicitly marked while computation is pending. New edits
cancel older requests; source digest and generation guard results. Undo
restores source and renders it again.

Wheel always approaches the view.
Arrow keys move or turn the selected object, or orbit in Look mode.

Construction and exact source are reversible disclosures. The same source
remains authoritative throughout. Native geometry, coordinate systems,
materials and camera rays belong to Slayer; Ferruccio does not build a
second renderer or invent component meshes. Chroma supplies every colour.

This first loop is deliberately bounded to supported space placements.
Surface terrain, object creation, live component isolation, animation and
sound are later work. Unsupported scenes retain their editable source and
existing renderer workflow. Visual approval belongs to maintainer.

Preview transport now belongs to the operator's external Azazel client. The
app needs the configured Azazel executable, queue and appropriate Slayer
backend. It never launches a queue worker or falls back to a direct Game
render. Without queue configuration the native window remains editable and
calmly says that its preview queue is not configured.

Submitting a request is distinct from observing Queued. The footer shows
verified Queued, admission waiting, held reason and Running observations.
These observations do not create pixels or restore picking for edited
source. Previous accepted pixels remain visible with their waiting state; a
first preview may remain blank while the queue holds it.

Even a succeeded
job observation waits for the client's completed output and the
source/generation acceptance guard. No elapsed percentage or queue position
is inferred from these states.

Move, Turn, Look and source/history behavior are unchanged. A placement
press on stale pixels is consumed without a source edit or camera fallback;
choose Look or use the wheel to examine the view while awaiting new pixels.

A new edit cancels this window's outstanding request and supersedes its
observations. Stop cancels that request's context; the client owns
cancellation of its private job. Only the matching generation and its bound
durable job sequence can update the UI. Sequence is job identity, not an
event or position counter; same-job state and held-reason changes remain
observable.

The external worker owns resource admission. See [BUILD.md](BUILD.md) for
configuration.

A cancellation request is not a confirmed stop. The worker returns its last
immutable owner observation with the result. Only the client's Terminal flag
confirms its own process outcome; interrupted state alone grants no cleanup
authority. An unconfirmed cancellation keeps its error and retained input
diagnostic visible while previous pixels remain unchanged. Old-job
diagnostics do not replace newer pending source feedback.

### Switching scenes and accepting a full render

Choosing another scene after a camera or placement edit first offers Save, Discard, or Cancel. The scene menu closes before that prompt opens, so the prompt remains available across UI frames. Discard loads an independent editable copy and clears the old scene's photograph and active gestures.

Render fully freezes the authored film dimensions and sampling. Once its verified result is accepted, the viewport selects Rendered and retains that photograph for idle display; later working-view updates do not replace it with wire geometry. A refused or failed render still reports its diagnostic rather than claiming a completed image.
