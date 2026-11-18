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

The home is `SLAYER_CACHE`, an absolute folder, and nothing else. Every
cache slayer keeps is under it. Where it is unset or not absolute,
whatever needs a cache refuses by name; nothing is written anywhere on a
guess. Set it in your game config, or in your shell.

Each kind has a folder of its own under the home. One file is one entry,
written beside its place and renamed into it, so a reader sees the old
entry, the new one or none. An entry that does not check (its first line,
its record, its length, its sha256) is a miss and is never served.

## What is kept

| Kind | Holds | Keyed on | Cap |
|---|---|---|---|
| `skies` | the moonlit and twilit sky tables | the build, the height, the direction, the light and the exposure | 64 MiB |
| `ground` | a set's elevation window, cut from the model and smoothed | the build, the model's pinned sha256, the window, its step and the smoothing | 2 GiB |
| `air-fields` | a named air's light at a set's ground, by wavelength (package airlight) | the air's name, its columns and sun at every wavelength, the ground, the wavelengths, the grid, the photons, the seed, the workers and the solver's version, not the build | 512 MiB |
| `pin-checks` | the sha256 a pinned file was found to have | its path, size, modified time, inode and device | 4 MiB |
| `pictures` | a set's finished still | the job's picture key, the build and the nearest `sources.yaml` | 4 GiB |
| `trees` | the GPU's trees: a file each with `SLAYER_TREE_STORE=files`, or one bbolt file, `trees.bolt`, otherwise | the hash of the running program and of the mesh's faces | 8 GiB, for either way |

Over its cap a kind removes its least recently used entries, never one
used in the last ten minutes. A `.part` file older than an hour is a
killed writer's and is removed.

Also kept, outside `slayer cache`:

- Measured scene workload profiles use `workload-profiles.bolt` in the cache
  home. `render/workload` stores source/build/device/render/probe-keyed evidence
  with independent physical footprint, lifetime peak, RSS and native/scratch
  measurements. It holds database locks only during storage operations.
  This API does not by itself collect runtime probes or admit jobs; see
  [the contract](../render/workload/README.md).

- The GPU's tree cache is one bbolt file, `trees.bolt` in the cache
  home, beside the `trees` folder. `SLAYER_TREE_STORE` picks how
  they are kept: `bolt` (unset), `files` (the `trees` kind) or `pids`
  (each run writes its own file and merges into the main one at its
  end). Another value is refused by name.

  What it holds. One entry for each tree: the key is a sha256 of the
  running program's own sha256 and of the mesh's faces in order, and the
  value is the tree packed, its own sha256 first (about 588 MB for
  moonlake's). A second bucket, `used`, holds when each tree was last
  used. A tree is kept only for a mesh of 4096 faces or more.

  Why it grows. The key holds the program, so every build that draws a
  scene grows its trees again and keeps them beside the last build's,
  which no later build ever asks for.

  The bound. The file is kept to 8 GiB on disc, as the trees kind is: 8 GiB
  holds a dozen of moonlake's scenes' trees for one program.

  At the end of a run (`reportTrees`), the run writes the use of the trees it
  took (one write, and only for a tree whose recorded use is none or more
  than an hour old), and then, if the file is over 8 GiB, drops the least
  recently used trees until the rest is 6 GiB, never a tree used in the last
  ten minutes, and writes the file fresh.

  Trees with no recorded use (every tree an older build kept) go first. The
  2 GiB between 6 and 8 is so one run's new trees do not rewrite the file
  again.

  Giving the disc back. bbolt never returns pages to the filesystem;
  deleting an entry leaves the file as large. So the rewrite writes a
  new file, `trees.bolt.compacting`, beside the old one, in
  64 MiB transactions, and renames it over `trees.bolt`.

  It does this holding the old file's lock, the one every reader and writer
  of the file takes (readers shared, writers alone), so nobody is inside the
  old file when it is swapped. A process that was waiting for that lock
  holds the old file, now unlinked, so `treebolt.Open` looks at the name
  again once it has the lock and opens the new file if it is not the same
  one; a tree stored to an unlinked file would be lost.

  It is safe while another process reads: that process holds the lock, the
  rewrite waits for it (up to six minutes, as every process does), and one
  that comes after waits for the rewrite, which takes about a minute for
  8 GiB. While it runs the disc holds both files, up to twice the cap.

  The file may be over its cap between a run's last store and its end, by
  that run's trees. A file under the cap is not opened by the trim at all.

  A cache only changes speed: a tree it cannot read or has lost is grown,
  and the picture is the same to the byte.

  The first run of this build on a file an older build made has no
  recorded use for any tree, so if the file is over 8 GiB it drops all
  of them but those that run took or stored, and the file becomes about
  that run's trees.

  `slayer cache` lists it as `trees (one file)`, beside the `trees` folder,
  and `slayer cache --clear trees` (or `--clear` with no kind) clears both:
  they are one kind kept two ways, so the two are said and cleared under
  one name, and the file's clear is a rewrite of nothing, which gives the
  disc back. `--dry-run` says how many entries it would remove.

  `SLAYER_TREE_STORE` picks how trees are kept: `bolt` (unset), `files` or
  `pids` (each run writes its own file and merges into the main one at its
  end, which the bound is kept to as for `bolt`).
- The weather's moments, in `weather/` in the cache home, shared by
  hillside.
- A film's frames are working files in `frames/TEAM/NAME-DIGEST` under
  the cache home, with a `manifest.json`; `SLAYER_TEAM` names the team;
  with none, `frames/NAME-DIGEST`. They are removed when the movie is
  written, unless `--keep-frames`.
- The smokes' results, keyed by the binary; `slayer assess --smokes
  --fresh` draws them again.

A still is kept only when it is a set's still. A job that is not one is
neither read from nor written to the `pictures` cache. The set's own files
(terrain, assets) are in the pin checks but not in the frames key of a
film.

## Look

```sh
slayer cache
```

says, for each kind, its folder, its entries, their bytes against the cap,
and when its least and most recently used entry was last used. The tree
file is a second line under the trees kind, `trees (one file)`, with its
bytes on disc and live, how many trees have no recorded use, and its path.

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
match is a miss, with two known gaps:

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
