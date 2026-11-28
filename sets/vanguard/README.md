# borrowed light

these small Vanguard scenes study a spotlight crossing a receiver and
painted dust. the cone is an inverse-square point spotlight. the dust is
an authored single-scattering volume with isotropic scattering.

- [opening study](borrowed-light-opening.yaml) uses a narrow spot, a
  moving sphere and a floor receiver.
- [anchor study](borrowed-light-anchor.yaml) places the receiver at the
  highlight and widens the cone. its jobs sample the scene at
  [zero](anchor-before.yaml), [one](anchor-interrupted.yaml) and
  [two seconds](anchor-after.yaml).

from the checkout root, check the first anchor job. to draw it, copy the
job and its set together so the outputs go beside the copies:

```sh
game run slayer -- assess sets/vanguard/anchor-before.yaml
mkdir -p out/vanguard
cp sets/vanguard/anchor-before.yaml sets/vanguard/borrowed-light-anchor.yaml out/vanguard/
game run slayer -- draw out/vanguard/anchor-before.yaml
```

the jobs request 360 by 320 pixels at one ray per pixel and retain a PFM
raw alongside the developed image. the narrow spot can show beads or
stripes where it intersects the volume march’s fixed steps. the two sets
change the light footprint and the receiver, so use a fixed set when
comparing sampling algorithms.
