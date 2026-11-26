# Loading the staged working view

A staged view must describe the current editor text, not a file last saved.
`stageWorking` admits a set or job buffer relative to its original path,
opens Slayer's session without rendering, and converts its first staged frame.
It keeps the session alive while the view's hit targets reference its shapes.
The aggregate world cannot be selected for authored edits. It returns proxy
and omitted-resource notices alongside the view; callers must show them.

Loading is costly and may not stop immediately after cancellation. The UI
must perform it outside the UI thread with one active build, coalesce newer
requests, reject results for older text or paths, and close every discarded
or replaced session. Closing the window cancels the request. Never claim a
scene is current because an old worker finished. These lifecycle rules and
focus/selection behavior must be tested before connecting this adapter.

The session currently reads cache and shelf locations from process-wide
environment variables. The live bundle supplies explicit runtime settings;
the adapter must receive and honor them without changing the process
environment or silently selecting a different cache before it is connected.

The staged session brings in Slayer's Metal code. The in-process prototype is
restricted to Cgo builds. The normal pure-Go frontend uses the separate Cgo
helper as a fallback after its existing display refuses a valid set or job.
Supported authored things can be selected and edited; generated groups and
proxies stay read-only, and the notice remains visible.

A small unsaved scene
has passed a headless workspace-to-helper check; the new fallback has
not yet been checked in a native window. No broad app compatibility is claimed.

For the child-process path, a workspace must capture the current path, text,
kind and resolved settings on the UI thread, then ask the coalescing loader to
start the helper. While it runs, no old result is called current. A change to
the document cancels the obsolete request, including when the replacement can
use the existing display. Completion wakes the UI and is accepted only for the
same path and text.

The staged view's notice stays visible. A failed helper
leaves a specific refusal, never an empty scene. Window close cancels and joins
the helper before quitting; tests cover both normal and delayed cancellation.

Each helper has its own temporary cache under the configured cache, which is
removed after the child exits. The cache does not benefit from shared entries;
there is no fixed resident-memory watchdog. Futian's shared geometry passed
through the helper and produced a headless Metal frame with omitted resources
reported. Its native-window behavior and responsiveness remain unverified.
