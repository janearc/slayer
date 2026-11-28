# Render admission and Azazel

on a shared machine, submit foreground render commands through
[Azazel](azazel.md#submit-work). `azazel work` dispatches CPU and GPU
work directly. Game’s build and test lanes have separate coordination.

the worker uses observed host pressure and disk space for admission.
unknown observations hold new work. pressure can latch a pause, and
low disk always does. the [worker scheduling guide](azazel.md#worker-scheduling)
explains tuning and recovery.

`azazel render` is the older direct set-rendering adapter. the
`azazel composite` admission entry currently refuses direct starts.
for the supported foreground commands use the queue’s `submit` path.

## Memory limits

group RSS, process footprint, Go heap and Metal counters measure
different things. `GOMEMLIMIT` is a soft Go heap target and Azazel
removes inherited values when it starts a render. see
[profiling](observability.md) and [workload profiles](../render/workload/README.md)
for measuring a job’s memory needs.
