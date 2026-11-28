# Mixed-level ground study

Jobs 301–302 used experimental binary f00f1865c. The body-space field,
filter, 10 km height scale, two cameras and 800 × 800/2-ray settings match
the cube-sphere study; the geometry is instead a whole level-five mesh with
one level-six patch around a measured target, with projected midpoint
constraints. The selection is hard-coded for the study and is not camera
adaptive. The binary did not record stitch displacement, so no quantitative
seam or error claim is made.

The globe looked continuous in this one orbital frame, but the land shows
a conspicuous diagonal crosshatched grid more visible than the uniform
level-six control. The close target still reads as mottled paint without
recognizable mountains or cast shadows. Neither image proves seam safety;
the result is diagnostic, not finished terrain. The saved jobs write both images and PFM
raws to `out/studies/2026-11-27-relief-mixed-{orbit,target}/`.

Job 303 used diagnostic binary 2fe3fb1bb with the same field, fan-mesh
builder, camera and light but uniformly level-five patches. Its log says
maximum stitch movement 0 m. The same crosshatch is visible without the
single finer patch, so that transition is not necessary for the artifact.
The comparison does not isolate a single remaining cause. Its saved job writes
the image and raw to `out/studies/2026-11-27-relief-uniform5/`.
