# Proposing an identified memory reservation

The existing memory predicate answers whether one amount fits. This next
pure operation uses it to propose an updated ledger of identified memory
reservations. It performs no I/O and returns no admission grant.

The private ledger holds a configured byte ceiling and a sequence of
reservations. Each reservation has an opaque nonempty identifier and a
positive byte amount. An identifier is not a PID or proof of owner identity.

The eventual transaction validates the boot, policy, owners and record
bounds under the stable lock before calling this operation. It also checks
fresh host conditions and the persisted stop. Those checks remain separate
from proposing the ledger's configured-ceiling arithmetic.

## One addition

Refuse a missing ceiling, empty requested identifier or zero requested
amount. Validate every existing reservation before proposing any update:
identifiers must be nonempty and unique, amounts positive, and the entire
existing ledger within its ceiling without unsigned wraparound.

A malformed existing ledger is invalid, including duplicate identities or
an overcommitted sum. Do not silently repair it by dropping entries or
treating a failed measurement as zero. Validation precedes a capacity
decision about the requested addition.

An identifier already present is refused as a duplicate, even when its
amount is unchanged. Addition is not an implicit replacement or release.
Request idempotency, if needed, belongs to a separately specified operation.

Otherwise, allow an exact fit and refuse an amount above the unreserved
capacity. Use the existing overflow-safe memory predicate for every
accumulated amount and for the new request. Check before adding.

A successful proposal preserves every existing reservation and its order,
then appends the requested reservation. Its storage is independent of
the input slice, including spare capacity. Mutating either ledger after
the call cannot change the other through shared reservation storage.

Every refusal has a distinct non-success disposition and no proposed
ledger. The zero result is unknown, with no proposed ledger. A caller must
never interpret refusal as a newly empty book or capacity becoming free.

The disposition distinguishes invalid inputs, duplicate requested identity,
capacity hold and a proposed addition. It does not acknowledge persistence,
authorize a process launch or prove that a corresponding child has ended.

## Test and anti-test pairs

Authored cases pair an empty-book addition with omitted amounts and IDs;
an exact fit with one byte beyond it; multiple existing reservations with
the request that would spend their capacity again; and full-width valid
arithmetic with the addition that would wrap.

Existing malformed entries are refused independently: empty identifier,
zero amount, duplicate identity and an overcommitted or overflowing sum.
Both a same-amount and different-amount requested duplicate are refused.
The original ledger must remain unchanged after each refusal.

Ownership witnesses use an input slice with spare capacity, inspect all
old entries and their order, then change each returned/input entry in turn.
The other ledger must remain unchanged. A zero result cannot claim a
proposal, and a refusal must have no next-ledger pointer.

Rapid compares capacity decisions with an independent arbitrary-precision
sum across the entire uint64 range. Another property exercises repeated
identified additions: each accepted proposal preserves the prior prefix;
refusals create no proposal and never change prior reservations.

Examples and bounds live in checked local test data, with exact uint64
decoding and input-role validation. No float conversion or hand-written
filesystem/process stand-in participates. The tests need no host samples,
timers, model, native files or live configuration.

Write these tests and their anti-tests before the friendly guide and
types. Compile them against an empty function, read their actual assertion
failures, then implement only this function. The shared record format,
codec and reservation authority remain unimplemented by this value layer.

The stub compiled at a7a0a79. Twelve of thirteen tests then rejected its
empty result through assertions; the zero-result anti-test passed. Both
exact Rapid repros were read before the body and committed unchanged.

The first body at 4544dd9 passed all 41 budget tests, including seven
Rapid properties at 100 trials each, with 100% production statements.
This is focused development evidence; broader acceptance remains open.
