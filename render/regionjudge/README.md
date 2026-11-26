# Comparing a region's light

This package compares two equally sized frames before display clipping. It
reports mean absolute error per RGB channel, and the signed mean difference
of candidate minus reference for each channel. A negative signed difference
means the candidate has less light in that channel. It also reports the mean
of the three channel errors as a single distance.

The reference is a
finite estimate, not ground truth; preserve its identity and measure its
variation alongside these numbers.

The separate isolated count is diagnostic. A pixel qualifies when its
candidate-minus-reference luminance exceeds a caller-supplied positive
threshold and none of its eight neighbours exceeds that threshold. Luminance
uses the linear RGB weights in maths.Light. Pixels on the edge are not
counted because their full neighbourhood is unavailable.

The threshold is
in the same linear light units as the frames; choose and report it with the
result.

Genuine detail can be isolated, and a finite noisy reference can
make a clamp look closer. Neither this count nor the error establishes why a
pixel differs.

Both frames must be nonempty, equally sized, and have exactly width times
height stored pixels. RGB and the threshold must be finite; the threshold
must be positive. Coverage and depth are not compared. Invalid inputs return
an error and no report.
