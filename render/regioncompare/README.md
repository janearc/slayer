# Comparing a set of regions

Run reads a set with `regionset`, asks a source for candidate and reference
pictures, compares their linear light with `regionjudge`, and returns the
numbers and a labelled sheet. `WriteReport` and `WriteSheet` encode that
result; they do not install files or publish a page. Reference time includes
preparation and reading and is not by itself a cache-hit indicator.

The caller supplies a candidate and reference identity for every rectangle.
The source must establish the build, effective settings and input content
represented by each identity, or refuse. Run checks the identities returned
by the source, but cannot independently audit its provenance promise.

Both pictures must have the region's dimensions in raw light and developed
view. The reference must report more rays per side than the candidate. An
error in any region returns no partial result. The source owns rendering,
reference verification, consistent development and timings. A finite
reference does not establish convergence.

`RunRepeated` can draw the candidate up to eight times. The report compares
each repeat's raw light with the first draw and records its elapsed time.
Repeatability is not reference convergence; both need their own evidence.

The labelled sheet enlarges small crops in whole-pixel blocks, bounded to an
eightfold scale, and shows the rays per side. It leaves large views at their
original size; it does not interpolate the pixels.

The session source checks the explicit scrooge mode against the environment
and refuses the trial plan. It permits CPU and software GPU; the GPU region
drawer refuses captured work-count and window diagnostics. Native GPU remains
unsupported.

The caller must account for environment-selected data and cache state. A
listed digest does not establish that every such input was accounted for. The
reference manifest must match the candidate's declared inputs, renderer and
effective settings apart from sampling. An existing reference's
origin remains caller-attested; its digest detects changes to its contents.

The session source can read an attested reference from a different executable,
but cannot generate one claiming a different build. The command can save a
reference plan and select it on a later run to keep the identity fixed. It
checks each selected manifest against the candidate before drawing, so an
incompatible plan is refused before spending time on its regions.

It also checks any retained artifact before drawing. A missing reference can
be created only by the executable named in its manifest; a missing reference
from an older build and any damaged reference are refused early.

After drawing a candidate, Run calls `EnsureReference` when the source
provides it. The session source draws and installs a missing reference,
reuses a verified file and rejects a damaged one. It does not replace an
existing reference. The experimental comparison command uses this method.

`PlanSession` derives manifests and reference paths from the job, running
executable and declared snapshot. It cannot discover the input closure: the
caller must assert completeness and keep inputs, environment-selected data
and caches stable. Each draw checks its effective settings and tone against
the planned identity.
