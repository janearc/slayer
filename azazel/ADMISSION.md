# Asking what ordinary admission would propose

This helper brings the value decisions together. Give it the checked
current state, host policy, supplied measurements and identified memory
request. It checks host conditions and the stop before proposing capacity.

Low disk proposes a stop even if the memory request or old ledger is bad.
A latched stop stays stopped after a favorable observation. The sample's
cached stop bit cannot override the current state's stop value.

When the host is ready and clear, the helper asks the identified ledger
to propose the addition. A refusal tells you whether the host held work
or the reservation was invalid, duplicated or too large. It has no next
ledger to publish.

A successful value proposal is still waiting for the transaction. You
must bind its inputs to validated records under the shared lock and
finish the intent/state publication protocol before issuing any grant.
Resuming and signaling running work remain separate operations.

The [design](ADMISSION-DESIGN.md) records the precedence and paired tests.
The stub compiled, then thirteen tests rejected it through assertions;
the zero-result check passed. The first body passed all 55 budget tests
and nine Rapid100 properties with full production statement coverage.

All eight generated mutations in this function were caught. These are
focused development checks; final acceptance and the shared transaction
remain open.
