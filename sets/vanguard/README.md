# Borrowed Light: opening light studies

These are synthetic, deliberately small scene studies for the opening and
return of the Vanguard reel. They are not accepted hero frames. The set
coordinates are metres east, north, up; the camera, light, dust and
materials are artistic choices, not measured values.

`borrowed-light-opening.yaml` is the first narrow-spot diagnostic, with a
moving sphere and a floor receiver. `borrowed-light-anchor.yaml` moves the
receiver to the editorial highlight target and uses a wider cone. The numbered
moments are separate still jobs at 0, 1 and 2 seconds. The cone is an inverse
square point spotlight, not a Gaussian beam. Both scenes use isotropic
single-scattering painted dust. The dust is not a measured aerosol.

At this revision both spot studies show visible beads, stripes and repeated
shadow forms, because a narrow spot is sampled by the volume march's fixed
spatial steps. The second scene changes the light footprint and the subject;
it is not a controlled comparison of sampling algorithms. Neither is accepted.

To keep images out of the source tree, copy a set and its jobs together to a
new folder under the shared renders home and draw the copied job through the
Slayer command. For example, from the Slayer checkout, after building the
`bin/slayer` command with Game:

```sh
bin/slayer draw /path/to/render-folder/anchor-before.yaml --fresh --renderer cpu
```

The jobs request a 360 by 320 picture, one ray per pixel, and a matching raw
PFM next to it. `anchor-interrupted.yaml` and `anchor-after.yaml` use the same
scene at one and two seconds. Record the build revision, command, output hashes
and limitations with every new render. This is a framing and transport study,
not proof of the final camera, photometry, film, visual quality or motion blur.
