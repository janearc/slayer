# Slayer disk cache

Slayer reuses derived data under the absolute path in `SLAYER_CACHE`.
The cache packages choose separate kinds and limits. Each kind is declared
with `MustKind` and opens its own folder.

```sh
slayer cache
slayer cache --clear trees --dry-run
```

The first command shows counts, bytes, caps and age for each kind. The
second previews what clearing the tree cache would remove. Without
`--dry-run`, `--clear KIND` clears that kind and `--clear` clears all
kinds. The `trees` report includes both individual entries and the bbolt
file.

## Cache identity

`NewKey(kind)` builds a key from named text, byte content and digests.
Callers supply every input on which the result depends, including build
identity where appropriate. `Build` hashes the running executable.
The filename is the SHA-256 of the complete key record.

`Store.Put` writes to a unique temporary file beside the destination and
renames it into place. `Store.Get` checks the version, exact key record,
payload length and SHA-256. An invalid entry is a miss. Valid entries
record their latest use to guide eviction.

Each kind trims its least recently used entries after a write, respecting
the grace period for recently used entries. A store can exceed its cap
while those entries are protected. Trimming also removes stale partial
files. The tree cache's bbolt store has its own locking and compaction.
