# diskcache

The one place Slayer keeps what it worked out before, so the next run
reads it instead of working it out again. Everything kept here is
exact: a hit is byte for byte what a fresh computation gives, and a
change of one byte in anything it was made from is a miss.

## Where

`Home` is the folder every kind lives under, in this order:

1. `SLAYER_CACHE`, an absolute folder, when it is set (as
   `SLAYER_TREE_CACHE` names the tree cache's file).
2. `cache/` under the renders home, `SLAYER_RENDERS`, when that is set.
3. `slayer/` under the user's cache folder (`~/Library/Caches` on a
   Mac).

Each kind of thing kept has a folder of its own under the home, named
for the kind. A kind is made by `NewKind` with its name, its cap in
bytes and the reason for the cap; the packages that keep things declare
their kinds, and `Kinds` lists every kind declared, for the verb.

## Keys

A key is made of named parts: `NewKey(kind)`, then `Text(name,
value)` for a word or a number, `Bytes(name, value)` for anything
already in memory, and `Digest(name, sha256)` for a file whose sha256 is
already known. Each part is kept as a line of its record, `name value`,
with bytes as their sha256; a part may be said only once. The key's
name is the sha256 of the record, in hex.

So a miss can be explained: the record says what went in, and two
records side by side say what changed. The build is a part like any
other; `Build()` is the sha256 of the running program, worked out once
a process.

What goes into a key is the caller's to choose, and the rule is the
content of exactly what the result depends on. Never a file's name or
its time.

## An entry

One file a key, `KIND/AB/NAME`, AB the name's first two hex digits so
no folder holds too many. It holds, in order: a first line saying what
it is and its version, the record, when it was written, the sha256 of
the payload and its length, and the payload.

`Store.Put(key, payload)` writes the whole entry beside its place, as a
`.part` file with a name no other writer has, and renames it into
place. A rename is whole: a reader sees the old entry, the new one or
none, never half of one. Two writers of one key write the same bytes,
since the payload is exact, so whichever rename lands last is the
same.

`Store.Get(key)` reads the whole entry under one open handle and checks
it: its first line, its record against the key's (so two keys whose
names collided would still be told apart), its length and the sha256
of its payload. Anything that does not check is a miss and is said as
one; it is never served. A hit marks the entry used, by its time on
disk, so the least recently used goes first.

## Why a file an entry, and not bbolt

The tree cache (`gpu/treecache.go`) keeps its trees in one bbolt file.
bbolt takes a lock on the whole file: a reader a shared one, a writer
one that shuts out everyone.

The tree cache opens the file for each look and each store and waits
up to six minutes (`lockWait`) for its turn; a run that waits that long stops using the file and grows every
tree for the rest of its run, and says so only in its tally. With a
dozen renders at once, each storing trees, every reader queues behind
every writer.

The file also never shrinks (647 MB on 2026-11-12) and
has no cap.

A file an entry has none of that: a reader never waits for a writer,
since a rename is whole; two writers of one key do not wait for each
other; an entry is removed by itself, so a cap is a plain list sorted
by time; and the disk is fast enough that opening a file costs less
than a millisecond. So every kind here is files, and the tree cache's
trees could move here as a kind of their own.

## Caps

After a put, a kind over its cap removes its least recently used
entries until it is under. It never removes an entry used in the last
`heldFor`: a run that has just read one may be about to read it again.

An entry being read is safe in any case: its reader has the file open,
and removing a file another process holds open leaves that process its
bytes. `.part` files older than `staleParts` are a killed writer's and
go too.

## The verb

`slayer cache` says, for each kind, its folder, its entries, their
bytes against its cap, and its oldest and newest. `slayer cache --clear
[KIND]` removes every entry of every kind, or of one. These are
`Survey` and `Store.Clear` here; the verb is in `door`.

## What is not here

What each kind keeps, and what its key is made of, is the business of
the package that keeps it. This package knows nothing about scenes or
pictures.

## Tests

Every test is quick: it uses a temporary folder and a test clock, no
GPU, no shelf and no staged scene, and the suite runs in under half a
second.

Each test is either a positive or an anti-test. An anti-test carries
`// anti-test of TestX` on its own line, naming the positive it answers
(a function that answers two carries two lines). Every positive is
named by at least one such line. To list them:

    grep -h '// anti-test of' diskcache/*_test.go | sort | uniq -c

### Tests with no anti-test, and why

None. Every positive here has a failing case of the same check: a key,
a path, a header or a declaration that must be refused or told apart.
TestDecodedIsWhatEncodedWrote and the other tests of a value read back
are answered by the refusals of a spoilt copy of that value.
