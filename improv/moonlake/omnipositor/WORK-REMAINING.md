# Remaining work

This is a working draft, not an accepted milestone. It came into the release line on 2026-11-14 as it was written; its tests pass there, and a review of the compositor itself is still owed.

## Compositor

- Validate and complete the experimental Metal lighting and ocean stages without changing their order or silently dropping cues. Searchlight and laser geometry is still planned on the CPU; their tiled per-pixel splat implementation needs numerical and performance validation. The GPU ocean assembles open-water ray, marching, slope, sky and shading primitives behind an explicit command switch; it has been run for the measurements below and no further. Preserve the beam correction and historical PFM precision boundaries.
- Keep the passes and intermediate buffers resident across a stretch, reuse stage allocations, and overlap GPU work with output using bounded double buffering. The current hybrid path transfers between stages and recreates most sessions.
- Bind GPU backend and actual device identity to the execution snapshot before enabling verified PNG resume on the GPU path. Preserve no-clobber and cooperative overlapping-job semantics.
- Implement stage dumps, capture, GPU-timestamp profiling, JSON-lines benchmark reports, machine-state sampling and comparison against the historical chain. Experimental hybrid runs can write CPU wall-time spans as a Chrome trace; they are not GPU timings. The oracle timing primitives do not yet run the chain or prove reproducible provenance.
- Implement the specified exit codes and complete settings support. Verify the encoder on real hardware and the piece-joining behavior for 72-frame pieces and 48-frame GOPs.

## Measured on 2026-11-11, at preview size

Frames 3480 to 3600 of the film, 1024 by 751, 120 frames, each run once on one machine, straight to HEVC. These are single runs, not benchmarks.

- CPU path: 66 s. Experimental Metal path with the CPU ocean: 50 s. Experimental Metal path with the Metal ocean: 51 s.
- Frame 48 of each film against the CPU film's, after encoding: the Metal path 66.3 dB, 99.49% of pixels within one code value; with the Metal ocean 64.0 dB, 99.21%.
- The CPU path's frame 3528 against the original film's still: the same picture but for the dark box under the beam's foot, which the original has and this does not. It does not pass `compare --check`, for that reason.
- By the trace's wall times the ocean is two thirds of a frame: 30.6 s of the Metal path's run with the CPU ocean, 32.3 s with the Metal ocean. Lighting is 10 s, matte and mist 2 s each.
- The 22 GB was a leak of ours and is fixed. On that first run the Metal ocean's process was seen at 22 GB of memory at this small size and pushed the machine's swap up by 4 GB, twice. The cause was in `metal/`: a bridge function called from Go with no autorelease pool, so every buffer of every closed session was kept. It is fixed (`metal/session.m`, `metal/bridge.m`), with a one-file regression test and a sentinel that runs the package's tests with the runtime's missing-pool switch on.
- Measured again after the fix, the same 120 frames, each run once, peak memory by the process's own accounting: Metal path with the Metal ocean 47 s and 1164 MB; Metal path with the CPU ocean 45 s and 729 MB; CPU path 66 s and 668 MB. Swap did not move.
- The Metal ocean is safe to turn on and is not worth it yet: it is no faster than the CPU ocean. Stages still allocate and copy for every frame and open a Metal session each time, as written above. Why the Metal path as a whole is only about 1.3 times the CPU path's speed is not known.
- The first compile of the Metal sources failed in both packages and is fixed; the faults are in the messages of the commits that fixed them, 0cbeaef7 and 7fa64d0d.

## Scheduling

- Complete shared azazel admission for both render and composite, including authoritative ownership, memory and disk accounting, stop enforcement, crash recovery and a sound work-lifetime boundary. The existing transaction and process observations are not a live grant; process snapshots cannot prove all descendants have stopped.
- Keep composite admission unavailable until those guarantees exist. Do not treat a typed trusted-caller assertion as independently verified quiescence.

## Evidence for later

- Draw the specified historical frames and stretch through the render queue at both sizes. Record exact binaries, settings, inputs and machine conditions; compare candidate images and inspect differences.
- Run the project's coverage, anti-tests, mutation checks, lint, sweep, benchmarks and independent reviews. No benchmark, historical equivalence or QuickTime playback claim has yet been made.
