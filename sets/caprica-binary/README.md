# Caprica binary: unassessed system draft

`system.yaml` is the uncommitted draft handed off on 2026-11-27,
copied unchanged from a worktree based on Slayer `6753d4163`. Its SHA-256
at handoff is
`402925184fa13134b37d454799c907a61902694a63b4aee38a926c79ae03ee8e`.
The earlier compact version was assessed, but this restored-scale revision
has not been assessed or rendered. Its proposed shots need geometric and
visual assessment before they are treated as results.

The creative brief names the lava drop **Fresno**, with Earth-like Caprica in
orbit around it; the yellow gas giant **Amber** is uncomfortably close to
the larger star. The blue ice giant is **Azure**. Fresno has an atmosphere
in the story because it has been inundated by comets for millions of years.
The brief calls for twelve Hatzegopteryx soaring in Azure's atmosphere with two
distant suns. These directions are not yet all implemented in the draft.

The draft's haze and wakes are authored wisps, its polar crown is illuminated
scattering rather than an emissive aurora, and its four comet pieces are
manually placed rather than a persistent procedural population or simulated
breakup. No deposition process is simulated. Absolute distances, local-origin
precision, lighting, stream continuity and camera geometry remain open.
The imagined animals and their presence in an ice-giant atmosphere are
explicit artistic choices.

## Reproduce the Azure jobs

Run a job from the checkout with `game run slayer -- draw` and its YAML
path. Each job resolves its scene beside the job file and writes
`proof.png` and `light.pfm` under `out/<date>-<shot>/` at the checkout
root.
