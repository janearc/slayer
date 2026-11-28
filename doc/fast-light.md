# Fast light

`lamp-store` and `sky-looks` approximate lighting on supported
native Metal devices. for a GPU set with boxed mist or smoke and a
pinhole camera, add this to the job:

```yaml
renderer: gpu
fastlight: on
shortcuts: {restir: off}
```

the umbrella enables ReSTIR too. leave it off for this workflow,
which uses the lamp store and shared sky looks. `lamp-store` accepts
up to sixteen lamps and conflicts with `volume-lighting: cached`.
`sky-looks` requires `lamp-store`. see [flag precedence](flags.md).

the store answers how much light survives from each existing lamp.
radiance cascades address a broader problem, including area sources
inside the medium and higher scattering orders. the two-dimensional
cascade experiment lives in `improv/cascades`.

## The lamp store

`gpu/reach.go` says the design beside the code. In short:

- For each lamp and each boxed medium, a grid of cells over the
  medium's box, 128 along its longest side. A cell holds the share of
  that lamp's light, each channel, that reaches the cell's middle past
  every thing and through every volume: the kernel's own `blocked` and
  `clear`, asked once a cell and not once a step.
- For a narrow beam, a table of 1,024 places along its axis in place of
  a grid. A beam a metre wide is narrower than any cell. Its profile
  across is still worked exactly at each step.
- It keeps what arrives, not what is scattered toward the eye. So it
  does not depend on the camera or on which ray of a pixel is asking,
  and the phase function is applied at the step from the lamp's true
  direction.
- It is baked on the device at the start of each frame that encodes the
  world, by two kernels that share the trace's buffers. A frame that
  only moves the camera (`gpu.Frames`) draws from the store already on
  the device and bakes nothing.

the store lives on the device. in the cloud-sea control, its grids
and beam tables used 16.6 MB. the 12.5 m cells softened gaps and
shadow edges on that scale. thin occluders can cast faint shadows
or vanish from the store. the separate surface and gas paths keep
their own lighting queries, and so do a giant and a body's air and
clouds: each is a shell far thinner than a cell over its box, and lit
from the store a giant's disc showed the cells as fine ripples.

## The sky's sixteen looks

The exact tracer's `skylit` in `gpu/trace.metal` sends sixteen shadow rays
round the normal at every surface point of every camera sample.

The shortcut `sky-looks`, under `fastlight`, shares the sixteen among a
pixel's samples: at four rays a pixel each takes four, at sixteen each
takes one. At one ray a pixel nothing changes, and sky-lit shade is a
little rougher within a pixel. The exact picture still pays all sixteen at
every sample.
