# Owned reservation proposals

The private budget package now has a separate pure owned-memory ledger. It does
not replace the existing admission composition or enable composite or render.

An entry carries a typed reservation identity, an owner name, an owner
incarnation identity and a positive byte ceiling. Owner names distinguish callers;
incarnation identities distinguish attempts of the same named caller. These are
supplied values, not authenticated process identities. Nonempty values are
required. Reservation identities are unique across the entire ledger, including
entries with different owners.

Addition validates every existing entry and the configured memory ceiling before
checking duplicates and remaining capacity. Exact fits are allowed, including the
full unsigned range. Invalid or overcommitted ledgers cannot be repaired by an
addition or a release.

Release requires the reservation identity and both owner fields to match exactly.
A stale incarnation, another owner, an unknown reservation or malformed input
returns a distinct refusal and no next ledger. An unknown reservation is not an
acknowledgement of successful release.

Releasing the final entry preserves the
configured ceiling; a repeated release reports not found. Successful additions
and releases return independent storage and preserve the order of retained entries.
The zero result is unknown, not success.

A release proposal does not establish that work has stopped. A later transaction
must authenticate the caller, bind incarnation identities to its validated durable
records, establish complete process-group quiescence where required, and persist
any release under the stable shared lock before capacity can actually be reused.
These helpers perform no recovery, process observation, signaling, I/O or clock
reads. They do not authenticate owners or issue admission grants.

Still missing for runnable shared admission: a reviewed durable record and codec,
boot and policy binding, locked observing/prepared intent transactions, durable
state replacement and intent reconciliation, resource accounting beyond configured
memory, fresh host sampling, safe process ownership and descendant recovery, and
integration with composite, render and Game's first and subsequent lanes. The
existing publication helper explicitly does not guarantee machine-crash durability.

Tests were written first, failed compilation against missing types, then compiled
against empty operations and failed assertions. The implemented operations passed
the requested focused budget suite. Cases cover named owners, stale incarnation
refusal, globally duplicate reservation identities, exact-fit and overflow
arithmetic, complete-ledger validation before release, independent storage,
retained order, final and repeated release, and zero-result refusal.
