# Ferruccio render requests

**Preview** and **Render fully** freeze the current draft and submit a
Slayer request to Azazel. A render folder retains the audit YAML, request,
producer receipt and accepted image. The frozen input preserves the original
context for relative paths. External resources can still be needed.

The app displays the acknowledged job's queue state and row progress when
the worker reports it. Previous pixels stay visible until successful
completion and PNG validation. A result from an earlier document or draft
cannot replace the current view.

**Stop** requests cancellation of the app's job. If the queue cannot confirm
the outcome, Ferruccio reports it as unconfirmed and retains the frozen input.
A temporary failure to read status leaves the job in place and the client
tries again. The [queue setup](howto-run-dis.md#render-your-scene) covers the
packaged app. The [client design](../../../ferruccio/internal/preview/QUEUE-DESIGN.md)
describes request identity, retries and cleanup.
