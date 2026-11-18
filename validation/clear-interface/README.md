# Synthetic clear-interface validation scene

This scene exercises the merged set YAML clear-material syntax using a
finite-thickness clear dielectric plate and an opaque coloured sphere whose
lower cap crosses the plate's horizontal center plane. The index value (1.333),
colours, chart, and illumination are synthetic test choices. They are not
measurements for San Onofre, Hawaii, or any water body.

The plate is a solid clear dielectric object and only a geometric stand-in for
a flat interface. The sphere is deliberately opaque and coloured so its
intersection with the plane is easy to inspect. The set has no liquid volume,
participating medium, absorption, scattering, or camera-underwater transport.

Calling the sphere “partly submerged” describes only the intended visual
analogy: this recipe does not simulate immersion. The plate's finite thickness
also means it is not an infinitely thin air-water boundary.

Run `slayer assess validation/clear-interface/job.yaml` to validate and stage
the scene without drawing. The job requests a 160 × 90, one-ray CPU still.

## Recorded validation

- Source checkout: `clear-interface-validation` in the agent's tree, based on
  `76e331e367bdc9f7d30ef579ff70ca1a3a0526cb`.
- Recipe: this `scene.yaml`; no external files or measured datasets are read.
  SHA-256 `f2528f216e8040cb96f53030b15d57abee2da2604408e7b09165a515675e9dfe`.
- Original job in that checkout: `jobs/clear-interface-validation.yaml` (the current portable job is `validation/clear-interface/job.yaml`).
  SHA-256 `379e80f939bd0a44528df14d2b1fcaa97a827a840f8c2acadb6e41c81eeaa623`.
- Assess: Game5 `511afb9`, with `GOMAXPROCS=2`, `GOFLAGS=-p=1`,
  `GAME_HOME` set to a scratch profile home for the run, `GAME_ROAD` set to
  the slayer road root and `SLAYER_CACHE=$SLAYER_RENDERS/cache`:
  `game5 lane tests -- /private/tmp/slayer-clear-interface-validation/bin/slayer assess $SLAYER_RENDERS/frames/2026-11-16-studio-clear-interface-validation/job.yaml`.
  It staged as a still and drew nothing.
- Still: same Game5 environment and tests lane, command:
  `game5 lane tests -- /private/tmp/slayer-clear-interface-validation/bin/slayer draw $SLAYER_RENDERS/frames/2026-11-16-studio-clear-interface-validation/job.yaml --fresh`.
  The job says `renderer: cpu`, 160 × 90, one ray per pixel. No GPU lane or
  window was used.
- The generated Slayer binary was an unstamped `game run heard-build` build,
  SHA-256 `28615da0c73629395958ad27ef06643f655c2a6ea65d65d7133fddd5bbd56316`.
  The source revision and recipe hashes are recorded beside the shared frame;
  the binary itself does not claim a stamped commit identity.
- Shared frame SHA-256:
  `17d7ca96de9e4bd545bdc710ed1f2acb4a41a16aa0848a97696bf5a9e25248af`.

The assess output is retained as `assessment.log` and the draw output as
`render.log` in the shared frame folder. No separate preflight image was
produced; the rendered PNG is retained there.

The portable shared copy uses `scene.yaml` and `job.yaml` in the same folder,
so it can be run without the temporary checkout. The shared scene SHA-256 is
the recipe hash above; the shared job SHA-256 is
`80d67eb71d4aee7b46a4f4c15785ccaa6f88bb4f40155ada7f7f58abe9477840`.

The gallery links the image at
`renders/frames/2026-11-16-studio-clear-interface-validation/`; the folder's
`index.html` and generated `job-yaml.txt` are retained too.
