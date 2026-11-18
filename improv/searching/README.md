V5: house of suns, recurring lantern/Zappy, developed four-character cast
and96beings beyondtheopening. SHOOT.md/MUSIC.md andPROOF-v5.txt lead.
Newrecipe studies/heard-v5-proof-480.yaml; priorproofs preserved.

V4 pacing pass: SHOOT.md merges the new timing notes with the earlier
ending. PROOF-v4.txt records the implemented changes and remaining limits.
New plan: studies/heard-v4-proof-480.yaml. V3 artifacts remain unchanged.

# heard.

The complete v3 proof is rendered: 1656 actual 480x270 frames at six fps,
four primary rays per pixel, with the revised Glass Hour arrangement.
Watch the `heard-v3-full-480` render.

[PROOF-v3.txt](PROOF-v3.txt) records the new scene integration, fastlight
settings, two producer segments and remaining visual limitations.
[SHOOT.md](SHOOT.md), [MUSIC.md](MUSIC.md) and [EDIT-v3.txt](EDIT-v3.txt)
retain the intended script and score. The separate uncompressed listening
review remains at the `heard-script-v3` render.

## Authored anatomy mesh sharing

Authored `individual` stages with an `anatomy` plan and `authored-multitude`
stages share identical octahedron and sail meshes within that stage by
default. Materials, placements and animation remain per component.

To inspect the same authored construction with separate mesh storage, set
`share-anatomy-prototypes: false` beside the stage's `kind`; omitting the field
preserves the existing sharing behavior. This switch does not choose mesh
detail, projected detail, quality or render shortcuts.

The switch is accepted only for those two authored stage forms. It is valid in
a direct `named: heard` job's `scene.with.stage` and in a Heard film shot's
`stage` plan. The named-scene `with` object is part of the job picture key, so
an explicit `false` receives a distinct key. Omit an explicit `true` when the
default behavior is wanted and the prior job key should remain unchanged.

The command walkthrough below documents the historical first proof.
Its old song, generic opening and cue positions are not the new script.

The first proof follows two optical observatories through passing days,
a shared lock, two radio arrays and stylized blazar views. The revised
screenplay expands the answer into a cast of respondents, space instruments
and a Laniakea-scale megastructure reveal. The v3 proof above implements
that edit with explicitly rough respondent and larger-scale field studies.

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
2800 MiB (owner-authorized 15 November 2026); the output filesystem must
retain 30 GiB. Cancellation is
checked between exposures; a slow native draw may finish before it stops.
These guards do not measure or control other agents' memory or jobs.

Keep the executable frozen while a sequence is live. Rebuilding it during
the run would make the producer stamp and saved provenance ambiguous.

## Overlap drawing and output

The sequence now overlaps the next GPU draw with CPU development, PNG
writing and movie submission for the previous frame. One finishing worker
keeps output ordered. An unbuffered handoff bounds the work to two frames:
one tracing and one finishing, with no queue of future scene copies.

The finished trace owns its pixels and display settings. Scene updates and
GPU calls stay sequential; the output worker never reads the mutable scene.
Cancellation or an output failure stops new draws and joins that worker
before closing the renderer or movie. A native draw may finish first.

Use `--serial` on `searching render` for a matched control. Both modes use
the same tracing and finishing code. The receipt records drawing and output
stage seconds separately; these overlap and must not be added to estimate
elapsed time. No measured speedup is claimed until a paired render finishes.

The Metal stream already submits the next ray band before consuming the
current one. This frame pipeline adds overlap above that stream; it does
not add concurrent scene mutation or promise to saturate the GPU. Future
scene preparation needs an immutable handoff before it can safely overlap.

While an older take is running, `game5 run searching-pipeline-build` builds
the candidate as `bin/searching-pipeline`. `game5 run searching-pipeline`
uses that separate executable. Keep the old producer frozen until its take
finishes. Use `game5`, the installed rc-v0.5.0 tool, for these commands.

## Progress in Barkeep

The progress watcher can attach to an older running producer. It counts
PNG files with a complete end marker, reports bytes and percentage written,
and estimates pace from at most the last 24 completed frames. Elapsed time
starts at the first saved PNG, so it excludes initial scene preparation.

```sh
game5 run searching-status-build -- ./cmd/searching
game5 run searching-status -- progress RENDER_DIRECTORY --follow \
  --pid PRODUCER_PID --progress-file RENDERS_DIRECTORY/render-progress.txt
```

The Barkeep side column reads that file for its `rndr`, `f/m` and `eta`
rows. Each atomic snapshot contains five integers: completed PNGs, total,
rounded frames per minute, seconds remaining and Unix update time. Unknown
ETA is -1; zero means movie completion. The watcher stops at completion
or producer exit; the column clears readings older than 120 seconds.

One watcher owns the shared file at a time. An updated column reader must
be installed and running to display these rows; the watcher never edits or
restarts the user's column. The detailed count and bytes remain in JSON.

Carrie also serves a separate loopback API. For that integration, use
`--barkeep http://127.0.0.1:PORT`, `--barkeep-kind` and `--barkeep-signal`.

