# moonlake (improv)

Kīlauea iki flooded to 1076 m, seen from 20 m up on the rise at its
west end through close spruces, toward a giant moon over the far rim:
oceantex's water, mist, lasers, strobes, searchlights, a ship's beam and
fires, played as instruments in time with the song moon-lake. Made on
2026-11-09 and 10: "it's going to look like elden ring. that's
the point."

The renders, with a README each, are kept outside this repository, under
`renders/2026-11-1?-moonlake-*`: the whole film at 2K is
`renders/2026-11-10-moonlake-2k`.

## the pieces

- `sets/moonlake.yaml`: the kīlauea iki moonrise set, the camera on the
  west rise, three close spruces 7 to 12 m ahead of it, eight more on
  the rim, the sun off opposition, an 85 mm lens on a 16:9 crop of the
  15/70 frame, no fires.
- `hillside`: improv flags, scene content that a set also says under
  its `fantasy` block (`sets/moonlake.yaml` holds the lake picture's
  flags as a commented example of it; the block is the file form of
  those flags, and a flag given on the command line stands over its key):
  - `-lake ELEVATION` still water (black on the GPU: see below)
  - `-lakeglow` the water shining on its own and nothing else
  - `-moon SCALE` the moon's disc this many times its size
  - `-moonglow F` the disc's light times F
  - `-giants TIMES` the rim's named trees grown this many times taller
  - `-close TIMES` the close spruces the same. Grown, not magnified:
    the trunk taller and thicker with more whorls, every branch and
    needle its own size (`manufacture.Taller`), and the grown branches
    carrying side branches of their own (`manufacture.Branched`)
  - `-closeglow` the close spruces glowing alone, for their mask
- `improv/moonlake/`, each its own `package main`:
  - `seawater`: oceantex's water from moonlake's camera, the moon as its
    sun, the scene's raw as the sky it reflects
  - `lightshow`: strobes behind the rim lighting the haze, the rim's
    trees cutting shafts; a ten-beam laser fan swept by mirrors, glowing
    in air and brighter in mist
  - `cueshow`: the cue compiler. From `score.yaml` (the track's measured
    beat grid and transcribed notes; "the score", below, says how it
    was pulled from the audio) and `cues.yaml` (what the lights do,
    written in bars), it says the moment's state as key=value lines.

    `CUES=cue.env lightshow ...` obeys them: rig level and blackout,
    beam colour per beat, beam count, bar-locked mirror sweeps, strobes
    gated to the grid or to the acid line's loud notes. Without `CUES`,
    lightshow free-runs as before. Editing `cues.yaml` is how the show
    changes; nothing recompiles.
  - `metronome`: the score checked by eye against the song. Raw frames
    to ffmpeg with the song under them: a square a beat, the score's
    meter of them (four when unsaid), flashes on each measured beat
    (beat one white), the bar and beat written large, a
    word for measured or counted-through, and a band for each cue as it
    fires.

    `game run metronome -- draw --score score.yaml --cues cues.yaml
    --fps 30 --width 960 --height 540 | ffmpeg -f rawvideo -pix_fmt rgb24
    -s 960x540 -r 30 -i - -i moon-lake.m4a ...`
  - `seamatte`: lays the water in below the far shore, then what stands
    in front of it (`FRONT`)
  - `mistlake`: mist on the lake, curling with time
  - `mirrorlake`: the first, plain mirrored lake, kept for comparison
  - `pfmstat`: where a raw's light is

`MOONREF=plate.pfm` keeps seawater, seamatte and mistlake taking the
moon's light from the plain plate, so a strobe is never taken for it.

The tools read a few more variables:

- `MOON_DISC=disc.pfm`: the moon's disc rendered alone, standing at
  `MOON_DISC_ALT` degrees, so cues can move and fade it.
- `FIRE_FRAMES=dir`: the fire pass, `NNN.pfm` from ignition, rendered
  without the close spruces so no flame hides behind them.
- `TREEMASK=t.pfm`: the close spruces glowing alone (`-closeglow`);
  they stand back in front of every beam, fire and the mist.
- `WATERMASK=w.pfm`: the lake glowing alone (`-lakeglow`); where it
  shines is the water the camera sees, past every crest and trunk.
- `MIST_THICK=TIMES`: that many times the mist hangs on the lake.
- `SEARCH_COUNT=N`: how many searchlights stand behind the rim;
  eight unless said.
- `DEBUGMASK=d.pfm`: lightshow writes what it takes for sky, near
  trees and the rim line, for checking the masks by eye.

## a frame

Two GPU renders, once (8 s each), from the slayer root with
`SLAYER_SHELF` set to the sources shelf:

    hillside -set sets/moonlake.yaml -lake 1076 -moon 6 -moonglow 0.042 \
      -giants 5 -gpu -raw g5.pfm -out g5.png
    hillside -set sets/moonlake.yaml -moon 200 -moonglow 0.042 \
      -giants 5 -gpu -raw mask.pfm -out mask.png

