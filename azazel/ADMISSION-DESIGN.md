# Proposing ordinary admission

This next private value function composes the host assessment, stop
transition and identified memory reservation. It proposes one update;
it grants no permission to launch, resume, release or signal a process.

The caller supplies a validated current stop value and reservation ledger,
the installed host policy, a newly supplied host sample and an identified
memory request. The transaction must bind those inputs to its locked,
validated boot/policy records. This function performs no I/O or clock read.

## The order matters

Assess the supplied measurements without using the sample's cached stop
bit. The current state's stop value is authoritative for this proposal.
Feed that assessment into the existing stop transition as an ordinary
observation, with no explicit resume request.

If the resulting stop is latched, report stopped and propose no reservation.
This includes a new low-disk stop, an existing stop and exhausted stop
generation. Return the stop transition for the transaction to handle;
invalid reservation inputs must not hide a known disk stop.

Otherwise, any host assessment other than ready holds admission without
adding a reservation. Unknown or invalid measurements, invalid policy,
high load and high swap remain distinct host reasons in the result.

Only a ready, unlatched host reaches the identified ledger addition.
Invalid ledgers, duplicate IDs and insufficient configured byte capacity
hold admission with that reservation reason and no next ledger.

A successful proposal carries the independent next ledger and unchanged
clear stop value. It does not return a live grant. The transaction still
needs observing/prepared intent publication, state replacement and intent
removal under its stable lock before it can consider a grant.

The zero proposal is unknown and has no next ledger. A hold's returned
stop value is not a persisted record or an acknowledgement of resume.
Every output must be interpreted through its disposition.

## What this operation cannot establish

An observation never clears a stop. Resuming remains a separate explicit
operation with generation, fresh host conditions and prior-target
quiescence checks. This function cannot substitute for that transaction.

Known disk loss must still stop identified running work if persisting the
stop later fails. Returning a proposed stop does not perform that action.
Input freshness, root identity, codec integrity and complete descendant
closure remain requirements of the later adapters and transaction.

Processor accounting and host residency versus unrealized reservations
remain separate policy work. A successful configured-memory proposal
does not claim a complete physical-memory or processor budget.

## Tests before the body

Pair ordinary ready admission with stopped, unknown, invalid-policy,
load, swap and disk refusals. Combine low disk with an invalid request
and corrupt ledger to verify that resource validation cannot mask the stop.

Pair an eligible clear state with exhausted stop generation. Poison the
sample's cached stop bit in both directions: the locked state must win.
A favorable observation of a stopped state cannot propose a reservation.

Keep exact-fit and overflow/duplicate ledger witnesses through the real
composition. Assert the complete stop transition and each host/reservation
reason. Refusals have no next ledger, and every input ledger stays unchanged.

Rapid varies measurement availability and pressure across observation
sequences. Once low disk latches a stop, no later ordinary observation
may produce an admission proposal. A paired ready-host property compares
the composed capacity decision with an independent unbounded sum.

These are pure value tests, with checked authored data and no fakes,
native files, host sampling or model calls. Write tests, friendly guide,
types and an empty function in the author's order. Read actual compile/red
results before implementing this single composition.

The stub at ef25297 compiled with no tests run. Thirteen of fourteen
tests then rejected the empty result through assertions; the zero-result
anti-test passed. Both exact Rapid repros were read before implementing
the body and committed unchanged at b9a2919.

The body passed all 55 budget tests, including nine Rapid properties at
100 cases, and every production statement. Focused mutation selected
eight faults; all eight were caught, with no other outcomes. The complete
logs, coverage profile and mutation rows were read.
