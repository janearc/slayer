# Looking at a process's native timestamps

Give this private Darwin reader a positive PID that fits the native API.
It asks libproc for that process's start and exit stamps, in native Mach
ticks. It uses the SDK's declarations, so Go does not guess the C layout.

An argument refusal or native error carries no observation. The native
error remains inspectable with errors.Is. A successful result reports
what the call returned; it does not promise the process will still exist
when you act on it.

The API can also describe a zombie. The supervisor still needs to bind an
observation to its expected boot and process identity before using it.
Signaling and proving that a whole child tree has ended require their own
ownership and lifetime checks.

The [design](DESIGN.md) explains that boundary and the bounded native
witnesses. Tests use the current process and a controlled child fixture.

The nine native checks pass, including both generated properties at one
hundred cases. Deliberate faults in the birth field, native cause, PID
narrowing guard and return-status check each fail their intended assertions.

Game's automatic source selection omitted this cgo file, so those explicit
faults remain separate evidence rather than an automatic mutation score.
