# Caches

For in-memory reuse, retained GPU inputs, baked fields and compositor
plates, see [optimizations](optimizations.md). A light interpolation grid
is an approximation; a completed-result cache has an exact reuse contract.
These mechanisms have different invalidation rules.

Slayer keeps what it worked out before so the next run reads it. Every
entry is meant to be exact: a hit is byte for byte what a fresh
computation gives, and a change of one byte of anything it was made from
is a miss. Nothing approximate is cached. Two known gaps are under "Which
are exact".

## Where

The home is the first of:

1. `SLAYER_CACHE`, an absolute folder;
2. `cache/` under the renders home, `SLAYER_RENDERS`;
3. `slayer/` under the user's cache folder (`~/Library/Caches` on a Mac).

Each kind has a folder of its own under the home. One file is one entry,
written beside its place and renamed into it, so a reader sees the old
entry, the new one or none. An entry that does not check (its first line,
its record, its length, its sha256) is a miss and is never served.

## What is kept

| Kind | Holds | Keyed on | Cap |
|---|---|---|---|
| `skies` | the moonlit and twilit sky tables | the build, the height, the direction, the light and the exposure | 64 MiB |
| `ground` | a set's elevation window, cut from the model and smoothed | the build, the model's pinned sha256, the window, its step and the smoothing | 2 GiB |
| `pin-checks` | the sha256 a pinned file was found to have | its path, size, modified time, inode and device | 4 MiB |
| `pictures` | a set's finished still | the job's picture key, the build and the nearest `sources.yaml` | 4 GiB |
| `trees` | the GPU's trees, a file each, with `SLAYER_TREE_STORE=files` | the program the tree is grown from | 8 GiB |

Over its cap a kind removes its least recently used entries, never one
used in the last ten minutes. A `.part` file older than an hour is a
killed writer's and is removed.

Also kept, outside `slayer cache`:

- The GPU's tree cache is one bbolt file named by `SLAYER_TREE_CACHE`;
  unset, every run grows its own trees. `SLAYER_TREE_STORE` picks how
  they are kept: `bolt` (unset), `files` (the `trees` kind) or `pids`
  (each run writes its own file and merges into the main one at its
  end). Another value is refused by name.

  The file has no cap and has never shrunk (647 MB on 12 November 2026).
  Three ways were put in so they can be measured; none has been chosen.
- The weather's moments, in the weather cache (`SLAYER_WEATHER` or the
  user's cache), shared by hillside.
- A film's frames are working files in `cache/TEAM/NAME-DIGEST` under the
  renders home, with a `manifest.json`. They are removed when the movie
  is written, unless `--keep-frames`.
- The smokes' results, keyed by the binary; `slayer assess --smokes
  --fresh` draws them again.

A still is kept only when it is a set's still. A job that is not one is
neither read from nor written to the `pictures` cache. The set's own files
(terrain, assets) are in the pin checks but not in the frames key of a
film; see the changelog, "Found, not fixed".

## Look

```sh
slayer cache
```

says, for each kind, its folder, its entries, their bytes against the cap,
and when its least and most recently used entry was last used.

## Clear

```sh
slayer cache --clear            # every entry of every kind
slayer cache --clear skies      # one kind
slayer cache --clear --dry-run  # say what would go, remove nothing
```

An entry being written is left to its writer. To draw one still again
without clearing anything, `slayer draw JOB --fresh`.

## Which are exact

Each key names what its entry was made from, and an entry that does not
match is a miss, with two known gaps, both found in the review of 12
November:

- A kept still's key holds the job, the files the job names, the build
  and the shelf's sources list, but not the files a set reads in turn
  from beside itself (its hud theme, for one). So a kept still is served
  only where `SLAYER_KEPT_STILLS` is `on`; stills are kept either way.
- A pinned file (the elevation model) is hashed again only when its
  path, size, modification time, inode or device changed since the last
  run. A file replaced in place with all of those kept is taken as
  unchanged; `slayer cache --clear pin-checks` makes the next run hash
  it in full.

The fast light store is not cached on disk, and a picture drawn with fast
light says so in its record. A cache never makes a picture approximate.
