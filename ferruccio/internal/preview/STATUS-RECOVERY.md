# Recovering queue status

A temporary failure to run the queue status command is not evidence that a
render failed. After the ordinary bounded retry batch, keep the acknowledged
job and its frozen input, report that status is unavailable, and retry at a
bounded cadence until status returns or the user cancels. Never accept pixels
without the matching terminal ledger record. An unreadable or mismatched
ledger still follows the existing conservative identity-failure path.
Explicit cancellation retains the existing owned-job confirmation and cleanup
rules. Test outage and recovery, cancellation during an outage, and rejection
of mismatched identity; failures must not submit a replacement job or claim
that unobserved work is running or complete.
