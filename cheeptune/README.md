# Cheeptune

Cheeptune synthesizes a score through a voicing and exposes the resulting
piece as both audio and a musical clock. Slayer uses it for film soundtracks
and music-driven scenes. It offers two machines: a four-channel Game Boy
register simulation and a studio with FM, analogue, plucked, modal and
drum voices, chords, pan, reverb and delay.

start with [music in Slayer](../doc/music.md) to choose a tune,
add it to a job or synchronize the scene with it. these commands
list built-in music and check the library in the checkout.

```sh
game run slayer -- list tunes
game run slayer -- list voicings
game run slayer -- list library
game run slayer -- tunes cheeptune/library --check
```

## Formats and composition

- [SCORING.md](SCORING.md): score and voicing syntax and bounds.
- [SOUNDS.md](SOUNDS.md): how studio voices and effects change sound.
- [VIDEO.md](VIDEO.md): musical structure and cut sheets for a film.
- [DESIGN.md](DESIGN.md): package and clock contracts.
- [examples](examples/): scores and cut sheets.
- [library](library/): video tunes with their metadata.

## Embedding in Go

```go
piece, err := cheeptune.Open("@crypt", "@crypt-dmg")
if err != nil {
    return err
}
wav, err := piece.Cook(cacheDirectory)
if err != nil {
    return err
}
// Pass wav to the application’s audio player.
```

use the piece’s timing through `cuesheet.Piece`. a played Slayer job
connects it to the scene. `OpenTune` renders an embedded library tune
with the voicing in its cut sheet and returns both.
