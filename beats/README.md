# Beats

A song's editable beat grid, in seconds from its original audio. The
grid feeds `cuesheet.Measured`; changing the tempo between two beats
changes the clock there, without stretching the audio.

`game run beats -- analyse song.opus --out score.yaml` makes a candidate.
It stays `status: estimated`. `--tempo` supplies a target BPM, `--meter`
an authored bar length (four unsaid), and `--first` discards estimated
beats before the chosen bar-one moment. Fine correction is in the YAML.
An existing score is never overwritten.

For a retained drum stem, add `--drums drums.wav`. The score stays bound
to the original song for playback and records the analysis stem's path
and hash separately. Both decodes must retain the same sample rate and
sample count; a shortened stem is refused rather than shifted silently.
The analyser does not itself perform learned source separation.

The detector uses positive log-energy changes in ten-millisecond blocks,
then autocorrelation and an onset/spacing path search. The search follows
equations 2-4 of Daniel P. W. Ellis's 2007
[Beat Tracking by Dynamic Programming](https://www.ee.columbia.edu/~dpwe/pubs/Ellis07-beattrack.pdf),
read at the author's page on 2026-11-14. Our energy front end is simpler
than his spectral front end. No implementation was copied.

The search band, continuity weight and onset display threshold are our
chosen settings, not his calibrated results. Full mixes can yield the
wrong tempo, its half or double, or the wrong downbeat. A supplied drum
stem is preferable when available; a transient is not proof of a beat.
The first beat's bar-one assignment is provisional until checked.

Moonlake is the stronger worked pattern: `improv/moonlake/README.md`
describes drum-stem tracking, measured versus counted-through beats and
the bar-one check. Its retained `make_score.py` was read in full for this
work. The quick full-mix pass is a diagnostic, not a replacement for that
workflow or a claim that complex percussion is solved.

Every beat has a `measured` mark: one for local onset support, zero for
counted-through time. This does not mean human verification. The visual
metronome labels an estimated grid accordingly. It currently visualizes
four-beat bars; another meter is refused rather than drawn incorrectly.

The score carries the audio file's SHA256. A job supplied with this
score refuses different audio bytes, even when the path is unchanged.
Both the sound and the grid already participate in a played job's image
cache identity. No global analysis cache is added.

Use `sound: {file: song.opus, score: score.yaml}` with a named scene's
`play` cue sheet. With no score, the file is estimated once when opened;
keep and check a score for repeatable editing. The only provided note
voice is `beat`; no instrument transcription is claimed.

`sound.from` trims the audio without renumbering the song's bars. Music
cues run on playback time, separately from `scene-duration`: an hour of
flight in thirty seconds plays thirty seconds of music. Cue-sheet bar
numbers refer to the original song, including after an audio trim.

Check before a long render:

```sh
game run metronome -- draw --score score.yaml --cues cues.yaml \
  --fps 24 --width 640 --height 360
```

This writes raw RGB24 frames, to mux with the same original audio. A
cue sheet containing `cues: []` checks beats alone. Verify pulses, tempo,
the bar-one phase and quiet breaks by eye and ear before setting the
score's status to `verified`. The old positional metronome invocation
is replaced by these Pzm flags.

`game run check-beats` runs small CPU checks, without a GPU or a scene.
