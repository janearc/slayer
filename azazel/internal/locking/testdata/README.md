# Keep the lock witnesses bounded

`locking.yaml` keeps the original two-second attempt timeout, generated
repeat range from one to eight, and four repeated release attempts.
Pzm reads exact integer nanoseconds and counts; setup refuses missing or
nonpositive values and reversed ranges.

Each test loads its inputs once. The generated properties retain a copy,
so a native attempt does not parse data while a competing handle is held.
Each timer still starts after the attempt goroutine, and that goroutine
still sends exactly one result to a one-slot completion channel.

These are witness limits, not production locking policy. The kernel
checks, handle and byte ownership, cleanup order and retained Rapid
counterexamples remain unchanged.
