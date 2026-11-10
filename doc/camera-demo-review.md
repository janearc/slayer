# Camera and demo checkpoint

Camera travel, timed easing, holds and resumable frame output are implemented.
The demo menu draws actual smoke and named scene factories. Miami-inspired
scenery, typography and HUD work remains planned; see the
[reference note](miami-reference.md) and [backlog](feature-backlog.md).

Work stays on the local Slayer branch `feat/scenery-and-sequences`.
The separate Game branch `fix/run-demo-arguments` is at `f6f33e3`.
Nothing was pushed or installed globally. Local repository locations belong
with the review receipts rather than in the published documentation.

## Validation and its scope

Full Game check passed on `b12ddef`: Go formatting, vet and all package tests,
including the 39-smoke comparison, GPU and Metal. That run used two CPU slots,
serial packages, a 30-minute package limit and no Go test result cache.
Normal renderer caches remained enabled.

The first one-slot run failed a concurrent Grissom error assertion and timed
out after 15 minutes rendering smoke 29. The assertion now requires the
requested missing directory and filesystem cause, without choosing which sky
worker fails first. No image size, comparison threshold or test was removed.

Later changes strengthen tests, name one local error explicitly, and reflow
prose and a condition. Changed test packages passed focused Game checks;
the dispatcher mutations also run its complete real-command test suite.
No rendering algorithm or scene data changed after the full check.

Core camera, checked view and pinhole scope: 135 mutations, 123 killed,
12 that do not compile, no survivors, timeouts or equivalence entries.
Dispatcher and budget guards: 26 mutations, 22 killed, three that do not
compile, one documented immutable-catalogue equivalence and no survivors.

The catalogue judgement has its own source key; preview-input validation was
tested and killed separately. All budget faults were caught during dry runs,
so a removed guard cannot allocate the oversized test image.

A real Metal probe passed on the Apple M3 Max using procedural Hillside
scenery. It checks changed views, repeated out-of-order sampling and measured
hardware dispatch. The self-contained smoke clip uses the CPU; its legacy
lamp types are not supported by the GPU route.

Source verification and policy-bearing Game sweep passed. New files have no
shuriken findings. Whole-module shuriken remains red on existing findings;
Game lint has no hard errors but still reports existing review findings.
These are not claims that every project gate is green.

The complete demo, adapter and input mutation scope remains open. Its broad
calibration refused a three-minute cap after the unchanged Hillside suite
took 2m23s. The completed narrower scopes above do not replace that work.

## Visible output

The final CPU camera demo was rendered from `64b46bd` through the rebuilt
companion Game. It produced 96 actual 640x360 PNGs at 24 fps, with 73 distinct
views and a 24-frame endpoint hold. Rendering took 18.78 seconds; the timed
process reported 253,313,024 bytes of maximum resident memory.

Every frame hash and timestamp matched the resume manifest. The encoded
four-second movie was decoded and counted as 96 frames. Start, middle and
endpoint PNGs were inspected.

Local output is under `out/feature-review/final-demo/`: `camera.png`,
`camera.mp4`, numbered frames, `verification.json` and `video.json`.
Reproduce it with the companion Game:

```sh
game run demo -- camera --out out/camera-review
```

Use a new directory after changing code or a plan. The guide explains the
[demo commands](demos.md) and [camera YAML](camera-motion.md). Encoding uses
FFmpeg; producing PNGs does not require it.

The woodland preview is still blocked by preparation memory. Two attempts
crossed their 1.25 GiB aggregate RSS watchdog before producing a frame. The
small camera clip is not evidence that this performance issue is solved.
