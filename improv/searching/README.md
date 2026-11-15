# heard.

The first proof follows two optical observatories through passing days,
a shared lock, two radio arrays and stylized blazar views. The revised
screenplay expands the answer into a cast of respondents, space instruments
and a Laniakea-scale megastructure reveal. That revision is not rendered yet.

[SHOOT.md](SHOOT.md) is the screenplay. [shoot.yaml](shoot.yaml) is the
first edit. This is a storyboard proof at 320 by 180 and six frames per
second, using the song's original playback speed.

## The three clocks

The source-bound beat grid supplies irregular song beats. The camera
integrates four authored velocity weights continuously: one moves most,
two and three slow down, four nearly holds. It never jumps back to the
beginning of a bar. Downbeat beam and lens accents use the nearest frame.

The weather and sky use an independent scenery clock. During the search,
six authored days pass before the shared lock. After it, the sky holds at
night while the mist keeps flowing. The score never accelerates with the
scenery clock.

Frames use their absolute position in the song. An excerpt beginning at
frame 696 at six fps starts its sound at 116 seconds; neither the score's
bar numbers nor the edit cues are reset.

## Prepare and draw

Set music and score in shoot.yaml to existing local files. The score must
name the original audio hash, use four-beat bars and cover the song.
Every story cue is a one-based bar.beat position; lock is 48.4, which
lands at 123.01 seconds in the supplied grid, approximately 2:03.

Build a committed producer, then inspect the compiled cue times:

```sh
game run searching-build -- -ldflags "-X main.build=$(git rev-parse HEAD)" ./cmd/searching
game run searching -- plan improv/searching/shoot.yaml
```

Draw a representative exposure before starting a sequence. The output
directory must exist for a still; the command refuses to replace a file.

```sh
game lane gpu -- game run searching -- still improv/searching/shoot.yaml \
  --at 126 --site hawaii --out /absolute/render-folder/hawaii.png
```

Sites are hawaii, atacama, radio-hawaii, radio-atacama and blazar. A still
override changes the set, while keeping the song time and edit state.
Normal sequences choose their sites from the edit.

```sh
game lane gpu -- game run searching -- render improv/searching/shoot.yaml \
  --from 696 --frames 96 --out /absolute/new-lock-preview
game lane gpu -- game run searching -- render improv/searching/shoot.yaml \
  --out /absolute/new-full-preview --limit-seconds 3600
```

The sequence saves its plan, score, actual PNG frames, native H.264/AAC
movie and completion receipt. The receipt records the producer commit,
input hashes, range, render time and probed movie facts. The encoder adds
12 black rows below a 180-row picture for QuickTime compatibility.

The default wall limit is 600 seconds. Own-process peak RSS is limited to
1400 MiB and the output filesystem must retain 30 GiB. Cancellation is
checked between exposures; a slow native draw may finish before it stops.
These guards do not measure or control other agents' memory or jobs.

Keep the executable frozen while a sequence is live. Rebuilding it during
the run would make the producer stamp and saved provenance ambiguous.

## What this proof leaves open

The set builder uses manufactured meshes and retained GPU drawers. The
two optical sites share an enclosure family; the radio sites share an
antenna family with different layout scales. Whole assemblies rotate in
azimuth, including their bases. Separate fixed piers, articulated mounts
and elevation motion still need work.

The optical lasers share an aim within each set and converge on one
authored direction at the lock. This tells the story; it does not solve
pointing between two geographic sites. Moon exposure and the flowing
weather are authored, without a measured weather-data claim.

The blazar uses emissive volumes. Changing component weights before
tracing reveals different structures; these are illustrations of named
spectral views, without measured band response or flux. Site details
remain in the builder; separate typed site recipes are a next step.

Omnipositor's existing Moonlake pass compositor remains on its own branch.
This driver does not merge that experimental implementation or claim to
provide its pass-level controls.
