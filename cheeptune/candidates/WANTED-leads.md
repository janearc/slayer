# wanted by the leads and basses lane

What the studio machine could not do when these instruments were made, on
12 November 2026. Measured with a scratch program on the audition riff;
nobody listened.

## a DC block (a fault, not a missing feature)

A pulse wave of any width but one half is not centred on zero, and the
studio does not remove the offset. A lead of two pulses at width 0.2
measured a mean of -0.042 against an rms of 0.062: most of the lead's
level was offset. It costs headroom, adds a click at each note start and
stop, and puts energy under 120 Hz that is not music (41 percent of that
lead's energy measured below 120 Hz, on notes that start near 250 Hz).

The voicings here avoid it by using width 0.5; `square-acid` uses 0.42 and
still measures -0.005. The fix wanted is a high-pass of a few hertz on
each analog channel's output, or on the studio's mix. Narrow pulse leads
become usable once it is there.

## oscillator sync

The analog kind has two oscillators and no way to reset one from the
other. Wanted: a `sync` flag on the second oscillator, so it restarts
each time the first completes a cycle, with its own pitch moved by
`octave` or `detune` or an envelope on the filter. It would give the
classic hard-sync lead, whose colour sweeps as the second pitch moves.

`sweep-lead` imitates the sweep with fm at a whole ratio, which holds
whole-number overtones but leaves every third harmonic missing; true sync
fills them in.

## a solo for one role

Measuring a voice alone means copying the voicing, keeping that role's
channels and dropping the others from the copy. The shared reverb and
delay stay on, so a solo carries its room and the roles are not measured
dry.

Wanted: a way to render one role, or one channel, with the effects
bypassed, as a call on the machine. It would let a level, a crest factor
and a spectrum be measured without the room mixed in, and make gates such
as "bass 2 to 8 dB under the lead" exact.
