# what the studio machine could not do

Three things wanted while designing the beds and kits candidates. Each is
a missing parameter; none was worked around by changing the machine.

## separate levels for a drum's tone and its noise

A drum that has both a `tone` and `noise` has one `noise` number, and the
tone is always full. A hat with a tone in the middle then measures as
that tone (about 90 percent of its energy in 1.2 to 2.5 kHz), so the
candidates use noise-only hats.

Wanted: a `tone_level` from 0 to 1 beside
`noise`, the tone's loudness against the hiss. It would let a hat or a
snare have a quiet tuned body under its hiss, and a clockwork tick carry
a faint pitch, without either drowning the other.

## a low-pass colour on drum noise

`colour` only cuts hiss away below a frequency, and the hiss is white up
to half the sample rate, so a hat cut at 2.8 kHz still has about 75
percent of its energy above 8 kHz. Wanted: `colour_top`, 20 to 16000 Hz,
a frequency above which the hiss is cut away (it must be above `colour`).
It would let a hat or a ratchet sit in a band such as 3 to 8 kHz, where
it can be heard against a tune, instead of being mostly above 8 kHz.

## formants for a choir

A choir or a vowel pad needs fixed resonant peaks that do not move with
the note, and the filter's `cutoff` follows the note and has one peak.

Wanted: an analog filter option `formants`, a list of up to three
`{"frequency": 700, "width": 120, "level": 1}` band-pass peaks in hertz
that are applied in parallel to the oscillators and do not follow the
note. With saws as the source (whole-number overtones) it would make
"ah" and "oh" pads whose overtones stay in tune with the note.
