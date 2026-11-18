# Fast light

Fast light is an opt-in way to draw a scene with mist or smoke in less
time, for pictures that are for watching and not for measuring. It is a
device path. The CPU tracer stays exact and refuses it.

## How it is asked for

Three keys, in a job and, the first two, under a set's `scene`:

    vibes: on
    fastlight: on
    shortcuts:
      lamp-store: off

`job/README.md`, "flags and shortcuts", is the contract. In short: a
shortcut is one named approximation; `fastlight` is the flag for the
light shortcuts; `vibes` is the flag for every flag. A shortcut is
decided by the first that says anything: its own word, then `fastlight`,
then `vibes`, each from the command line, then the job, then the set.

With nothing said the picture is the exact tracer's, byte for byte what
it was before these keys existed.

There are two light shortcuts today: `lamp-store`, and `sky-looks`, which
needs it. The last section says what `sky-looks` does.

A picture drawn with it says so. The job's canonical form, and so its
picture key and its frames ledger's key, hold the flags as said; the
copy of the job beside the frames says them; hillside prints a `fast
light:` line; `gpu.Profile.LampStoreBytes` is not nought.

Refused by name: on the CPU; through a rig's lens; for a named scene;
beside a proof (`-prove`), which replays pixels on the exact tracer;
beside `volume-lighting: cached`; for a world with more than eight
lamps, where the exact tracer picks among them at each point; and
`sky-looks` without `lamp-store`.

## The method, and why not cascades

Radiance cascades (Osborne and Sannikov 2024, arXiv 2408.14425; the
two-dimensional proof is `improv/cascades`) solve for light arriving
from every direction at every place. They earn that where light comes
from the medium itself or from many places: fire lighting its own smoke
from within, a glowing nebula, light scattered more than once.

The cloud sea has five lamps, each with one direction at any point. The
question at a step is not "what arrives from everywhere" but "how much
of this one lamp gets here", five times. A store of that answer for each
lamp is smaller, simpler and nearly exact: its only error is how coarse
its cells are. It also keeps each lamp's direction, which a mist that
scatters forward needs and a store of summed light would lose.

So fast light, today, is a store for each lamp. Cascades would still
add:

- fire and embers as area sources lighting smoke, in place of the few
  point lamps a flame is cut into now;
- a gas that glows, such as a nebula, which the store does not touch;
- light scattered twice or more in thick mist or cloud, which nothing
  in the tree draws at all.

None of those is the cloud sea.

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

On the cloud sea the store is 16.6 MB (two media, one moon's grid in
each, five beam tables).

No store is kept on disk.

What it gets wrong: a shadow cast into the mist is as soft as a cell,
12.5 m in the cloud sea, so a shaft of moonlight between two trees is
blurred where the exact march shows it.

What it leaves alone: surfaces; a gas (`gas.Of`), lit by its own
`gas_gathered`; a puff, a wisp, a shell, a ring and a wake.

Surfaces do not read the store and should not be made to. A surface
point lies between cells whose middles are inside its own trunk or under
its own ground. Those hold no light, so the surface would darken in
patches that follow the grid, and a tree's shadow on the ground would be
12.5 m soft.

## The sky's sixteen looks

The exact tracer's `skylit` in `gpu/trace.metal` sends sixteen shadow rays
round the normal at every surface point of every camera sample.

The shortcut `sky-looks`, under `fastlight`, shares the sixteen among a
pixel's samples: at four rays a pixel each takes four, at sixteen each
takes one. At one ray a pixel nothing changes, and sky-lit shade is a
little rougher within a pixel. The exact picture still pays all sixteen at
every sample.
