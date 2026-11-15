# Experimental stage trace

The hybrid GPU path accepts `--trace PATH`. It records per-frame wall spans for cues, lighting, ocean, matte, mist, develop and output hand-off, then writes a Chrome trace after successful movie publication. Open the JSON in Perfetto or Chrome's trace viewer.

These are host wall spans. GPU kernels are currently submitted synchronously, so a span includes its allocation, dispatch, waiting and readback. It is not a Metal GPU start/end timestamp, a utilization sample or a benchmark report. The trace file is installed without replacing an existing file. Trace write failure can occur after the movie is already complete.

The GPU trace, renderer and trace tests were written without being run; they pass as of 2026-11-14. Metal capture, per-dispatch GPU timings, allocation telemetry and the specified JSON-lines bench remain to be implemented.
