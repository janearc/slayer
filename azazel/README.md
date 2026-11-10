# azazel

Azazel is Slayer's planned queue for frames, films and checks.
The [design](DESIGN.md) describes it; no runnable queue exists yet.

You should be able to ask for a render, leave it waiting, and come back to
a verified result while the machine stays usable. Its record will identify
the executable and inputs, resource use and checks. Interrupted work will
resume from verified outputs.

The private pieces cover memory and host decisions, stop transitions,
[snapshot publication](internal/publication/README.md) and
[nonblocking locks](internal/locking/README.md), plus
[native process observations](internal/processinfo/README.md).

Eighteen implemented functions have focused passing checks at the recorded
development pins. Named cleanup serves replacement; identified reservations
and ordinary admission remain proposals rather than live grants.

The [local design](internal/publication/LOCAL-DESIGN.md) records publication
witnesses and source reviews. Selected comments have a reviewed Qwen pass.
Final independent acceptance and private test-boundary design remain open.

Shared admission, recovery, supervision and engine integration remain.
The command and configuration will use Pzm. Flipr is not required.
