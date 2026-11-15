# pan

A film made of stills, each moved slowly under a window, cut on a
tune's bars, with the tune under it. It is for a quick upload when
tracing motion would cost too much: a still is traced once, a little
larger than the film, and every frame is a real crop of its pixels. The
window is never smaller than the film, so nothing is scaled up.

## The script

YAML, read by `Read`. A relative path is under the script's folder.

    size: 2k                    # a job preset (job.Presets), or
                                # {width: 2048, height: 1152}
    fps: 24
    sound: {score: PATH, voicing: "@night-pad", bars: [1, 14]}
    title: {words: glass hour, until: 4, rows: 40}
    shots:
      - still: moonlake.png     # a png at least the film's size
        bars: 4                 # how long, in the tune's bars
        from: [0, 0]            # where the window starts and ends,
        to: [1, 0.5]            # as shares of the still's slack
        fade: {in: 1, out: 0}   # seconds from and to black

The shots' bars add up to the sound's bars: a shot starts and ends on a
bar line, so the cuts fall on the music. The window moves from `from`
to `to` eased in and out. A still smaller than the film on either side
is refused by name. `title` is drawn with `hud` in the default theme's
ink, small (many rows), centred high in the sky, until its seconds.

## The sound

`cheeptune.Open(score, voicing)`, cut from the start of the first bar
to the end of the last by the score's own clock (`SecondsAt`), with a
short fade at the end so a cut inside the piece does not click.

## The film

`interfaces/movie`, at the film's size, the frames handed over in order;
a cancelled context throws the movie away. Nothing else is written.

## What it is not

No motion is traced and nothing is scaled. The film is landscape or
portrait as its size says. No ffmpeg.
