# Slayer's tests in tiers, and anti-tests one to one

For practical setup and focused test commands, start with
[Testing Slayer](testing.md). Timing/count records below describe the
original tier migration; use the inventory tool for the current tree.

## Why

Most of Slayer's tests do not draw, but they used to run beside the ones
that do. The tree has about 6,230 tests. One `hillside` suite took 7 to 25
minutes and `gpu` about 4, so `go test ./...` passed the default 10 minute
timeout, and the Metal autorelease sentinel in `metal/pools_test.go` failed
when another package drew on the GPU at the same time. Tests are now in
tiers, and each tier is run by itself.

## The tiers

| Tier | Holds | Runs by | Measured |
|---|---|---|---|
| hot path | the non draw tests of the packages on Slayer's hot paths | `game run test-hot` | 4 s |
| non draw | every test that does not draw | plain `go test ./...`, and `game check` | 51 s, whole tree |
| draw | a test that opens a Metal device, traces a picture, stages a scene, reads the shelf, renders sound or a movie, or takes more than about 2 s | `game run test-draw` | before a merge; `-p 1`, up to an hour |
| all tests | non draw and draw together | `game run test-all` | before a release |

About 5,000 tests are non draw and 1,240 are draw. The non draw tier opens
no GPU device.

Mechanism: one build tag, `draw`. A test that draws is in a file named
`NAME_draw_test.go` whose first line is `//go:build draw`. A file that
already had a build line keeps it, as `//go:build draw && darwin`. Nine
packages draw as a whole: `gpu`, `metal`, `fuelbed`, `render/smokejudge`,
and omnipositor's `gpudevelop`, `gpulights`, `gpumatte`, `gpumist` and
`gpuocean` under `improv/moonlake/omnipositor`.

Every test file in them
carries the tag in place, and their plain build has no tests.

Helpers used by both tiers stay in untagged files; a helper used
only by drawing tests moves with them. The anti-test marker moves with its
test.

`-tags draw` adds files and cannot take the plain ones away, so
`test-draw` is `go run ./tools/tiers draw`: it lists each package both
ways and runs, with the tag, only the names the plain build lacks.
`test-all` is every test with the tag, one package at a time.

The hot path tier is a list of packages, not a tag: the non draw tests of
the packages in `tools/tiers/hot.txt`, each with a line saying why it is
there. `go run ./tools/tiers hot` runs them. Add a package by the hot path
tables (fan-in, and on the main drawing path), and keep the tier under a
minute.

## Where a test goes

A test is draw if it opens a GPU or Metal device, traces a picture, stages
a scene (`hillside.Open`), renders sound or a movie, reads `SLAYER_SHELF`,
or takes more than about 2 s. Otherwise it is non draw. 

A new test that
draws goes in a `_draw_test.go` file. If a test is in the wrong tier, the
non draw tier shows it: it runs slowly, or `metal/pools_test.go` fails.

Two Examples that draw stay in the plain build, `Example_assess` in `door`
and `Example_write` in `improv/pan/cmd/pan`: the packages check that every
verb has an Example in a file the plain build includes.

An anti-test sits in the same tier as the test it answers where it can.
Where the positive draws and its anti-test does not, both stay where they
are. At the first split that was 17 anti-tests, all in `door`.

## Proof that nothing was lost

`go run ./tools/tiers check` lists the tree twice with `go test -list`,
once plain and once with `-tags draw`, and compares both to the saved list
of every test name, `tools/tiers/before.txt`. It exits 1 if a name is lost,
added, in the plain list but not the draw list, or twice in one list.


`go run ./tools/tiers check ./hillside` does one package. After a test is
added or removed on purpose, `go run ./tools/tiers snapshot` saves the list
again. 

`go vet ./...` and `go vet -tags draw ./...` are clean, and the
counts of `tools/antitests` are the same before and after the split.

The placement of every test, with its time and the reason, is in the
renders home at `fable-team/reviews/test-tiers/placement.md`.

## Anti-tests, marked so they can be counted

Every anti-test is marked with a comment naming the test it answers, `// anti-test of TestX`, on the anti-test function
or on the anti part inside a test. A script then counts the ratio exactly
per package, and a gate can hold it at one to one or better. The audit of
12 November (`fable-team/reviews/test-audit/` in the team's renders home,
with its scripts) found about half the tree's tests with an anti part,
about 2,700 missing, about 370 of them in the packages new that day.

Existing markers (`// anti-test:`, `// anti:` and the rest) are rewritten
to name their test as each package is moved into its tiers, and the
missing anti-tests written in the same pass, failure cases first, since
they belong to the non draw tier.

## The pilot: diskcache

The first package done, 12 November, to set the pattern. Result:

| | before | after |
|---|---:|---:|
| tests | 58 | 66 |
| anti-test functions (marked) | 0 | 43 (and 1 inline) |
| positives (tests that are not anti-tests) | 23 | 23 |
| positives named by a marker | 0 | 23 of 23 |
| marker lines | 0 | 45 |

