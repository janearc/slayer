# walk

Short vertical films that walk a stranger through a set or a scene: a
picture on top, labels on it pointing at real places, and a plain panel
under it naming what is looked at, two or three of its numbers, and one
sentence. Any set or scene can use it; nothing here knows moonlake.

It lives in `improv/` beside the sets' own improv tools because it is
made for shorts, not for the renderer, and it is its own
package, not under `improv/moonlake/`, because the next sets (a night
fire, an orbital, a star in shells of glass) use it as they are.

## What a set supplies

1. A set file whose cameras the shots name (`set.Read`). Its film must
   have a rig (frame and lens), since that is how the tracer frames a
   picture, and so how a place is projected; or, with no rig, say how
   `wide` it sees, and it is the pinhole the session draws: that angle
   across, square pixels, a still the film's own shape (`rigless`,
   tested against `testdata/pinhole`).
2. Where its named things stand: the table `hillside -set SET -places
   FILE` writes (of, kind, east, north, height). A cheap run is enough:
   the places do not depend on the kinds, so a copy of the set with
   every thing a cobble and the film 64 across writes the same table.
3. A traced still for each shot, drawn from that set and camera, as a
   PNG. Its width is the one the projection is told.
4. The script (below), with every number's source beside it.
5. Optionally a song and its bars' times, for the sound under it.

## What a scene that is not a set supplies

A named scene or a smoke has no set file and no places table. Its
script names no `set`; then:

- its places, if any, are given in the script with their unit, x, y
  and z with y up: `places: {unit: km, at: {the star: [0, 0, 0]}}`;
- a shot gives its camera as an eye, in the scene's units, and the
  angle a pinhole sees across the picture (`film.Pinhole`'s `Wide`):
  `eye: {from: [0, 627, 2457], look-at: [0, 0, 0], up: [0, 1, 0],
  wide: 40}`; the picture's own size says the rest;
- or a shot gives no camera at all, and points only at pixels.

A label may always be put at a pixel of the still, `{pixel: [x, y],
words: ...}`, taken as given with no projection: for light that bends,
where only the caller's own ray march knows where a thing appears. Its
words should say it marks where the thing appears.

A scale bar may be given outright, `scale: {pixels: 120, length: "10",
unit: km}`: so many pixels of the still stand for that length. It is
drawn at the picture's foot on the left. A bar measured at a place is in
the places' unit: "1,000 km".

## The script

YAML, read by `ReadScript`. The sentences are drafts, to be rewritten.

    set: the set file, relative to the script
    places: the places table, relative to the script
    shorts:
      NAME:                       # words, used for the folder
        title: what the short is
        sound: {bar: 17}          # the song from this bar, low
        shots:
          - camera: NAME          # a camera of the set
            set: other.yaml       # its own set copy, if not the script's
            picture: FILE.png     # the traced still
            seconds: 8
            lit: show             # or plate, inspection or drawing
            tag: own words        # in place of lit, at most tagMost
            window: {from: [0, 0, 1], to: [0.2, 0.1, 0.6]}
            name: what is looked at
            figures:
              - {value: 71.7, unit: m, says: above the water,
                 from: where the number comes from}
            labels:
              - {thing: the west rise, words: you stand here}
              - {at: [263450, 2148300, 1076], words: the lake}
              - {toward: [109.65, 2.3], words: the moon}
              - {pixel: [812, 400], words: the ring appears here}
            scale: {thing: the lookout}
            sentence: one plain sentence

`ReadScript` takes a folder of pictures: the shots' stills and their own
sets are found there, so the traced pictures need not sit beside the
script. With none, the script's folder is used.

A shot may name its own set: a copy of the set with another frame, lens
or light, for that camera. Its things must stand where the places say.

A window is a share of the still: its left, its top, and its width, the
height following from the picture's shape. A shot with `from` and `to`
moves the window between them, eased. No window is the whole still.

A label points at a named thing, at a place (east, north, height in
metres on a set; x, y, z in a scene), toward a way in the sky (azimuth
and altitude in degrees, for something as far as the moon; sets only),
or at a pixel.

`side` puts the words left, right, up or down of the place; up is the
default. Words stand on their side if they fit there clear of the place
and of the labels and bar drawn before them; else on the first other
side that does.

`ReadScript` refuses, by name: a camera the set lacks, a label's thing
the places lack, a figure with no `from`, a sentence longer than its
shot gives time to read at `readingRate`, a `lit` it does not know or a
`lit` and a `tag` both, a window off its still; and in a script with no
set, a camera or set named, an eye with no unit to measure in, and a
thing, place or way with no eye to find it through.

## The projection

`PinholeCamera` is a scene's: a pinhole at an eye, as `film.Pinhole`
draws. `Camera` is otherwise a set's camera made ready: its eye and the way it faces from
`set.Shot`, its frame and lens from the set's rig, and the picture's
width. `Project` takes a place to the pixel the tracer draws it at, and
says whether it is in front of the eye and on the picture.

The tracer's lens is thin and focused (`ninjafilm/lens.Thin`): the film
stands behind it at the image distance, focal times focus over focus
less focal, and a point is seen along the ray through the lens's middle.
So the projection is a pinhole at the image distance, not the focal
length, which matters for a near focus.

It is tested against a traced picture of a small bright marker at a
known place (`testdata/marker`), within `markerPixels`.

## The scale bar

`ScaleBar` is a round length (1, 2 or 5 times a power of ten metres)
and its length in pixels, measured across the picture at a place on the
ground, so a bar is true where it stands and nowhere else when the
ground is seen at a slant. The panel says where it is true.

## The frame

1080 by 1920 at 24 a second. The picture is the top 1080 by 1080; the
panel is the rest, on the hud's default ground, in the hud theme's ink:

1. The name of what is looked at.
2. Two or three figures, each a large number, its unit, and a few words.
3. The sentence, wrapped, at most four rows.

A shot not lit as in the show carries a plain tag on the picture: a
plate, the set's own light with the show's lights off, says so; one
drawn for inspection says "Lit for inspection, not as in the show"; a
drawing says "A drawing, not a photograph"; a `tag` says its own words.

Labels are ink on an opaque plate of ground, with a leader to a ringed
dot on the place, so their contrast is the panel's whatever the picture
under them.

Between shots the picture dips through the ground for a fifth of a
second.

## The sound

The song from a bar, at `soundLevel`, faded in and out over
`soundFade`. Written by `interfaces/movie`. No other program is run.
