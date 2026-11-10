# two minutes in the grove

This is a dated development record. Its timings and check results belong to
the revisions named below. Use the [Game workflow](developing.md) for current
setup and commands, and [the committed 4K review](review-4k-2026-11-05.md)
for the later dense-forest measurement.

state: 2026-11-04. the actual sequence is complete: 2,880 distinct GPU
renders, 3840x2160, 24 fps, 120 seconds, one ray per pixel. this is the
500-tree night scene in [fire-night.yaml](fire-night.yaml), with clump
foliage, a central fire, a prescribed front, smoke and glowing firebrands.

## the run

the frozen renderer was built from commit 2baf232. frames took 4 hours,
53 minutes, 56 seconds, from 08:54:08 to 13:48:04 UTC. the 4K master and
1080p viewing copy finished encoding and probing at 13:51:54 UTC.
rendering used two Go CPU workers, with one heavy job at a time.

the saved frames occupy 20,675,796,395 bytes. every PNG was read back and
matched to its recorded SHA256 and exact integer timestamp. all 2,880
PNG hashes are distinct. the last frame is at 119.958333333 seconds;
there are no interpolated, duplicated or looped frames.

both H.264 files were decoded for their frame counts: 2,880 frames,
24/1 fps and exactly 120 seconds. decoded images at 40 seconds and the
final frame were inspected beside the source PNGs. the poster is a copy
of the actual 40-second PNG.

the artifacts are under the builder's local/renders/fire-night directory,
outside this repository:

- 14-night-fire-4k.mp4 is the 4K master.
- 14-night-fire-1080p.mp4 is the smaller viewing copy.
- 14-night-poster.png is the lossless still.
- 14-complete-validation.json records hashes, timestamps and timing.
- 14-night-frames holds the lossless intermediate frames and manifest.

## measured cost

per-frame wall time includes dynamic lights and particles, rendering,
development and PNG output. it excludes the one-time scene construction.
the complete render wall time above includes construction and progress writes.

- mean: 6.118 seconds.
- median: 6.326 seconds.
- 95th percentile: 6.969 seconds.
- range: 4.389 to 7.777 seconds.

there were 469 to 1,159 firebrands and 153 to 252 finite/distant lamps per
frame. the prescribed front finishes around 95 seconds; later frames cost
less. observed resident memory varied from about 4 to 11 GB on a 48 GB
machine, returning to about 7 GB near the end.

the matched 960x540, four-ray study at 20 seconds took 20.055 seconds
with direct native lighting and 4.378 seconds with the smoke-light cache.
GPU tracing fell from 16.766 to 0.762 seconds. this is a matched comparison;
the long run's one-ray 4K settings are a separate measurement.

in that pair, 0.134645% of developed pixels moved more than one OKLab JND,
with a worst difference of 2.280869 JND. it fits the recorded 2% still
budget. this isolates the lighting cache against direct native rendering,
not every approximation or a temporal acceptance budget.

## output and colour

fast lossless intermediate PNG writes reduced a real 4K image's write
time from about 1.72 to 0.375 seconds. both writers preserved every decoded
RGBA byte on three actual 4K inputs. file sizes increased. the general
PNG writer retains its old behavior.

separate GPU repeats are not promised byte-identical: one pilot pair
differed in four channels, by up to 13/255, while two other pairs matched.
that is separate from the writer's same-input, lossless-pixel check.

the videos use libx264, veryfast, CRF 16 for 4K and CRF 18 for the smaller
copy, yuv420p and two encoder threads. their colour tags explicitly record
sRGB primaries/transfer and the limited-range BT.601 matrix actually used
by the encoder.

matrix interpretation was checked against the source image. tags were
added by stream copy, without re-encoding; a full decoded RGB frame
matched the original BT.601 interpretation byte for byte. untagged copies
were retained with an -untagged suffix.

this is a separate encode from the fuel study, whose checked file uses a
BT.709 matrix. the matrices describe their respective existing conversions;
the night film was not re-encoded merely to make those tags uniform.

## model and checks

the [design and source notes](fire-night-design.md) explain the authored
front, temperature profiles and chemical emission inputs. sodium line
positions come from the linked NIST table; their powers are authored.
the smoke coefficients are an explicit Douglas-fir proxy.

the front does not consume or char tree geometry. its shared smoke has
no transported burn history. flames and smoke remain vertical; firebrands
respond to prescribed wind, loft and gravity. this is a rendering study
with prescribed ignition, not a prediction of wildfire behavior.

the full Game functional suite passed after the front work. the overall
check still fails the existing history rule at 03d6b0a. after the later
PNG-only optimization, the complete file package and focused sequence
checks passed, including actual GPU time samples and resume.

lint and review gates remain open. the last configured lint run reported
49 hard findings and 1,230 review findings. the full new-code mutation
sweep, whole-project 100% coverage, prose-model and independent review
are not established. these are local development commits, not a release.
