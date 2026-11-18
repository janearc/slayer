# Shared job queue

Azazel stores a request around an existing Slayer command. Arguments stay
as an array, without a shell or a second scene schema. A request identifies
its agent and idempotency key, executable, working directory, environment,
input files and timeout. Submit fingerprints the executable
and declared inputs; execution checks those identities again.

One queue serves cooperating agents of the same operating-system user.

A bbolt transaction assigns the next sequence and changes job state. A
permanent worker lock admits one main dispatcher. Each resource class selects
its oldest eligible request. GPU work shares one actual Game lane under a
batch coordinator; CPU work enters its own Game lanes alongside it.
Unknown attempts retain their ownership fence rather than being replayed.
Different queues do not establish a common budget: use one shared queue.

The dispatcher enters the requested Game lane before executing the command.
Every command remains a foreground process group. Cancellation, timeout,
unknown monitoring or elevated system memory pressure stop that group.
Pressure and low-disk stops persist and require explicit resume. RSS is
measured, not capped; no small Go heap is imposed. The two-CPU limit remains.

Programs must not daemonize or escape their group; interactive editors
belong at the terminal rather than in an unattended batch.

An unsuccessful process is a stored failure, never an automatic retry.

Restart preserves queued requests, adopts surviving supervisors and requeues
unstarted attempts. Recovery refuses a live attempt lease or process group;
a dead parent alone is insufficient evidence. Unknown launches require an
operator's explicit confirmation that all descendants stopped. No old PID
is signaled on recovery. Successful exit records command completion, not
independent validation of every image; Slayer owns its output receipts.

Tests cover concurrent agents, FIFO order, duplicate acknowledgements,
conflicting reuse, cancellation, failure retention, ownership, interrupted
work and exact argument forwarding. Controlled processes exercise the native
worker and resource boundaries without rendering production jobs.
