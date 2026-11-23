# External Azazel preview transport

The app prepares one adjacent frozen YAML and one unique output directory per
intended render. Azazel alone schedules and supervises producer commands.
Submission uses configured Azazel/queue/agent, an absolute existing producer
binary and a finite request checked against the owner's published schema.
No direct producer, Game-lane wrapper, worker, resume or private queue fallback
exists in the preview transport.

The plan identity determines an idempotency key. The exact serialized request
is retained beside an audit YAML copy; uncertain acknowledgement retries use
the same key and content. Acceptance is queued work, never image completion.
RunObserved polls the retained ledger until its verified agent/key/sequence is
terminal. A claimed job remains waiting until Executing is true. Only succeeded permits PNG decoding, dimension verification and a
receipt. Queued/running updates include admission or queue pause reasons.

Cancellation names only this plan's submitting agent and sequence. It uses a
fresh bounded context after the user's render context ends, then checks for a
terminal ledger result. Unknown acknowledgement, malformed ledger, failed
status observation or unconfirmed cancellation retain the adjacent input.

Plan.Close removes input only when no submission could reference it, or a
verified succeeded, failed or cancelled result proves the queue no longer needs it.
Operator-interrupted or uncertain results retain input because descendant
quiescence is not independently established by the current owner recovery path. Audit files and
output directories remain. Terminal failure/cancellation never accepts PNGs.

The client runs no queue engine and starts no publishers. Azazel owns dispatch,
partial images, live pages and progress. Tests use generated Runner mocks to
assert actual subprocess arguments and ownership; no real queue is submitted.


The request declares the adjacent snapshot, retained audit YAML, and known
direct job scene.set or scene.world.imports file references. This does not
claim discovery or freezing of every transitive asset/data dependency.
Current native playground geometry is inline. The renderer remains the
authority for its complete source closure.

Retry routing is frozen to the original Azazel executable and shared queue.
Own failures include bounded result text and the owner's documented attempt-log
path when an attempt number is present. Full ledger JSON stays inside the
client and is never copied to the UI diagnostic stream.

Update.Sequence is the durable job ID, not a progress counter. Repeated
observations of the same ID carry state and held-reason changes. Terminal is
an explicit cleanup-authority flag derived from verified process state;
interrupted keeps Terminal false. Retry acknowledgements expose their actual
current owner state, including waiting or terminal states, while image
acceptance still waits for the matching ledger outcome and PNG verification.


Owner clarification, 2026-11-15 at 23:52:22 UTC: Executing is a retained
execution-claim marker, not a current-liveness signal. Matching terminal state
and result are authoritative only after the owner has observed command exit
and process-group quiescence. Safe terminal authority therefore ignores the
historical Executing flag and still rejects Uncertain results. Interrupted
remains conservative because recovery attestation has its separate limitation.

Running with Executing false still reports waiting. Historical process-group
metadata is never signalled by this client.

Output directory prefixes are optional validated basenames; the native caller
supplies dated owner labels under its configured shared destination. Before
submission, recipe.txt records exact argv and requested settings, source hash,
audit/request/acknowledgement links and known limitations.

It claims neither active rendering nor an estimated completion time. Receipts
record the verified producer command and Azazel; worker executable/Game
fingerprints remain linked through the retained owner acknowledgement, never
inferred from client config.

## Explicit shared-Slayer transport

The default remains Azazel. An operator-authorized caller can explicitly select
Tools.Transport=slayer and supply the shared Slayer path and a cache folder.
This route starts the existing foreground Slayer draw command itself, with no
Game lane: the person at the keyboard does not queue behind batch work. It is
never an automatic queue fallback.
Catalogue settings are refused in this route.

Direct execution creates its own process group, terminates that group on
cancellation, and confirms group disappearance before authorizing cleanup.
An unconfirmed stop retains the input. Direct updates contain only a terminal
outcome, Sequence zero, and confirmed cleanup authority when quiescent.

Measured receipts record the producer and Game actually invoked; they contain
no fabricated queue identity. Runtime inputs and outputs use the caller's
shared render destination, with the frozen YAML and exact invocation retained.
