# Finish samples a band at a time

the Metal trace dispatches in bounded bands of sample rows.
`Kernel.GridBands` keeps input buffers for the call and returns
one reusable band to a synchronous consumer. The GPU renderer finishes
that band's sky and textures, adds its samples to the final pixels and
then permits the raw band to be overwritten.

## Sampling contract

a pixel retains down-then-across summation order even when a band
splits its sample rows. the finisher averages coverage and retains
nearest depth.

Band callbacks arrive in order. The finisher refuses skipped, overlapping
or malformed bands. Callback errors release retained buffers and return
the original error; a partial picture is not returned as a completed one.

The band row count still bounds each GPU command. It does not grow to
match an unusually large supersampling side. Finishing uses the current
Go CPU allowance instead of starting a worker for every hardware core.

`Kernel.Grid` is the full-array interface. the tracer uses the
streaming interface.

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

sample counts still follow the job’s tracing settings.
