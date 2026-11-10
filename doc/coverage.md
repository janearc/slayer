# Coverage

Run `game cover` from the checkout root after configuring the source shelf.
It runs the Go tests and checks coverage per source file. A passing test suite
can still fail that last gate: a repository average does not clear a file
below the configured bar.

The ordinary measurement includes every Go package, including commands and
generated mocks. Nothing is omitted from its denominator to raise the score.
Metal coverage measures its Go host code; it does not measure shader branches.

## Commands are part of the program

Command tests build the real executable once in TestMain. When ordinary
package coverage is enabled, the child uses the same counter mode and package
scope as its parent test. It inherits GOCOVERDIR, and Go collects those child
counters when the test finishes. Ordinary checks build an ordinary binary.

Matching scope matters. Go's test coverage collector accepts pods whose
metadata hash matches the test binary. A child built with every dependency
instrumented does not contribute to a test instrumented for its own package
alone. See internal/coverage/cfile/testsupport.go in the Go toolchain.

The tests decode actual PNGs and linear raws, check cache reuse and refusal
paths, and capture terminal output in their own pipes or pseudo terminals.
Grissom's small data fixtures are synthetic; they are not survey measurements.
No test opens a viewer window or sends a picture to someone else's terminal.

## Smaller runs

These fixed Game runs keep useful groups available while working:

- `game run check-coverage-focus`: command entry points and the Grissom viewer.
- `game run check-coverage-scenes`: ring, swarm, weather and planetary models.
- `game run check-coverage-cache`: weather and render cache contracts.
- `game run check-coverage-terminal`: captured Show and Quorum protocols.
- `game run check-coverage-set`: authored YAML and its public views.
- `game run check-coverage-mutator`: tiny temporary mutation fixtures.

Each writes a separate profile under bin/.cover. These focused results describe
their named packages. They are not a whole-repository result.

## Changes found by these tests

Weather's wind summary now advances even on grids with fewer than 32 rows.
Its command rejects non-finite or negative durations and non-positive or
non-finite output intervals before creating files.

The ring's face winding now follows each strip's outward direction. At low
polygon counts, testing the face against the circular midline could turn the
outer hull inward and make mesh validation refuse it.

Cooking haze requires a top atmospheric layer in every zone. A body with an
empty column now refuses that request before face preparation, which formerly
indexed the missing layer and panicked.

## Reading the remaining gaps

Keep defensive returns and opaque object invariants intact. An allocation
failure or an internal error that a valid constructor prevents is still in the
denominator; a test should not corrupt an object just to reach it. Generated
helpers that have no caller are also visible.

Whole-program instrumentation is a different experiment from ordinary Game
coverage. On Darwin/ARM64 it currently changes some conifer vertex bits and
fails the existing raw geometry references.

Keep those references unchanged; that failing run is not evidence of a passing
suite. The ordinary package measurement remains the reproducible baseline for
this work.
