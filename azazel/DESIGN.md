# Azazel scheduler design

Azazel owns render admission and job supervision. Slayer owns scene
interpretation and rendering. A request wraps a foreground command and
its argument array. Game builds and checks the software. See
[the operator guide](../doc/azazel.md) for current commands and policy.

## Requests and identity

The request contract carries an agent, idempotency key, executable,
arguments, working directory, environment, declared input files, resource
lane and timeout. Submit validates the schema and records executable and
input hashes. Execution checks those identities again. Clients declare
the referenced inputs whose identity matters.

The durable ledger gives each submission a sequence. Retrying unchanged
content with the same agent and key returns the existing acknowledgement.
Changed content with that key is refused. Output locations belong to the
producer. Choose fresh destinations for new work.

## Admission and stopping

One dispatcher owns a queue. It selects eligible CPU and GPU work
independently and starts Azazel-owned supervisors. The default is two CPU
attempts and four GPU attempts, with five seconds between GPU starts.
Live tuning can change the GPU concurrency and spacing.

The queue checks available disk and system memory pressure. Unknown
samples hold new work. Low disk latches a stop. Elevated memory pressure
also latches a stop under the default policy. The observe policy reports
pressure while allowing work to continue. Reported process-group residency
helps diagnose load. Disk and pressure stops require explicit resume.

## Utilization under backlog

The GPU scheduling target is 100% device utilization while runnable GPU
jobs wait, subject to host checks and attempt ownership. Measure the GPU
counter and sample time alongside completed work per unit time. A busier
device can still deliver less throughput. Scheduler status records active
attempts, the device observation and the reason admission is held.

CPU work can proceed while GPU work runs. Other GPU users and unrelated
queues do not share Azazel leases. Operators coordinate them separately.

## Lifetime and results

Each attempt has a renewable lease, a supervisor, a process group and an
attempt lock inherited by the command. A stopped dispatcher leaves active
supervisors running. Restart adopts surviving supervisors and requeues
attempts that never executed. Lease expiry alone does not authorize replay.
An uncertain attempt stays fenced until the worker or operator resolves it.

The queue records command success, failure, cancellation and interruption.
It retains failed attempts without automatically retrying them. The
producer owns its outputs and any verification it performs.
[Gallery publication](pages/README.md) observes jobs independently and
records its own failures.

## Private admission experiments

A separate set of budget and storage primitives explores cross-engine
admission and durable reservation records. These contracts are distinct
from the live queue's lease and dispatch mechanism.

- [Ordinary admission proposals](ADMISSION-DESIGN.md)
- [Reservation proposals](RESERVATION-DESIGN.md) and
  [owned reservation proposals](OWNED-RESERVATIONS.md)
- [Stop transition draft](STOP-DESIGN.md)
- [Work limit values](WORK-LIMITS.md)
- [Locked publication transaction](TRANSACTION.md)
