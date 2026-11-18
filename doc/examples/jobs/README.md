# Example jobs

One job of each kind, to read and to draw. Each reads only files in this
repository and writes under `out/` or the renders home:

    slayer draw doc/examples/jobs/first-picture.yaml

- `first-picture.yaml`, `smoke-still.yaml`: a still of smoke.
- `first-film.yaml`: a short film with music rendered from a tune.
- `lantern-portrait.yaml`, `orbital-portrait.yaml`: named scenes.
- `city-*.yaml`, `macrostructures.yaml`: sets drawn as stills.
- `alfa/`: a submarine underway by moonlight, planned frame by frame
  with `cmd/cavitation-clip`; its README says how.

Jobs that need data slayer does not ship, such as terrain models or
star catalogues, are not here.
