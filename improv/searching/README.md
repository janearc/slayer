# heard.

A wordless film set to Glass Hour. Two observatories search the sky.
Their signal is answered, and the view pulls back from the instruments to
the beings and structures behind the answer.

The v6 take contains 2,005 rendered frames at 480 × 270 and 6 fps, with
the full score at its original speed. A separate review contains 76
2K stills. See [the delivery record](PROOF-v6.txt) for the artifact names,
producer revisions and checks. [The making-of](MAKING-OF.md) records visual
limits and how the scene and timing evolved.

- [Screenplay](SHOOT.md) and [score](MUSIC.md)
- [Measured catalogue positions used in the pullback](CATALOGUE.md)
- [Earlier v3](PROOF-v3.txt), [v4](PROOF-v4.txt) and [v5](PROOF-v5.txt) proofs

## Inspect a plan

After building Slayer's commands through Game, use
`bin/searching plan PLAN.yaml` to print the checked cue times without
drawing. [The original edit](shoot.yaml) is an earlier proof plan.
Use the music and score files named by the plan. Its beat grid governs
story cues and camera motion, and its separate scenery clock governs
the passage of days. An excerpt keeps the original frame and audio clock.

`bin/searching --help` lists the producer commands. Submit still and
sequence renders through [Azazel](../../doc/azazel.md). Keep the producer
binary and declared inputs unchanged while queued, and use a fresh output
destination. The `render` action can write PNGs and a movie with the
matching audio interval. `--defer-movie` releases scene resources before
encoding. `--serial` disables draw/output overlap for a comparison.

The [flow study](studies/heard-flow-720.yaml) uses authored paths, luminous
landmarks, and a fictional return destination. It illustrates the answering
pulse. The measured sky catalogue is a separate input.

## Authored anatomy mesh sharing

Heard's `individual` stages with an `anatomy` plan and its
`authored-multitude` stages share identical component meshes by default.
Materials, placement and animation remain per component. To compare
separate mesh storage, set `share-anatomy-prototypes: false` beside
`kind` in a named Heard stage or film shot. This field is limited to those
two stage forms, and an explicit value changes the job's picture identity.
