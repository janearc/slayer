# cuesheet

Plays a scene's channels from music. See `doc.go` for the sheet's format.

## Tests with no anti-test, and why

None. Every test that says something works has a failing case beside it,
named by a `// anti-test of TestX` line.

## Tests that draw

None. Every test is quick: no GPU, no shelf, no staged scene. Four tests
read `../cheeptune/examples/sunrun.json` and render it as a piece (about
0.4 s each, the first of them the slowest); the rest are arithmetic. The
suite is about 2.3 s with `GOWORK=off GOMAXPROCS=3 go test -count=1 .`.