Supply its address and the
names allowed by its configuration. The watcher sends running/pass/fail
signals and timestamped meter messages. It sends messages at most once a
minute and only when the queue is empty; this is an occasional status line,
not a continuously replaced graphical widget. It never changes tmux setup.

Without the Barkeep options, it prints the same line and updates
`progress.json` beside the take every ten seconds. Excerpts need the same
`--from` and `--frames` as their render. `--pid` lets the watcher distinguish
a stopped producer from a slow frame. Interrupting the watcher leaves the
render alone. Unreachable or refused Barkeep updates are reported.

ETA extrapolates recent wall time, including the current wait. Later shots
may have very different costs. After a gap longer than three recent frame
intervals or 30 seconds, whichever is longer, the ETA is withdrawn. All
PNGs written means movie finalization is pending; completion additionally
requires the producer's matching receipt and movie.

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

The generic Omnipositor set adapter is available separately. This shoot
driver draws complete frames and does not use its reduced transport or
reused scenery plates.

## Flow and answering-pulse study

[studies/heard-flow-720.yaml](studies/heard-flow-720.yaml) opts into a
separate reveal study. Its flow-study block contains explicit polylines,
fixed luminous landmarks, camera placement and three song cues. The cyan
point is an authored return destination, not a measured position of Earth.

Large studies may put `paths`, `landmarks` and `clouds` in one relative YAML
file and name it with `flow-study.geometry.file` plus its exact lowercase
SHA-256 in `flow-study.geometry.sha256`. Camera, cue and return-path values
stay in the plan.

The reader rejects mixed inline/reference geometry, unsafe paths, missing
files and changed bytes. A rendered take copies the referenced file beside
`shoot.yaml` at the same relative path, writes its `.txt` source view, and
records the geometry digest in `render.json`.

Copied plans, progress and behind-the-scenes checks therefore read the same
pinned geometry. Each referenced asset is limited to 2 MiB at both validation
and copy time.

The paths and their units are authored. They are not a numerical Laniakea
reconstruction, a galaxy catalogue or a magnetoconvection simulation.
The reader requires provenance: authored-previs so this distinction stays
with the inputs and the completion receipt.

Light packets follow cumulative path distance. Later, the inner geometry
folds and one bright packet travels outward along return-path. Landmarks
and the camera remain fixed. The fold slightly stretches filament widths
with their segments; these meshes visualize a gesture, not material strain.

The preview accepts up to 1280 by 720, with one or two rays per axis.
The retained recipe uses 120 paths, 1600 authored landmarks and 28 faint
wispy volumes. The input limits are 128 paths, 128 vertices per path,
4096 landmarks and 48 volumes. Diffuse emission adds depth; it is not a
measured image or a physical model of gas in the supercluster.

```sh
game lane gpu -- game run searching -- render \
  improv/searching/studies/heard-flow-720.yaml \
  --from 1785 --frames 123 --out /absolute/new-flow-study
```

That range is 297.5 to 318 seconds of the original music, six fps.
The cues are start 117.1, fold 121.1, answer 123.1. These are editable
dramatic placements; only the earlier observatory lock was owner-timed.
The command also saves browser-readable shoot.yaml.txt and score.yaml.txt.

### Guide-beam width

Ground scenes use a nominal 30 cm guide-beam diameter from ESO 4LGSF
(`eso-4lgsf` in `sources.yaml`). The Gaussian emitter takes this as its
1/e² intensity diameter; ESO does not specify that contour. Construction
and animated updates use the same radius.

Peak irradiance is rescaled to preserve the earlier authored beam's total
light at the narrower width. This retains its visual brightness without
claiming calibration to published laser power. Matched 720p controls
separate this choice from the dimmer fixed-peak alternative.

The ground beam contribution is drawn after its complementary base at
full output resolution. Both retain geometry and all foreground extinction;
their linear light is added before exposure, bloom and tone mapping.
Surface beam lighting and secondary paths stay in the base. Both retained
drawers close at scene transitions and when the renderer closes.

### An authored soundtrack

Prepare Glass Hour once, preserving its composition and voicing:

    game run searching -- music improv/searching/studies/music/glass-hour-source.json --voicing cheeptune/builtin/night-pad.json --out NEW_MUSIC_DIRECTORY

The new directory holds music.wav, score.json, browser-readable .txt copies
and music.json with source hashes. Use music.wav and score.json as the
shoot's music and score. Tempo changes and rests follow Cheeptune's clock;
the estimated status records pending editorial approval. Synthesis requires
a non-looping score of at most ten minutes.

A flow-study may add travel with to, look-from, look-to and until. Eye is the
starting camera; to is its destination. Until is a bar.beat cue no later
than fold. The radius changes logarithmically with eased endpoints, then
the camera holds. These coordinates remain authored scene units.

For a memory-constrained preview, render with --defer-movie. It writes real
PNGs first, closes the retained scene, then encodes those pictures with the
exact audio interval. The receipt separates encoding time. The default
still overlaps drawing and finishing; --serial additionally holds only one
frame in that pipeline. A deferred take is complete only after its movie
and synchronized companion have passed the ordinary completion checks.