Eight anti-tests were written, one existing test was reshaped (its
anti part made able to fail), and 35 existing tests were marked. The suite is 0.2 to 0.4 s (`go test -count=1
./diskcache/`, GOMAXPROCS=3), all non draw; it imports nothing that draws,
so it needs no tag. No test needed a move.

### How a test was classified

Read the test, then ask which of two it is.
- An anti-test: its point is that something is refused, missed, told
  apart or left alone (`TestKeyRefusesANameOfTwoWords`,
  `TestGetMissesATornEntry`). It needs no anti-test of its own.
- A positive: its point is that something works. It needs at least one
  anti-test, which is the failing case of the same check.
The audit's list of "none" is mixed: of the 28 listed, 10 were
anti-tests with no marker and 18 were positives. Of those 18, 11 already
had an anti-test in the file (a `Refuses` or `Misses` test beside them)
that the classifier did not tie to them; 7 had none and got a new one.

Two positives the audit had not listed (`TestKeyKeepsTheFirstRefusal`,
`TestKeyIsNotChangedInPlace`) were found to need one too. Read the list
as "unmarked", not as "missing".

### How a test is marked

A line of its own, exactly `// anti-test of TestX`, before the anti-test
function (as its doc comment, with any prose on the lines after it). One
anti-test that answers two positives carries two lines. A positive may
have many anti-tests. Where the anti part sits inside a test, the same
line goes on its own line before that part and names the test it is in
(`TestKeyIsNotChangedInPlace`).

Prose after the name on the same line
is not allowed, so the line can be counted by pattern. No package here
had an old marker (`// anti-test:`), so none was rewritten.

Count: a test is a positive unless a marker is above its function; the
ratio is positives named by a marker over positives. Here 23 of 23.

### How an anti-test was written

Same call, same check, broken input. The new ones:
- a key made twice from the same parts has one name (answers "one byte
  changes the name");
- the same parts in another order are another key;
- the first refusal is kept whichever refusal is first;
- two keys made from one base differ, with a base that has spare room in
  its list, since with one part the sharing cannot show;
- a build that cannot be found is refused, by Build and by Built (the
  package's `build` variable is swapped for a failing one and put back);
- a refused declaration is not listed;
- writers of two different payloads: a reader sees one whole payload or
  no entry, never a mixture;
- a .part file is not counted in the usage;
- clearing one kind leaves another kind's entries.

### Proof each can fail

Every new anti-test, and the inline one, was run against a deliberate
break of the code it guards, failed, and the code was put back:

- a name that changes each call;
- a record that sorts its parts;
- a refusal that is overwritten;
- a part list shared between keys (the three-index slice taken out);
- Build and Built each swallowing the error (two breaks, one for each);
- a declaration listed before it is checked;
- a write straight to the entry's place with no rename (failed at count
  5 with "not an entry of this version");
- a part counted as an entry;
- Clear walking the home instead of the kind's folder.

Marked old tests were not broken one by one: they are refusals that were
already there.

### Tests with no possible anti-test

None in this package. A test that a constant equals its documented value
would be listed in the package's README under "Tests with no anti-test,
and why", with the one line saying why, and would carry no marker. Try
first: nearly every such test has a neighbour (the value one off).

### What to change before the others follow

- Say in the instructions that the audit's "none" list is mostly
  unmarked anti-tests and positives whose anti-test is already beside
  them: mark first, write second. Otherwise an agent writes 28 tests
  where 8 were needed.
- Define the count (positives named over positives) and give the grep.
  The audit script's pattern matches `anti-test` followed by anything;
  the gate should match `// anti-test of Test\w+` exactly and check the
  name exists in the package.
- Allow one function to carry two marker lines, and many anti-tests to
  name one positive; say so, or agents will duplicate tests to get 1:1.
- Tell them to look for a swappable seam (a package variable holding a
  function) before calling an error path untestable.
- A concurrency anti-test should be run at count 5 under its break; one
  run can pass by luck.
- The marker sits on its own line, never inside a condition.
- The proof by breaking is slow by hand (twelve edits). A small tool
  that takes a file, an old string and a new one, runs one test and
  restores the file would make it routine; the one used here is a
  thirty-line script and could be kept in `tools/`.

## Tools

- `go run ./tools/tiers check|snapshot|hot` proves the tiers hold every
  test once, saves the list of names, and runs the hot path tier (see
  above).
- `go run ./tools/antitests [-v] [-min RATIO] PKG...` counts, per
  package, the tests, anti-tests (a marker directly above the function),
  positives and positives named by a marker, and the ratio. A marker
  naming a test that does not exist is an error (exit 1), as is a
  package below `-min`. Old markers (`// anti-test:`, `// anti:`) are
  counted and listed by file:line. `-v` lists unanswered positives.
- `go run ./tools/antiproof -file F -old OLD -new NEW -run REGEX [-count N]
  PKG` replaces one exact occurrence of OLD with NEW, runs
  `go test -count=N -run REGEX PKG`, and puts the file back byte for byte
  on every path, an interrupt included.

  Exit 0 if the tests failed under
  the break (the proof holds), 1 if they passed, 2 if the tool could not
  do its work (OLD not found once, the break does not build, no test
  matched), 3 if the file could not be put back.
