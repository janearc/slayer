# Finish samples a band at a time

The host streaming path has compiled and its focused finishing witnesses
have passed. [The verification record](verification-2026-11-09.md) names
those checks and the selected native scenes that ran. Matched device
fields, buffer lifetime, memory and timing still need separate evidence.

Metal already dispatches the picture in bands of at most 64 sample rows.
Previously it copied all those bands into one Go raw-sample array before
pixel averaging. That array grows with width, height, samples per pixel
and the shading-record size, even though the GPU output buffer is small.

The new `Kernel.GridBands` keeps input buffers for the call and returns
one reusable band to a synchronous consumer. The GPU renderer finishes
that band's sky and textures, adds its samples to the final pixels and
then permits the raw band to be overwritten.

## Sampling contract

The shader, camera rays, global row offsets, samples and integration
rules are unchanged by this host-only slice. A pixel retains its old
down-then-across summation order even when a band splits its sample rows.
Coverage is averaged and nearest depth is retained as before.

Band callbacks arrive in order. The finisher refuses skipped, overlapping
or malformed bands. Callback errors release retained buffers and return
the original error; a partial picture is not returned as a completed one.

The band row count still bounds each GPU command. It does not grow to
match an unusually large supersampling side. Finishing uses the current
Go CPU allowance instead of starting a worker for every hardware core.

The old full-array `Kernel.Grid` remains available to callers that need
its result. The tracer now uses the streaming consumer.

## Memory and accounting

Raw readback now needs one Go band and one Metal output band, plus the
final pixel frame. CPU-generated non-pinhole rays, encoded geometry,
Metal input buffers, native acceleration structures, weather grids and
image development remain separate allocations.

The streaming input bridge also avoids its earlier whole-input C byte copies.
Each pointer-free Go byte slice is passed to one synchronous C call that
creates a Metal-owned copy, then returns. No Go input pointer is retained in
the stream.

Empty inputs keep their existing one-byte placeholder and buffer order;
partial allocation failure closes the stream.

This follows Apple's copying [newBufferWithBytes
API](https://developer.apple.com/documentation/metal/mtldevice/makebuffer%28bytes%3Alength%3Aoptions%3A%29?language=objc)
and Go's [cgo pointer rules](https://pkg.go.dev/cmd/cgo#hdr-Passing_pointers).

It removes an intermediate copy; it does not make Metal read Go storage
directly or establish a measured process-memory saving.

`Profile.SampleBytes` records the logical raw samples consumed. On a
successful frame it has the same count as the previous full-array result.
`PeakSampleBytes` records the largest Go readback buffer handed to the
finisher. It excludes Metal's separate band and is not a process RSS peak.

CPU finishing happens between dispatches. Its measured wall time is
separated from dispatch/readback wall time; GPU timestamps remain nested
inside dispatch time. Native-query counters accumulate across all bands.

This does not resolve the earlier woodland preparation failures by itself.
Their retained geometry and encoding allocations still need investigation.
It also does not reduce the number of rays or mist samples being traced.

## Review inputs

Compare raw streamed bands against full-grid results with global row
addresses, short final bands and callback failures. Compare finished pixels
against the original full-array averaging, including sample sides that do not
divide 64, sky, textures, coverage, depth and native counters.

Also check direct input order, empty placeholders, failure after a successful
input allocation, caller storage released after preparation, and the runtime's
stricter cgo pointer checks.

Then measure a bounded 4K pilot, including preparation and development.
No speedup, byte equality or lower process peak has been established yet.
