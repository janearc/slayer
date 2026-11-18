# Ferruccio preview client

Ferruccio remains an interactive native GoGPU application. It edits Slayer
scene source and displays completed producer images. The separate operator
Azazel executable admits and supervises finite preview jobs. The GUI never
starts a queue worker or invokes the renderer directly.

The parent Ferruccio module owns snapshot preparation, the external control
protocol and measured output receipts. The native module owns document
history, selection, camera and placement gestures, UI state and rendering of
the completed picture. Slayer owns scene geometry, materials and light
transport. Chroma owns the presentation palette. All three modules live in
this Slayer repository.

Each edit freezes an adjacent input so relative references still resolve,
a retained audit copy and a unique output folder. A persisted request and
key bind a retry to the same content, executable route and shared queue.
Azazel submit acceptance means queued work.

Status distinguishes queued,
claimed but waiting for admission, executing, and a verified terminal result.
The durable sequence identifies the job; repeated observations of that same
sequence carry progress. It is not an event counter.

Worker callbacks enqueue immutable generation-tagged observations. Only the
native UI thread updates the visible status. The previous picture stays
visible and labelled while work waits. An earlier generation cannot replace
the current picture, selection geometry or source. A successful queue result
still needs a readable PNG of the frozen dimensions before image acceptance.

Native proof gestures require that current accepted result, so an earlier
picture cannot satisfy a new proof.

Stop and superseding edits cancel only the acknowledged job submitted by
this plan. Cancellation uses a fresh bounded control context after the
render context ends. The client confirms the same job's terminal result.

Uncertain acknowledgement, malformed or unavailable status, interrupted
recovery, and unconfirmed cancellation retain the frozen input. Closing a
window or finishing the client cancellation attempt does not itself prove
worker quiescence. A proof timeout requests cancellation from the UI thread
and waits for the client's cleanup before quitting.

Known direct file references are declared as queue inputs. This does not
claim discovery or freezing of the complete transitive dependency closure.
Current native playground geometry is inline. Producer execution and full
source interpretation remain Slayer's responsibility.

The GUI holds no Game render lane while waiting for the external worker.
Run the GUI and queue proof unwrapped; the worker owns resource admission.
Builds and checks still use Game5. There is no direct Game rendering fallback.
Azazel owns live render pages and progress publication; the client does not
introduce another gallery writer.
