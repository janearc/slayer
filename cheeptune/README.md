# Cheeptune

Cheeptune synthesizes a score through a voicing and exposes the resulting
piece as both audio and a musical clock. Slayer uses it for film soundtracks
and music-driven scenes. It offers two machines: a four-channel Game Boy
register simulation and a studio with FM, analogue, plucked, modal and
drum voices, chords, pan, reverb and delay.

Start with [Music in Slayer](../doc/music.md) for choosing a tune,
adding it to a job, auditioning and synchronizing lights. The old standalone
`cmd/tune` examples are superseded by the Slayer front door:

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
- [examples](examples/): small supplied scores and voicings.
- [library](library/): video tunes with their metadata.

## Embedding in Go

```go
piece, err := cheeptune.Open("@crypt", "@crypt-dmg")
// Handle err before using piece.
wav, err := piece.Cook(cacheDirectory)
// Handle err, then send wav to your audio player.
```

An application can use the piece's timing with `cuesheet.Piece`; a played
Slayer job does that connection for you. `OpenTune` additionally reads a
library tune's cut sheet. Cooking audio does not start playback or acquire
a renderer. Read the score/voicing error rather than assuming every
machine expresses slides and dynamics identically.
