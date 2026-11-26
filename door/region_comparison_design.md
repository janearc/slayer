# Comparing regions from the command line

The experimental `compare-regions` verb takes one YAML set path. Puzzlemouth
parses the verb and flags. The caller names an existing frozen snapshot and
its input files, asserts that list is complete, and supplies frame dimensions,
sampling and separate reference and output directories.

The command draws each rectangle, compares its linear light to a finer
retained reference, and writes a report and labelled sheet after all regions
succeed. A dry run draws and writes nothing. Existing output and damaged
references are refused. See `doc/region-comparison-design.md` for the intended
identity and evidence requirements.

Each result contains a reference plan that can be selected on a later run.
An existing reference's origin is caller-attested, and the current session
source requires matching input and effective settings apart from sampling.

The caller's input list and stable-cache assertion are not independently
verified as complete. The full street input closure, device identity, reference
convergence, timing separation and project-wide gates remain unfinished.
