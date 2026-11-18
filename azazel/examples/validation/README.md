# queue validation recipes

Copy these jobs into fresh shared render folders and replace every `/ABS`
path before submission. Include the copied job in the Azazel request inputs.
Run each through `slayer draw JOB --fresh`, submitted to Azazel. Do not run
render smokes outside the queue. These recipes test execution, not maturity.

The still is 64 by 36 pixels. The film has three frames at 2 fps over
1.5 seconds; H.264 output pads the 36 rows to 48. Both use one primary ray.
The required `$schema` identifies a Slayer job rather than a set file.

For a quick check, use `doc/examples/jobs/first-picture.yaml` with explicit
160 by 90, one-ray preview overrides and a fresh output path. Record the
branch, revision, complete command and known defects beside each output.

The 2026-11-15 validation results are in `azazel/VALIDATION.md`.
