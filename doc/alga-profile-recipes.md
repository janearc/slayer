# An alga from a profile book

An alga recipe reads one checked kelp book relative to its set:

```yaml
scene:
  recipes:
    kelp:
      alga:
        family: kelp
        file: ../models/kelp.yaml
        species: macrocystis-pyrifera
        condition: moderate
        forest: true
```

`family`, `file` and `species` are required; `kelp` is the supported family.
The species must match the book. `condition` selects a named water condition
and defaults to the book's moderate water; `forest` requests the authored
stand instead of one plant and defaults to false. The example book is
`manufacture/algae/kelp/kelp.yaml`. It holds one species, its dimensions,
looks, shared motion choices, water conditions and provenance.

The checked values and exact source bytes are retained by the recipe and job.
A changed book changes the render keys, while a job already read keeps its
snapshot after the file is changed or removed. Invalid or missing files and
unknown species or water conditions are refused. Old kelp factory-kind names
are not aliases; catalogue cards use profile recipes.

In Go, `manufacture.ReadKelpProfiles(path, bytes)` returns a checked book;
`book.NewKelp(options)` selects the species and water, and
`book.NewGiantKelp(condition, forest)` is the named convenience constructor.
Kelp is an alga; its current scaffold is not a generic model of all algae.
