# Example jobs

these jobs use inputs in the checkout. inspect the planned files
with `--dry-run` before drawing. on a shared machine use the
[Azazel queue](../../azazel.md#submit-work).

    bin/slayer draw doc/examples/jobs/first-picture.yaml --dry-run

- `first-picture.yaml`, `smoke-still.yaml`: a still of smoke.
- `first-film.yaml`: an 18-frame film with music rendered from a
  tune. it writes PNG frames and an MP4. MP4 encoding requires macOS
  with cgo.
- `lantern-portrait.yaml`, `orbital-portrait.yaml`: named scenes.
- `city-*.yaml`, `macrostructures.yaml`: sets drawn as stills.
- `alfa/`: a submarine underway by moonlight, planned frame by frame
  with `cmd/cavitation-clip`; its README says how.

Jobs that need data slayer does not ship, such as terrain models or
star catalogues, are not here.
