# cloudsea

The cloud sea, corrected, and a walk through it: short vertical films,
made with `improv/walk`, that tell a stranger which parts of one picture
are measurements of the real world and which were made up, from the
picture's own records.

The scene is an observatory enclosure on a wooded summit above
a deck of cloud, four laser beams rising from it, on measured ground,
under a moon. It was drawn on 8 November 2026 from
`feat/griffith-observatory` at 97de8644, which this tree holds. Its
folders on the renders page are read only; nothing here changes them.

## What is here

- `cloudsea.yaml`: the study's `scene.yaml`, corrected. Each change has its
  comment; in short:
  - the guide stars: four beams on one aim out of the enclosure's open
    slit, at its authored 55 degrees, spread by GALACSI's 64 arc seconds
    (`eso-galacsi-layout`), 30 cm across (`eso-4lgsf`), at sodium's
    589 nm line.
  - each from a launch tube on the slit's rim, the rim pose
    `doc/golden-hour-site.yaml`'s turned with the enclosure. The study's
    four splayed aims and their origins on its skin are gone.
  - the enclosure's `heading`, 145 degrees, so its slit faces the moon's
    side. The mist is the study's, unchanged.
  - every complexity at `demo`, the finest, for pictures that ship.
  - the study's other framings (`close`, `overhead`, `south east
    overlook`), the chosen views, and the walkthrough's.
  - the deck says no `asymmetry`. A single forward lobe is what droplets
    are drawn with (`jendersie-deon-2023`), but the tracer scatters once,
    and with the moon behind the camera a forward lobe darkens the sea
    that many scatterings would keep bright.
- `scene_test.go`: holds the guide stars and the study's cameras to that.
- `places.csv`: the named things in hillside's table of where the
  scene's things stand, from a copy drawn 32 pixels across on the CPU.
  Walk takes one place a name, so the trees' rows are left out.
- `walkthrough.yaml`: the script, in `improv/walk`'s format,
  every number with its source beside it. The sentences are drafts, to
  be rewritten.
- `shorts/`: the command that writes them, each over glass hour
  through `@night-pad` from the bar its `sound` names (cheeptune
  renders it; walk.SongSpan cuts and fades it).

## What is still not right

- A beam is a thin thread with no glow round it: there is no multiple
  scattering, so the halo a beam has in fog is not drawn. A thin
  forward-scattering mist of its own (0.002 per m, asymmetry 0.85,
  inside the deck) adds nothing measurable: the beams spend only tens
  of metres in the deck. A halo needs multiple scattering.
- The 1.6 km square of ground ends in a cliff, which shows from high or
  far cameras. Widening `terrain.across` and `down` is a setting (the
  model runs about 12 km west and 10 km north of the window); the
  stands and the mist follow the window, so the tree counts would rise
  with it. A fade at the ground's edge is code: no key says one.
- The air above the deck thins over the deck's own depth (a smoothstep),
  not by a scale height as real air does.
- A real beam reaches the sodium layer near 90 km (`eso-4lgsf`); these
  stop at 1 km.

## The host

The walkthrough's sentences are zappy's (`specs/zappy-the-teacher.md` in
the team's folder): first person, plain, and it says what is false about
the picture before anyone asks. Zappy is pointing at someone else's
picture here, so it says so, and says whose.