Then at time t (about 0.8 s a frame on the CPU):

    export MOONREF=g5.pfm
    cueshow improv/moonlake/score.yaml improv/moonlake/cues.yaml t > cue.env
    mist=$(grep '^mist=' cue.env | cut -d= -f2)
    level=$(CUES=cue.env lightshow g5.pfm mask.pfm k.pfm n.pfm t)
    seawater s.pfm 1280 720 t 8 k.pfm
    FRONT=n.pfm seamatte k.pfm s.pfm o.pfm 7.0 -8.04 -1.9 1.554
    mistlake o.pfm m.pfm 7.0 -8.04 -1.9 ($mist + 0.25 level) 3 t
    slayer develop print.yaml m.pfm --out f.png

`print.yaml` is any job whose print is the one `develop -exposure 23
-bloom 0.8` gave: `print: {knee: 0.7, top: 8, stops: 23, bloom: 0.8}`.

## the score: from the song to the grid

`score.yaml` was made on 2026-11-09 by `make_score.py`, which is not in
this repository. It is in the session scratch folder, with its own
Python venv: demucs 4.1.0, librosa 0.11.0, basic-pitch 0.4.0, torch
2.15.0. Bring it here, or port what can be ported, before this ships.

The steps, as it ran them:

1. The song to a wav. The m4a holds Opus at 48 kHz, stereo, and a
   subtitle track of the lyric timings (`lyrics.srt` in the scratch, a
   cue source for the chant not yet used). Take the audio only:

       ffmpeg -i moon-lake.m4a -map 0:a moon-lake.wav

2. Stems. This is the slow part, minutes on the CPU:

       python -m demucs -n htdemucs -o stems moon-lake.wav

   It writes `stems/htdemucs/moon-lake/{drums,bass,other,vocals}.wav`:
   drums, bass, the chant, and other (the acid line and arps). demucs
   is a learned split, not an unmixing: the stems carry faint ghosts of
   each other. They are for measuring, never for listening.
3. The grid, from the drum stem alone: librosa's beat tracker,
   tightness 100, keeping beats where the drums play. Where they are
   out, beats at the median spacing carry the count through, and
   `measured` says which beats are which.
4. The sections: the drums play where the drum stem's loudness, in
   100 ms steps, is over 8 % of its 99th percentile. Gaps under 4 s are
   bridged, and spans under 4 s dropped.
5. Notes: basic-pitch transcribes the bass, other and vocals stems,
   which become the score's `bass`, `lead` and `chant`: 642, 2,377 and
   403 notes. A note's `vel` is its amplitude times 127. On the whole
   mix it would smear the three into one.
6. The bar phase: bar 1 beat 1 is the first drum beat, checked by eye
   and ear against `metronome` (2026-11-09).

### aubio

aubio is installed (`/opt/homebrew/bin/aubio*`) and needs no Python.
Measured on this song on 2026-11-10, comparing each list with the
nearest entry in the other, from 59 to 190 s:

- On the whole mix, aubio's beats fit to 117.3 bpm with 30 ms of
  scatter, which is wrong. That first pass, `beats.txt` in the scratch,
  is why the grid comes from the drum stem. Never track beats on the mix.
- On the drum stem, `aubiotrack -i drums.wav > beats.txt` finds
  145.2 bpm against the grid's 143.55. Its beats land a median 13.6 ms
  before the drum hits, the middle half spread over 22 ms.
- `aubioonset -i drums.wav > onsets.txt` marks every drum hit's attack,
  1182 of them. The grid lands a steady 17.6 ms after them, the middle
  half spread over 8 ms.

The two aubio commands above write the beat and onset lists. The script
that compares them with the grid, and `drums.wav`, are in the session
scratch and not in this repository, so these figures are as recorded on
2026-11-10 and were not rerun.

So the grid is librosa's, and aubio is for the attacks. The lights fire
17.6 ms after the drums: under half a frame at 24 fps, about a frame at
60. Before a 60 fps final, snap each grid beat to the nearest onset
within 30 ms. `aubionotes` and `aubiopitch` could do the transcription
without Python; neither has been tried on this song.

## why the lake is a matte

hillside's moon is a painted disc marked clear, so it never blocks its
own moonlight, and clear things never appear in reflections; flames are
volumes, which mirrors do not see. The `-lake` water renders black, so
the water is oceantex's, laid in below the far shore.

## next

- the mist as wisps only, lit by whatever shines: moon, strobe, beam
- the moon rise from near darkness, the disc sliding up behind the mask
- the blackout, then the ship: a daylight-strength cool light from
  overhead as its own pass, the exposure adapting, then the fade back
- a front light on the crater wall
