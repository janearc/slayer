# Keeping a proposed reservation book

Give this private value helper the current memory ledger and one identified
request. It checks the old entries first, then proposes a new ledger if
the request fits. Every existing reservation stays in its original order.

An empty ID or missing amount is refused. A duplicate request does not
replace an older reservation, even if both amounts match. An invalid old
ledger stays invalid; this helper never frees capacity by repairing it.

A refusal carries its reason and no proposed ledger. A successful result
owns separate reservation storage, so changing it cannot alter the input.
The zero result proposes nothing.

This is one part of admission accounting. A proposal still needs the
transaction's host checks, stop check, verified identities and publication
under the shared lock. It cannot start a child or release an old grant.

The [design](RESERVATION-DESIGN.md) records the paired examples,
full-width arithmetic and ownership checks. The stub compiled, then twelve
tests rejected it through assertions while the zero-result check passed.
The first body passed all 41 budget tests and 100% production statements.
