# Queue smoke recipes

Copy these jobs into fresh render folders and replace every `/ABS` path
before submission. Include the copied job in the request's declared inputs.
Submit `slayer draw JOB --fresh` through [Azazel](../../../doc/azazel.md).

- `still.yaml` requests a 64 × 36 still with one ray per side.
- `film.yaml` requests three 64 × 36 frames at 2 fps. Its H.264 output
  pads the frame to 64 × 48.

The required `$schema` identifies each as a Slayer job. Keep the command,
revision and settings beside the outputs. See the
[original validation](../../VALIDATION.md#native-jobs-through-azazel-and-game)
for observed results.
