# Shared work limits

This private ledger proposes changes to memory, CPU-worker and GPU-slot
reservations together. Duplicate IDs, invalid owners and malformed
ledgers are refused. An accepted proposal returns an independent ledger.
A release requires the exact owner incarnation.

The transition applies a supplied host policy and stop latch. Low disk
proposes a stop and unknown observations hold work. The bounded record
codec stores the policy, stop and resource values with a checksum and
predecessor digest. The digest binds the proposed transition to the checked
predecessor and the checksum detects corrupt bytes.

These are value decisions for a proposed shared admission system. The
composite command remains unavailable. The running render queue uses its
own [durable attempts and leases](DESIGN.md#lifetime-and-results).
