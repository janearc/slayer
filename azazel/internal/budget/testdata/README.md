# Keep the admission examples exact

`memory.yaml` keeps the original 23 authored ledger examples, with the same
names, byte counts and expected decisions. Keeping them in data makes the
cases easier to inspect beside the assertions.

Pzm reads byte counts and generations as unsigned integers, including the
largest unsigned integer and nearby values. Load averages keep their
original floating-point representation.

The memory oracle still adds with arbitrary-precision integers,
independently of production subtraction. All fixture migrations keep the
original property labels and bounds. Empty groups fail fixture setup.

`host.yaml` preserves boundary samples, invalid loads and policies, all
eight availability combinations and the latched-stop sequence. Expected
decisions stay typed Go constants.
Setup checks the adjacent float and every distinct availability tuple.

`stop.yaml` keeps the original generation examples, stale requests and
history lengths. Setup keeps distinct ordinary and final resume cases
and checks stale neighbours against the stopped generation. The history
still checks eligible resumes and counter advances independently.

These values are test examples, not host measurements, installed policy
or authority to start work. Production code and retained counterexample
bytes are unchanged by these migrations.
