# Looking at a process's native timestamps

Give this private Darwin reader a positive PID that fits the native API.
It asks libproc for that process's start and exit stamps, in native Mach
ticks. It uses the SDK's declarations, so Go does not guess the C layout.

An argument refusal or native error carries no observation. The native
error remains inspectable with errors.Is. A successful result reports
what the call returned; it does not promise the process will still exist
when you act on it.

The API can also describe a zombie. `CaptureOwner` now binds two matching start reads to a validated boot UUID.
Persist its canonical `Incarnation()` token with the transaction owner; recover
it with `ParseOwnerIdentity`, and compare it with `ObserveOwner`. Do not replace
a stored identity by recapturing an old PID. Parsing does not authenticate the
record or bind it to the supervisor's actual launch.

`OwnerDead` describes only that exact process incarnation, never its children.
`OwnerPresent` is an observation during the call, not a promise of continued
liveness. Errors and boot mismatches remain `OwnerUnknown`.

`ObserveQuiescence` requires a context deadline and samples direct children and
the current process group with fixed native bounds. Its lists are diagnostics,
not ownership proof.

It always returns `QuiescenceUnknown` with
`ErrDescendantsUnknown`: even an empty group or dead leader cannot establish
that escaped or reparented descendants ended. Do not use it as a transaction
release proof. No signals are sent. Native read calls are synchronous; context
checks prevent further work but cannot interrupt a kernel call.

The [design](DESIGN.md) explains that boundary and the bounded native
witnesses. Tests use the current process and a controlled child fixture.
