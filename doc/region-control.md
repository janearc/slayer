# Repeatability control

On release candidate `19dce148`, smoke 1 was drawn twice on CPU and twice
on software Metal, in separate invocations. Each picture was 48 by 32
pixels with two rays per side. For each backend the two PFM files and the
two PNG files compared byte for byte equal. The draws reported no kept rows
on CPU. This is evidence for this small scene and configuration only; it
does not establish repeatability for every scene or native Metal traversal.

The clean build used Game and verified the local source shelf first. The
control jobs and resulting files are kept beside that build.

The focused region checks on the work-in-progress branch passed for job,
trace, Metal, GPU, session and door. They include CPU and software Metal
whole-frame comparisons, with nonzero origins and a spread filter, and a
developed CPU pixel selected where its neighbour differs. They do not prove
the street image, native Metal, or the full project gates. Separate focused
checks passed for the comparison, reference, input snapshot and set packages.
