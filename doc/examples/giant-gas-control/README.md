# Giant gas comparison

The local Azure flock scene and four profile books are copied from the
giant-only control rendered by Azazel job 578. The
giant, its banded appearance, flock, sky, lights, camera and exposure are
retained; local wisps are absent. `default.yaml` omits the gas override;
`opacity-4.yaml` adds depth 0.08 and opacity 4, compared with the default 40.
Both were rendered on the same `2dfb4bb89` binary, at 960×600, one ray per
axis, jobs 586 and 587. The jobs now write pictures and raw light in `out/`
`2026-11-27-giant-gas-default` and
`2026-11-27-giant-gas-opacity-4`.

The builder saw a softer, lower pale transition with opacity 4; independent
inspection judged the change marginal, with both frames retaining a smooth
pale field and no recognizable cloud structures. The images do not identify
the field's cause or establish physical correctness. `opacity-0.4.yaml` is a
preserved unrendered option, not a result. The gas override does not alter the
existing opaque-core clipping described in `doc/model/sky/giant-gas-look.md`.
