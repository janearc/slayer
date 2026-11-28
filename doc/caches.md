# Caches

set `SLAYER_CACHE` to an absolute path in your shell or Game
configuration. Slayer refuses commands that need the cache when the
path is missing or relative. a stored entry that fails validation is
computed again.

```sh
slayer cache
slayer cache --clear trees --dry-run
slayer cache --clear pin-checks
```

`cache` reports the paths, entry counts, bytes, caps and last-use
range of its stores. `--clear KIND` removes one kind. `--clear`
alone removes all listed kinds. add `--dry-run` to report the
entries without deleting them. use `slayer draw JOB --fresh` to
bypass a still’s saved image and rows on one run.

## What is kept

| Kind | Holds | Cap |
|---|---|---|
| `air-fields` | spectral ground light for a named atmosphere and sun-angle bins | 512 MiB |
| `ground` | elevation windows at the requested spacing and smoothing | 2 GiB |
| `pin-checks` | previously checked hashes of pinned files | 4 MiB |
| `pictures` | finished stills drawn from sets | 4 GiB |
| `rows` | completed rows of unfinished CPU set stills | 2 GiB |
| `skies` | moonlit and twilit sky tables | 64 MiB |
| `trees` | GPU acceleration trees keyed by mesh faces and executable | 8 GiB per store form |

the file-per-entry stores drop least recently used entries over the
cap, protecting entries used in the last ten minutes. a finished
entry is renamed into place, and a reader verifies its record, size
and hash before using it.

## GPU trees

`SLAYER_TREE_STORE` selects the storage layout:

- `bolt` (default) keeps trees in the shared `trees.bolt` database.
- `files` keeps one file per entry under `trees/`.
- `pids` keeps each process’s new trees in its own database and
  merges them into `trees.bolt` when the process exits normally.

the database can grow during a run. end-of-run compaction can hold
the old and new file at once and contend with other users for its
lock. `slayer cache --clear trees` clears both the file-per-entry
store and the shared database. a cache miss rebuilds the tree.

## Working files and reuse

- completed set stills are served from the picture cache when
  `SLAYER_KEPT_STILLS=on`. the cache key can miss files referenced
  by the set, such as a HUD theme. use `--fresh` after changing
  such inputs. saved rows have the same limitation.
- pin checks reuse a hash when the path, size, modification time,
  inode and device agree. clear `pin-checks` to force rehashing.
- a movie’s [working frames and resume manifest](../door/README.md#resume)
  live under the cache unless the job names a frame directory. working
  frames are removed after a successful encode unless `--keep-frames`
  is set.
- `slayer assess --smokes --fresh` ignores stored smoke verdicts.
- weather snapshots and the [workload profile database](../render/workload/README.md)
  have their own storage under the cache path.

[rendering optimizations](optimizations.md) covers in-memory buffers
and [lighting approximations](flags.md) covers light grids.
