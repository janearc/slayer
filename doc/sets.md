# sets

a set file is a scene written as data, in yaml, so it can be read,
diffed and handed to anyone without reading go.

it says what a scene wants, at whatever level the scene knows it: a
hillside by its terrain and its stands of trees, a planet by its air,
or a house box by box. the set package reads it and checks every
value; the package each part belongs to builds it.

sets/shoebox.yaml is the house under saturn, its world piece by piece,
and house draws it. sets/kilauea-iki.yaml and the others are hillsides
and planets, and hillside draws them.

the later [ecosystem subset](observatory-ecosystem.md) adds recipes,
complexity and prescribed weather. its bundled schema covers only that
subset; the legacy fields below remain available without `$schema`.

a set may name one of two schemas in `$schema`: version 1, the schema
above, or version 2 (`set/ecosystem-v2.schema.json`), which has all of
version 1 and adds scenes set in space, bodies in orbit, and cameras that
move. one reader takes both and checks a file against the schema it names.

a feature of version 2 that the reader keeps and no stage draws yet is
refused by the session by name, with the file, the key and where it stands,
before anything is staged.

two are drawn. `setting: space` is drawn by the session's stage for a scene
in space (session/afloat.go), and a film's `fps` is read and has no use
in a still. sets/webb-at-l2.yaml is the example.

a scene in space:

- has no ground and no air. its sky is black unless its world half says
  one.
- stands each thing, a kind the manufactory makes, either at its legacy
  east, north and height or at an explicit `position` in metres from the
  scene's origin, and turns it by its heading or not at all.
- is lit by `light: sun`, one distant lamp from `sun:` at any altitude,
  as bright as sunlight at the earth at the film's exposure; or by
  `light: none`, no lamp.
- is seen by a fixed camera, `location.at` and `orientation.look_at`,
  or one placed off a thing.
- refuses by name stands, texts, ecosystems, fire fronts, fuel beds,
  layers of air, a planet, a burning thing and a recipe.

every other version 2 key is still refused by name: the date of a scene
in space (on the ground the date works the sun, see the sun, below), the
frame, an anchor, a globe, a sky of stars, an orbit, an attitude, an
eye, a camera's moves and duration, and the kinds point, star, lantern,
culture-orbital and nebula.

## what a set says

everything is under scene:

- name: the scene's name.
- light: ambient, moon or sun.
- location: lat and lon, in degrees. timeofday: hhmm, on a local clock.
  date: the day, as year-month-day. utc-offset: how far the clock is
  ahead of UTC, as `{value: 8, unit: h}`, from -12 to 14 hours. these
  four work the sun; see the sun, below.
- terrain: the ground, a window of a published elevation model. the
  set names it: `from` is the dataset's key in sources.yaml, `file` the
  file of it that is read from the shelf.

  a window is placed one of two ways, and never both.
  - by `west` and `north`, the window's first corner in the model's own
    metres, then `across`, `down` and `step`, and `smooth` if it is
    wanted. this is how every set so far reads its ground.
  - by `lat` and `lon`, for a model that is a map of a whole sphere
    (equirectangular, as the Magellan map of Venus is). `lat` is degrees
    north, -90 to 90. `lon` is degrees east, any value from -360 to 360.

    the window is laid round that place, with the place at its middle,
    and has its own metres: east runs from 0 at the west edge to
    `across`, north from 0 at the south edge to `down`, so the middle is
    half of each and nothing is negative.

    a thing's `east` and `north` are in those metres. a window may not
    be wider than a hundredth of the sphere's radius, the most a flat
    patch is laid on it for.

    without `detail` the ground is the map alone, read between its
    samples, so below the map's spacing it is a smooth ramp. with
    `detail: {seed: N}` ground finer than the map is made by package
    titan from the seed, and the run says how much of it was made. the
    same seed makes the same ground.

  a place the map has no height for is an error that names its
  latitude and longitude; nothing made stands in for missing map.
  `sets/venera-13-plain.yaml` is the example.

  the ground has one colour, its `reflectance`, unless the set says a
  `cover`: surfaces chosen by rule from where the ground stands and
  which way it leans. a terrain says a reflectance or a cover, not both.

      cover:
        level: 0
        layers:
          - name: rock
            reflectance: [0.19, 0.16, 0.12]
            source: "where the colour came from"
          - name: sand
            reflectance: [0.69, 0.55, 0.26]
            source: "where the colour came from"
            height: {from: -3, until: 5, blend: 4}
            slope: {until: 0.35, blend: 0.2}

  the sea's level, the height of the sea on the model's own datum in
  metres, has one owner. where the terrain says its `sea`, that is the
  level, and the cover leaves `level` out; a `level` that differs from
  the terrain's `sea` is refused, with both numbers.

  where the terrain
  says no sea, as these sets' do not, the cover says `level`. an ocean
  in the world whose `middle` height differs from the level is refused
  too, with both numbers: the sand's edge and the water's edge are one
  height. every height a rule names is above the level, so a bed is
  under nought.

  `layers` is the surfaces from the ground up. the first is the ground
  under the rest, everywhere, and says no rule. each later one is laid
  over those before it as far as its rules hold, and says at least one of
  them.

  `reflectance` is the share of red, green and blue light the
  surface gives back, in linear light, and `source` says where the
  numbers came from; both are required, as a world material's colour and
  source are.

  a rule is `height`, in metres above the sea, or `slope`, rise over run
  (one is forty-five degrees, a wall is past ten), of the ground's
  shading normal, which a survey's samples carry smoothly across a
  triangle. it says `from`, `until` or both, and a `blend`, which is
  always said.

  the surface lies fully between `from` and `until`, none of
  it lies a half blend outside them, and across the blend, centred on
  the end, it is eased, so there is no line. a surface that says both a
  height and a slope lies where both hold.

  the processor tracer reads a cover by height and slope. the GPU's
  paint is asked for a point and not the way the ground faces, so it
  could only paint by height, and a cliff would not be rock there: the
  GPU refuses a ground with a cover, in words that name the processor
  renderer (`--renderer cpu`). `sets/honopu.yaml` and
  `sets/rarotonga.yaml` are the examples.

  a window draws as its survey's triangles, flat between samples, unless
  it says `made`: ground finer than the survey wherever the camera
  resolves finer. a window placed by lat and lon may say it too, but not
  beside a `detail`, which makes ground below the map another way, nor
  with a `near` window, which it would lie over.

      made:
        seed: 1
        steep: 1.7
        flutes: {depth: 1, width: 1.6}
        beds: {spacing: 3, standout: 0.4}

  only `seed` is required, and the same seed makes the same ground.

  each cell of the survey is cut in halves each way until its edges are no
  longer than a pixel spans there, through the camera the run draws; a
  cell out of view, or one the survey is as fine as, is left whole. the
  made faces are kept to two a pixel of the film, so what is made grows
  with the film.

  the survey's own samples and triangles are held whole
  as well, so a bigger window costs memory too.

  measured on
  sets/honopu.yaml, 817 by 721 samples, camera close: 589 MB at most
  without `made`, 1318 MB with it at 640 by 360; a larger film costs
  more, at roughly the same rate per pixel. every surveyed height stays
  where it was, and no made corner moves more than a quarter of the
  survey's spacing.

  the camera decides only how finely the ground is laid, not its shape:
  every made height at a place is the same from every camera, but for
  the waves of roughness too fine for that camera's pixel, whose sum is
  less than a pixel there.

  what is made: roughness at every wavelength under the survey's own,
  read from how broken the survey is over seven samples and how much more
  broken over thirteen (28 and 52 m at a 4 m step), and continued below
  at that rate.

  `flutes`, channels
  cut down faces steeper than `steep` (rise over run, sixty degrees
  where unsaid), deeper where more ground drains through; and `beds`,
  level ledges standing out of those faces, `spacing` metres apart in
  height. a process the set leaves out is not made. the run's tally says
  how many cells were laid finer and how far the made corners moved.

  made ground needs a camera, and is laid for the one the run draws.
  both tracers draw it: it is a mesh, as the survey is.

  slayer downloads nothing unless asked; a set whose terrain or moon
  face is not on the shelf fails and names the flag.

  `slayer draw FILE --fetch-terrain` fetches what is missing, checks it
  against sources.yaml and draws. `--refresh-terrain` fetches again and
  replaces a file only with a good one. `slayer assess FILE
  --check-terrain` downloads nothing, says whether the shelf is the
  source's and draws nothing.
- film: the picture's width, height and rays, its exposure, how wide it
  sees; or a rig, a frame, a lens and a shutter. and a print.
  filter, under film, says where in a pixel its rays start, and so how
  an edge is drawn. it costs no time whichever it is: a ray costs what
  it costs wherever in its pixel it starts. a pixel is the plain mean
  of its own rays under all four.
  - tent (the default): a pixel's rays are spread over it and a pixel
    beyond, thickest at its middle, in one pattern every pixel has.
    an edge fades across two pixels with no steps, and a line thinner
    than a pixel is as strong wherever it falls. the smoothest edge
    and the softest detail of the four.
  - trapezoid: the rays lie evenly over the middle half of the pixel
    and thin to none three quarters of a pixel from its middle. between
    the tent and the box, in edge and in detail.
  - box: the rays are spread evenly over the pixel's own square and no
    further. detail is as sharp as it can be drawn; an edge is even,
    and changes shade in one pixel where the tent takes two.
  - grid: each ray at the middle of its own square of the pixel, a
    regular grid. an edge along the grid has one more shade than there
    are rays along a pixel's side, five at `rays: 4`, and shows steps.
    this is how every picture was drawn before the key, and a set that
    says it draws those pictures again, to the byte.

  under the first three, sixteen rays draw an edge along the grid in
  seventeen even shades. what a wider filter costs is contrast in the
  finest detail. of the contrast the scene has, stripes one pixel on
  and one off keep 0.64 under the box, 0.57 under the trapezoid and
  0.41 under the tent; two on and two off keep 0.90, 0.88 and 0.81;
  four on and four off, 0.97, 0.97 and 0.95.

  at `rays: 1` there is nothing to spread, and the ray looks through
  the pixel's middle under all four. a draw through a gravity field, a
  restir draw and a wire picture do not read the key and draw the grid.
  sections, under film, says how its frames are drawn in pieces:
  - over: the width in pixels a frame must be wider than (320).
  - size: a section's side in pixels (64).
  - keep: whether a section that cannot have changed since the last
    frame is kept from it (true). false draws every section of every
    frame, to check the kept ones against.
  - drift: how far a kept section may be from the frame drawn whole (0):
    the share of a volume's light from the lamps that did not change
    that the lamps that did may come to. with a drift, a ring a far fire
    barely lights is kept; with none it is drawn again.
- cameras: each by its name. see cameras, below.
- moon: where it stands in the sky, by azimuth and altitude.
- sun: where it stands in the sky, said one of two ways, never both.
  - worked: say `location`, `date`, `timeofday` and `utc-offset`, and
    slayer works the azimuth and altitude from them (scene/daylight,
    the NOAA solar equations: geometric, without refraction).

    the clock is the one `timeofday` is read on, so `timeofday: 1725` with
    `utc-offset: {value: 8, unit: h}` is 09:25 UTC. a date with no
    location, no timeofday or no utc-offset is refused, and the refusal
    names what is missing.

    a `timeofday` under `light: sun` with no
    date, a `utc-offset` with no date, and a `sun` with no direction
    and no date are refused too. a sun that this puts at or below the
    horizon is not an error: the sun is down, it lights nothing, and
    the set has whatever other light it says.
  - typed: say `sun: {azimuth: A, altitude: B}`, in degrees, the azimuth
    clockwise from north. a sun with one of the two and not the other is
    refused.

  a scene on the ground that says a date and an azimuth or altitude is
  refused, and the refusal names both. under `light: moon` a date works
  a sun for its dusk. a moon scene whose date and clock time put that
  sun above the horizon is refused, and the refusal names the date and
  the hour and says a moon scene needs the sun below the horizon.

  in a scene in space a date beside a typed sun is read without that refusal. `sun: {tint: [r, g, b]}` goes with either. in
  a scene in space the date is read and no stage draws it yet.
- planet, hud: a planet seen from its air, and what is drawn over the
  picture. go doc ./set Planet and Hud say what they take.
- fantasy: what the scene states that the physics does not say. each
  key departs from the measured or derived value, and the render prints
  a line for each one. a flag of the same name given on the command line
  stands over its key. the keys:
  - lake: still water up to this elevation, in metres from zero; zero
    is no lake.
  - lakeglow: true shows the lake as a glowing mask of the water.
  - moon: the moon's disc this many times its size, above zero.
  - moonglow: the moon's disc gives this many times its light, zero or
    more.
  - giants: the rim's named trees this many times taller, above zero.
  - close: the close spruces this many times taller, above zero.
  - closeglow: true shows the close spruces as a glowing mask.
- world: the world piece by piece. see the world, below.
- set: the bounded box, the assets, the manufactory's things and
  stands, and the texts.

a key a block does not take is refused, saying the keys it does. a
typo should fail loudly, never draw a picture without the thing it
misspelled. a key said twice in one block is refused too: yaml forbids
it, and the reader would otherwise read both.

## places

a set's places are in the world's unit, as a map's run: x east, y
north, z up. the unit is a metre, unless the world says otherwise:

    unit: 10000000

makes it ten thousand kilometres, as the widow's is. the world's space
is y up with north its minus z, and the set hands places back in it, so
a set file never says the tracer's axes. a nought north is the world's
nought, never minus nought.

A manufactured thing may name an absolute coordinate separately from its
terrain-relative height:

    position:
      east: {value: 120, unit: m}
      north: {value: 180, unit: m}
      up: {value: 50, unit: m}

`position` requires all three named metre coordinates. It cannot accompany
legacy `east`, `north`, or `height`, nor the version 2 `place`, `orbit`, or
`around` placement forms.

On terrain it is rebased into the terrain window's
local frame; `up` remains an absolute elevation, while legacy `height`
remains a lift above sampled ground. In a space set the same coordinate is
placed directly in the scene frame. Sets without `position` retain their
existing placement behavior.

## boats on the water

the kinds `sampan`, `fishing-boat` and `ferry` are boats, made from a seed
(doc/model/boat/boats.md). a boat's origin is at its waterline, so a boat is set on
the sea by putting its `up` at the sea's height. one boat is a thing:

    things:
      a sampan:
        kind: sampan
        seed: 3
        position:
          east: {value: 400, unit: m}
          north: {value: 1800, unit: m}
          up: {value: 0, unit: m}
        heading: {value: 90, unit: deg}

a stand of a kind that floats, a boat or the Alfa, stands on the sea and
never on the ground: where the terrain is under the sea by three metres or
more, or by the stand's `least-water`, `{value: 0.45, unit: m}` for a
surface model whose water is flat nought under a sea half a metre up, at
the sea's height less the kind's waterline.

the sea is the
terrain's own `sea`, or, where the terrain says none, the height of the
middle of `world.ocean`. the stand's `above` and `below` are not read. each
of its things is drawn afresh, so that no two are alike, and is turned any
way. the habit `scatter` lays them over a rectangle:

    stands:
      anchored sampans:
        kind: sampan-at-anchor
        habit: scatter
        count: 6
        seed: 21
        area: {west: 2850, south: 2250, east: 3600, north: 2850}

or over a wedge, `within`, about a thing. a scene on no terrain, with the
sea of `groundless-air` and `world.ocean`, has no ground to read a depth
from, so its boats stand in a channel's water by the habit `frontage` (the
section after next), or one by one, as sets/alfa-san-onofre.yaml sets the
Alfa.

    middle: [-5, 0, 2.5]

is five units west of the middle, at the middle's northing, two and a
half up. a size is said the same way, east to west, north to south,
and up, and every side is above nought.

## rows of terraces on a slope

a stand of one of the terrace kinds (terrace-portofino, terrace-groningen,
terrace-midwest) on terrain stands each bay on the ground under it, on a
plinth that reaches the lowest ground under the bay. the habit `contour`
lays such a stand as a town climbing a hill: rows that follow the ground's
contours, one above the next, with a lane between.

    stands:
      the town:
        kind: terrace-portofino
        habit: contour
        count: 8
        seed: 7
        within: {of: the foot, from: 0, to: 360, near: 0, far: 400}
        climb:
          rows: 4
          runs: 2
          lane: {value: 4, unit: m}
          door: downhill

- `of` in `within` is a thing, and the first row starts where it stands,
  at its back corner. `far` is how far from it a run may start. `from`,
  `to` and `near` are said and not used by this habit.
- `count` is the most runs laid, and the schema asks for it: rows times
  runs lays every one. `rows` is how many rows climb the slope, one above
  the next. `runs` is
  how many straight terraces make a row, end to end.

  each run is a whole
  terrace of its culture, drawn from its own seed, and is turned to the
  contour where it stands, so a row bends where the ground does. where two
  runs meet at an angle the outer corner leaves a gap and the inner one
  overlaps.

  where a culture says no two neighbours share a colour, a run
  is drawn from another number if its first bay would be the colour of the
  last bay of the run before it in its row, so the rule holds across the
  joint as it does inside a terrace.
- `lane` is the gap between a row and the one above it, a lane or a stair
  as wide as the number says. the ground in it is the hillside; nothing is
  laid on it.
- `door` is the way the fronts, and so the doors, look: `downhill` to the
  fall of the ground, `uphill` against it. the floor of each bay is the
  highest ground along its front wall, so the door is at the ground
  outside it. a plinth of concrete carries the front down to the lowest
  ground under the bay, and the hill is cut into behind it.
- the ground says where the contours are, from the heights read 20 m
  either side of a place (`contour-span` in
  `manufacture/cities/slope.yaml`). ground flatter than two in a hundred
  has no fall line and is refused.
- a model of 30 m posts, a surface model, and bays under 6 m wide: the
  ground is smooth between posts, so several bays share one slope and
  step by the same amount, and what steps at a post is the whole row. no
  ground finer than the model is made up.

`sets/terraces-on-a-steep-coast.yaml` is the example, a proof of the
mechanism on a steep coast on this machine's shelf. `sets/portofino-cliffs.yaml`
is the same staging at a harbour whose ground is not here yet.

## rows along a channel

the habit `frontage` lays a stand along the line of a channel of the
world: terraces end to end with their fronts to the water, trees along the
quay, boats along the wall. it takes no `within` and no `area`; the channel
says where the walls are.

    stands:
      west bank:
        kind: terrace-groningen
        habit: frontage
        count: 12
        seed: 7
        complexity: demo
        frontage:
          of: canal
          side: left
          setback: {value: 5, unit: m}
          gap: {value: 8, unit: m}

- `of` is a channel by its name. `side` is `left` or `right` of its line,
  as one goes along it from its first point. `setback` is how far from the
  outer face of the channel's wall the fronts stand, away from the water,
  the width of the street between the coping and the houses; a negative
  setback puts the line on the water side of the wall, as a boat's. `gap`
  is the room between one run of terrace and the next.
- `count` is the most it lays. the runs are laid reach by reach, from the
  first point of the line for the left and from the last for the right.
  a terrace is a whole run of its culture, drawn from its own seed, and
  stands with its front on the line of fronts and its back corner a bay's
  depth behind it. a run that does not fit whole in what is left of a
  reach goes to the next, so a bend of the channel is a cross street.
- at a bend the line of fronts is mitred as the walls are. on the inside
  of the bend a run stops short of the corner by half a gap and by the
  depth of the run times the tangent of half the turn, so no two runs
  overlap; on the outside, by half a gap.
- a kind that is not a terrace is laid at the spacing its data gives, one to
  each stretch of that length (`manufacture/cities/frontage.yaml`): the
  alder and the poplar for the quay, the boats for the wall. a boat's data
  turns it to the line, bow along it.
- on a scene on no terrain the stand stands on flat ground at the scene's
  zero, which is where a thing with an `up` of nought stands. a boat
  stand floats at the height of the world's `ocean`, in a channel whose
  water is at least as deep as the most the boat draws (`manufacture.
  BoatDraught`); a channel with less is refused, naming both depths.

  no stand on such a scene may say `above`, `below`, `steepest`, `atmosphere`
  or `alight`, which need terrain.

## how steep the ground a stand grows on may be

a stand scattered over terrain, one with no habit, may say `steepest`: the
most the ground may rise in a metre of run where one of its things stands,
read along the ground's steepest way at the terrain's nearest post. with
`above` and `below` it sows a stand by height and slope together:

    recipes:
      hala:
        plant: {family: seasonal, file: ../manufacture/plants/seasonal/profiles.yaml, species: pandanus-tectorius}
    set:
      stands:
        hala a:
          recipe: hala
          count: 200
          seed: 611
          above: 3
          below: 610
          steepest: 0.7

the profile path is relative to the set; this example assumes the set is in
`sets/`.

- `steepest` is from 0, level ground only, to 10, past any cliff; unsaid,
  a stand grows on ground of any slope.
- a stand laid by a habit is refused if it says `steepest`: a habit lays
  its sites by its own rule and would not read it.
- a place is offered at random, as for `above` and `below`; a stand whose
  band and slope leave too little ground is refused after the tries run
  out, naming both.

## cameras

a camera stands off an asset and points to one, tipped up by alpha
degrees:

    road:
      orientation:
        point_to: the box
        alpha: 4.4
      location:
        loc_relative:
          object: the box
          standoff_x: 12
          standoff_y: -17
          standoff_z: 6.4

or stands at a place and looks at one:

    front:
      location:
        at: [11, -17, 6.4]
      orientation:
        look_at: [-1, 0, 8]

never some of each. up is the world's up, except for a camera looking
within a hair of straight up or down, which takes its heading for up.
one looking exactly up or down, or at where it stands, has no heading
and is refused when it is drawn.

A camera that is explicitly inside one or more clear materials names them
in entry order:

    underwater:
      initial-media: [seawater]

Each name must resolve to a clear material in the same set. The ordered
stack is explicit; Slayer does not infer containment from the camera's
position. The highest material rank is active at a boundary, with the last
entry winning a rank tie. At most 16 media may be named. This supplies the
clear-interface start state only; it does not add absorption, scattering or
a participating water volume.

Fixed-camera set renders carry this explicit stack through both CPU and Metal
tracing. A moving job camera is refused: one authored stack cannot follow a
camera that may cross a boundary.

Composite passes and Metal's gravity kernel
also refuse optical state, and named media are not combined with volume or
gravity transport. Neither backend infers the camera's medium or derives a
new stack as scene boundaries move. Without `initial-media`, the camera
continues to start in air as before.

The set-camera Metal route currently accepts only a pinhole.

It refuses a rig
or thin-lens aperture because those rays start across a physical aperture, and
the authored stack does not prove every origin is inside the named media.
CPU tracing retains its existing rig and thin-lens behavior.

## the film's print

    print:
      knee: 0.7
      top: 8
      source: eye

the knee is where the print's shoulder starts, as a share of white;
the top is the brightest light it holds, in whites, which lands on
white. the top is above the knee. the film's exposure multiplies the
light first.

## the world

every piece is by its name, and says every key it takes but the ones
said here to be its own to leave out.

- materials: a colour, or a glow, or both; source. a glow is a black
  body's light at a temperature in kelvin and a luminance,
  glow: {temperature: 2700, luminance: 0.45}, or a colour,
  glow: {swatch: tungsten}. A material may also say `clear`, a renderer
  dielectric surface:

      clear: {through: 1, index: 1.333, gloss: 1, rank: 0}

  `through` is the share passed by the surface (above zero through one),
  `index` its refractive index (1e-9 through 1e9), `gloss` scales its
  Fresnel reflection (zero through one), and `rank` chooses which clear
  surface contains a point where clear things overlap (signed 32-bit
  integer; zero is the renderer's default). Omit `rank` to use zero.

  The broad index bounds keep pairwise index ratios and their squares
  representable in both the CPU float64 and Metal float32 Fresnel and
  refraction calculations; they are numeric safety limits, not a
  material-property claim.

  Colour and glow may accompany `clear` and retain their ordinary surface
  and emission behavior. Colour is not absorption tint: this setting adds
  no wavelength-dependent loss, participating water medium, or underwater
  camera model. Rays still start in air.
- boxes: material, middle, size; heading, source. the material is one the
  world holds, or the set is refused. `size` is [east, north, up] in
  metres. `heading` is a turn about the box's middle, in degrees from 0
  through 360 clockwise seen from above, written `{value: 30, unit: deg}`; with none the box's
  width runs east and west. a run of boxes, each at its own heading, lays a
  quay along a shore that is not straight.
- channels: a cut in flat ground for still water, walled, by its name:
  line, width, depth, top, reach, ground, bed, wall, coping; source. a
  canal, a ditch, a quay or a moat.

      channels:
        canal:
          line: [[0, -300], [0, 100], [30, 300]]
          width: {value: 12, unit: m}
          depth: {value: 3, unit: m}
          top: {value: 1.5, unit: m}
          reach: {value: 60, unit: m}
          ground: paving
          bed: silt
          wall: {material: brick, thickness: {value: 0.5, unit: m}}
          coping:
            material: stone
            height: {value: 0.2, unit: m}
            overhang: {value: 0.1, unit: m}
          source: eye

  `line` is the middle of the cut, from two to 64 points of [east, north]
  in metres, a straight run or a bend. `width` is the clear width between
  the walls, `depth` how far the bed lies under `top`, and `top` the
  height of the ground and of the coping's upper face, flat along the whole
  channel. `reach` is how far the ground runs from each wall; past it the
  set lays what ground it wants, boxes for instance.

  each side has a wall of its thickness, a coping over it that juts out
  over the water by `overhang`, and the ground beside it in `ground`. the
  bed has a floor under it, and each end of the line is closed by a wall
  and a coping across it, one wall thick, past the end of the line.

  a bend is mitred, and a joint is refused if it turns past a right angle
  or if, for the reach, it slides past half of a segment beside it: the
  ground would fold over itself. the channel holds no water. the water is
  the world's `ocean` below, laid at a height of the set's choosing between
  the bed and the top, its square sheet at least as long as the channel.
- ocean: the world's sea from the wave cascades. An ocean that says no
  `extent` and no `samples` is the sea with no edge, which is what a picture
  asks for (described below). One that says both is a finite, static
  height-field square, a study road for the optical-study keys; one of the
  two without the other is refused.

  It says the name of a world material
  with a clear interface, a centre `[east, north, up]`, and every sea input,
  seed and time explicitly; the finite square also says its extent and
  samples per side:

      ocean:
        material: water
        middle: [0, 0, 0]
        extent: {value: 256, unit: m}
        samples: 129
        sea:
          wind: {value: 8, unit: m/s}
          fetch: {value: 1500, unit: m}
          depth: {value: 0, unit: m}
          toward: {value: 1.2, unit: rad}
        seed: 41
        time: {value: 2.25, unit: s}
        source: "wind study, stated inputs"
        direct-sun: mean-plane
        flat-mean:
          mean-level: {value: 0, unit: m}
          collar-width: {value: 64, unit: m}

  `extent` must fit within the largest source cascade; samples are from 2
  through 1024 per side. The field is a single instant and its coordinates
  are converted from east/north/up to Slayer's x/y/negative-z convention.

  World-only sets scale the metre geometry by `1 / world.unit`. A terrain
  backed set requires `world.unit: 1` so the world half and measured terrain
  share the same metre frame.

  The finite edge is real: rays outside it do not
  cross a water interface. The existing conservative height-field marcher can
  miss at grazing views. This adds no shore model, horizontal choppy
  displacement or coastal coefficient.

  A camera inside the field must still
  name the clear material in `initial-media`; the reader does not infer
  containment.

  `flat-mean` is an independent, opt-in CPU shape continuation. Both entries
  require explicit metre units.

  `mean-level` is relative to the ocean's
  authored middle elevation; `collar-width` is added on every side and cannot
  exceed the core extent (the shape implementation also bounds it to one core
  width). The core wave surface and samples are unchanged; a smooth collar
  damps the boundary cell toward that authored level.

  The continuation remains
  finite and may still show its outer boundary. It is a controlled flat-far-
  field approximation, not an infinite ocean, and Metal explicitly refuses it.
  The `direct-sun: mean-plane` CPU path accepts this exact wrapped boundary and
  traces its actual core or collar crossing; this does not extend its other
  light, medium, or atmosphere limitations.

  An ocean with no `extent` and no `samples` is the sea with no edge; the
  key `unbounded: true` may still be said and is no longer needed.

  The wave bands of the sea's wind (four, 4096, 512, 64
  and 8 m across, each repeating over its own side) are summed at every
  point of the plane, so there is no square to end and no flat plane
  beyond it. Every band is kept at every distance; far water shows the
  finest band's waves, and the bands repeat, the longest every 4096 m.

  Each band holds waves down to eight of its samples long and is read
  where its water is carried to (Gerstner's wave for one wave,
  Tessendorf's choppy sea for many, the displacement the waves' own), so
  crests are peaked and troughs flat; short waves ride the long ones.
  No white water is drawn on the open sea yet.

  A look mirrored by the sea is roughened by the slope of the waves
  shorter than the bands hold (Cox and Munk's total for the wind, less
  the bands' own) and of the bands a pixel is too wide to show; under a
  wind of one knot the sea is like a mirror and is not roughened. Every
  draw of such a sea prints the height, peak wave and slope asked, held
  in the bands and met by a ray.

  `swell` adds long waves from a distant wind to the sea, on either road:
  `swell: {height: {value: 2.5, unit: m}, period: {value: 12, unit: s},
  toward: {value: 1.57, unit: rad}}`, all three or none. `height` is the
  swell's significant height in deep water, `period` its peak period, and
  `toward` the direction it travels toward, in radians from east.

  Its spectrum has JONSWAP's shape, spread about its direction as
  cos^(2s)(theta/2) with s = 75 (Goda, 2000). The sea's significant height
  and its slope for a `cox-munk` glint include it. A swell is refused
  beside `motion`, which moves only the wind sea.

  Over a set's `terrain` the sea meets the ground, with no key of its
  own. Water lies only where the open sea reaches: from every edge
  sample of the ground below the sea's level (the ocean's `middle`
  height), over the samples below that level the ground's triangles join
  to them by an edge. A hollow below sea level that no such path reaches
  is dry.

  Off the ground's grid there is water everywhere, at the depth
  of the nearest edge sample. Over each place the waves are cut to what
  the depth there carries: a wave breaks at 0.78 of the depth
  (McCowan), so the surface rises and falls about its mean by at most
  half that, and not at all over land.

  The waves keep their deep-water
  lengths and directions: there is no shoaling, no bending toward the
  shore, no set-up, no run-up and no foam. A channel narrower than the
  ground's step joins nothing.

  Over land the surface is the mean plane,
  under the ground and hidden by it from above; a ray that reaches it
  some other way meets it, so "water lies only where the open sea
  reaches" is true of the picture and not of every ray. Off the ground's
  grid beyond a land edge the water is flat at the mean.

  A refracted
  sun ray (`direct-sun: mean-plane`) from a point under shore water
  that crosses a dry column before the mean plane finds no boundary,
  and the draw is refused.

  The draw says, in one line, how many land
  samples the open waves stood over, how many samples lie in hollows
  cut off from the sea, and how many of those the open waves filled.
  A terrain with a near window is refused under an unbounded sea.

  A ray meets it by stepping the gap over the most the gap can change
  along the ray, which the bands' steepest rise bounds, and over ground
  the ground's steepest rise where the waves are cut, so a ray that
  skims the waves does not pass through a crest.

  The GPU takes the same
  steps in single precision, so its touch grows with distance, a
  millionth of the way to the point, as its height fields' does, and
  draws a `cox-munk` glint by Beckmann's lobe as the CPU does. The GPU's
  sea has no coast: over ground it draws the open sea's waves, and the
  ground stands over them where it is higher.

  The sea's glint is Cox and Munk's unless the set says
  `glint: {model: artistic, ...}`; `glint: {model: cox-munk}` takes the slope the bands
  do not resolve (Cox and Munk's total for the wind, less the slope the
  bands carry down to the finest band's Nyquist wave number).

  It refuses
  the keys that belong to a square: `extent`, `samples`, `flat-mean`,
  `shading-normal`, `surface-wake`, `interface-events`,
  `diffuse-sky-samples`, `clear-cloud-sky-samples`,
  `clear-cloud-water-step-mode`, `in-scatter`, `direct-moon` and
  `motion`.

  `direct-sun: mean-plane` is the only optical approximation it
  takes, on the ordinary ground route. A shadow ray from the ground
  passes a clear sea, so without `direct-sun` the ground under it is lit
  as if there were no water. sets/honopu.yaml and sets/rarotonga.yaml use it.

  `glint` is the sun's highlight on the sheet: the roughness of the water
  below one height-field cell, which the sheet's normals cannot carry. An
  ocean that says no `glint` is drawn exactly as before. A glint names its
  `model`:

      glint: {model: artistic, roughness: 0.3}

  `artistic` is a look chosen by eye, not a measured sea. `roughness`, above
  0 and at most 1, becomes the sheet's own copy of its material's satin, so
  the tracer's GGX highlight lobe has width roughness squared. Other things
  wearing the same material keep theirs. The receipt says `glint=ARTISTIC`.

      glint: {model: cox-munk}

  `cox-munk` derives the glint from the sea, and takes no roughness. The
  total mean-square slope is Cox and Munk's 1954 clean-surface fit for the
  sea's wind, 0.003 + 0.00512 U (0.0388 at 7 m/s); the wind is read as
  theirs at 12.5 m, though the sea's is at 10 m.

  The resolved part is the
  spectrum's slope, the integral of k squared times the spectrum over the
  wave vectors within the sheet's Nyquist wave number, pi over its spacing,
  each way.

  The unresolved part is the total less the resolved, floored at
  0.0001 (oceantex `slope.floor`). It is the mean-square slope of a Beckmann
  highlight lobe, whose Gaussian slopes match Cox and Munk's first-order
  form exactly.

  The receipt gives all three, and says so when the spectrum
  resolves more than Cox and Munk's total. Both tracers draw it.

  `direct-sun: mean-plane` is a separate, explicit CPU approximation for
  direct sunlight on surfaces inside the named ocean medium. It computes
  refraction and Fresnel transmission with the horizontal mean-plane normal,
  uses the sampled finite wave surface as the water-path endpoint, and applies
  the square-on irradiance projection factor and extinction over that path.

  It supports the ordinary ground `light: sun` route and the explicitly
  referenced groundless Earth-air route, with one fully clear, named,
  extinction-only ocean boundary. The ground route tags its sun and distinct
  Distant fill; groundless air tags the exact single Distant inserted by
  `aired`, retains the air-computed sky separately, and has no synthetic fill.

  Additional directed lights, other clear boundaries, volumes, gravity
  fields, separate haze, other air routes, space and world-only film-sun
  staging are refused. Existing directionless `film.Flood` fill remains a
  separate contribution.

  Opaque blockers are tested on both
  sides while the ocean sheet alone is skipped.

  A crossing outside the finite
  ocean footprint, or a height-field marcher miss, refuses the render.

  This
  approximation uses wave height only for the segment endpoint; it does not
  use wave normals, model caustics, transport sky/flood fill through water,
  add scattering or support Metal.

  See
  [the direct-sun limitations](model/ocean/ocean-mean-plane-direct-sun.md).
- spheres: material, middle, radius; source. and motion, the name of
  the motion that carries it, from its middle; a sphere with none
  stands still.
- motions: seconds, ease, hold, frames. a run through keyframes, each
  frame {percent: P, at: [x, y, z]}, in order. ease is one of css's
  curves, linear to ease-in-out, eased once over the run; hold keeps
  the last place after, and without it the thing goes back.
- lamps: at, temperature, strength, radius; source. a bulb, shining as
  a black body; strength is its luminance at a unit, and inside its
  radius it does not light.
- spots: at, aim, cone, soft, temperature, strength; source. a lamp
  kept to a cone: cone is its half-angle, soft how much of it the
  light takes to fade at the edge, both in degrees.
- beams: at, aim, radius, reach, divergence, temperature, strength; source.
  A finite Gaussian light column. `at` and `aim` are world coordinates and
  direction; radius is the source-plane 1/e² intensity radius, reach is its
  finite axial length, and both distances use world units.

  `divergence` is
  the geometric half-angle in degrees, from 0 up to but not including 90.
  Temperature and strength make a black-body source; strength is its peak
  source-plane luminance and spectrum.

  The beam has no visible skin and only
  appears where surfaces or participating media scatter its light. Its radius
  and reach are passed to the emitter in the same numeric world-coordinate
  units (not multiplied by `world.unit`).
- floods: colour, strength; source. light from everywhere alike, its
  colour so many times over.
- sky: horizon, zenith, ground; source. or stars, {seed, share, size,
  bright}; source. one a world.
- bodies: template, middle, detail: a body from scene/bodies/template
  by name, its moons' ground tiled detail deep. its own to leave out:
  radius in the world's unit, else its true size; tilt in degrees,
  else its template's; aurora false to hide its template's.
- suns: toward, colour; source. a sun so far off its light comes one
  way. toward is the way to it, three numbers with no unit.
- wisps: colour, middle, radius, thick, seed, scale, amount; source. a
  ball of gas thinning from its middle to its radius, ragged by noise:
  seed makes it its own, scale sizes the wisps, amount is how much of
  the density they take, colour what it gives back of the light.
- fizzes: colour, foot, radius, height, thick, rise, churn, seed;
  source. the mist of bubbles too small to see: a column on its foot,
  radius wide there and twice that at its top, height tall, thick at
  the foot's middle and thinning to nothing at its side and its top.

  churn is how much of the density the water's churn takes, nought to
  one, and rise how fast the churn climbs, in units a second: a fizz
  drawn at two moments is two pictures. with a churn of nought or a
  rise of nought it stands still. `clear-water: true` opts it into CPU
  segment-wise clear-water composition. sets/fizz.yaml is one.
- vapor-cavities: colour, origin, axis, tip-radius, rpm, slipstream-mps,
  length-m, core-radius-m, density-per-m, pressure-deficit-pa,
  recovery-pa-per-m, visual-onset, clear-water; source.

  This is a single
  analytic volume with five phase-locked tip-vortex cores.

  The pressure
  deficit and recovery gradient are explicit visual controls; `visual-onset`
  must be true to turn the field on. `clear-water: true` opts into CPU
  segment-wise clear-water transport. It is not a cavitation solver.

a colour is linear light in srgb's primaries, each part not below
nought, or one of slayer's swatches by its name:

    colour: {srgb_linear: [0.34, 0.11, 0.05]}
    colour: {swatch: rust}

the world is built in the order the file writes its lists, and each
list in its own order, so the same file gives the same world.

the radiation of a lamp's black body comes from celestial, carried as
a spectrum, and its colour from libtheme, through slayer's observer. a
spot's black body is libtheme's colour of one at that temperature, as
the css sheets had it, and has no spectrum.

## the air: mists, clouds and plumes

`weather` names mists, and the `wind` that carries them. A mist is
`kind: prescribed-mist`: an extinction per metre where it is thickest, a
depth, a coverage and a seed, and its `asymmetry`, the lobe it scatters
with. `atmosphere: layers` lays each one over the terrain, from the ground
or from a level `base`, with the share of what it meets it scatters, its
`albedo`. A mist may also say what it is made of and how it lies:

    smog:
      kind: prescribed-mist
      extinction: {value: 0.004, unit: m^-1}
      depth: {value: 450, unit: m}
      edge: {value: 60, unit: m}
      coverage: 1
      seed: 3
      asymmetry: 0.63
      angstrom: 1.4
      nitrogen-dioxide: {value: 300, unit: ppb}

- `angstrom` is its particles' Angstrom exponent: their extinction goes
  as the wavelength to its minus, from the green's at 550 nm. Unsaid, the
  mist is as thick to every colour, as before.
- `nitrogen-dioxide` is that gas mixed through the mist where it is
  thickest, in parts in a thousand million. It absorbs and does not
  scatter, most in the blue: 0.33 inverse megametres a part at the green
  channel (the revised IMPROVE equation), three times that at the blue and
  0.45 of it at the red, which two are not yet read from a measured
  spectrum.

  The layer's albedo is then its particles' share of what each
  channel meets, so a smog is brown from the side and the sun through it
  is orange. Its extinction is the particles' alone, at 550 nm.
- `edge` holds the mist's whole density up to that far under its top and
  thins it to nothing at the top, as air does under a lid. Unsaid, it
  thins from the bottom up.
- `size` is the size of its patches where the coverage is under one;
  unsaid, its depth.

A layer may say how far past the terrain's edge it reaches, each way,
`beyond: {value: 3000, unit: m}`, so its edge is not in the picture; the
ground's edge height holds out there. It may say the thing it is thickest
over, `toward: {object: the city, reach: {value: 1500, unit: m}, least:
0.3}`: it thins by a smooth step from there to the least of itself at the
reach and past it.

A cloud deck is `kind: prescribed-cloud`: an extinction, a `base` and a
`top` (scene heights in metres, up to 20 km), a coverage, a seed, a
`size` and an asymmetry. It says no depth and no edge.

A layer names it
as it names a mist. Noise cut at the coverage makes separate bodies with
clear sky between, and the share the bodies take narrows from the
coverage at the base to nothing at the top, so each has a flat base and
a rounded top.

A layer of cloud may say `toward` and `beyond` as a mist's
does: toward a thing its cover is greatest, and away from it the bodies
are fewer and smaller, so a deck can stand over one side of the sky and
leave the other clear for a low sun to come under it.

The sun lights it through the same march as a mist: a lit
side and a side in its own shadow, by single scattering only, so the
shadowed side is darker than a real cloud's, whose light scatters many
times.

`plumes` is smoke or steam from a mouth, carried down the weather's wind,
which the set must say:

    plumes:
      ferry stack:
        from: {object: the ferry, east: {value: 0, unit: m},
          up: {value: 9, unit: m}, south: {value: 0, unit: m}}
        mouth: {value: 0.4, unit: m}
        exit: {value: 8, unit: m/s}
        temperature: {value: 620, unit: K}
        air-temperature: {value: 293, unit: K}
        rate: {value: 2, unit: g/s}
        mass-extinction: {value: 10, unit: m^2/g}
        albedo: 0.2
        asymmetry: 0.63
        angstrom: 1
        reach: {value: 800, unit: m}
        lid: {value: 450, unit: m}
        source: "what the numbers are and where they come from"

- `from` is an offset east, up and south of a thing's origin, or of the
  scene's where it names none. `count` lays that many mouths by `seed`
  within `spread` of it, each standing `up` over the ground under it.
- The mouth's radius, the gas's `exit` speed and `temperature` and the
  air's make Briggs's buoyancy flux, g w r squared (T - Ta) / T. The
  plume's middle rises by Briggs's bent-over law, the cube root of 3 F
  x squared over 2 times 0.6 squared u cubed, until it meets the `lid`, a
  scene height; over the lid it holds nothing, and what would be there is
  laid back under it.
- Its section is a gaussian: the rising plume's radius (the mouth's and
  0.6 of the rise) over root two, joined with Martin's fits to the
  Pasquill-Gifford spreads of neutral air. What leaves is carried through
  each section: on the middle the density is the rate over two pi, the
  wind and both spreads, times the `mass-extinction`.
- `albedo`, `asymmetry` and `angstrom` are its particles' optics; soot
  absorbs most of what it meets, steam almost none.
- Each plume is cut into two stretches down the wind, the first an
  eighth of the reach, each its own volume, so the march near the mouth
  is finer where the plume is thin. Every volume is tried by every shadow
  ray of every step of a march, so each mouth costs every ray of the
  picture: a few dozen mouths are a long draw on the CPU.

The CPU draws all of this. The GPU's boxed media take the mists' density
from the CPU on a grid; the cloud deck, the gas's tint, a layer's lid and
the plumes have not been tried there.

## units

numbers are bare, and each key's unit is fixed: the world's unit for
places, sizes and radii, kelvin for temperatures, degrees for angles,
seconds for a run. a share, a strength or a way has no unit.

## source

every piece whose values stand for something real says where they
came from. source: eye means judged by eye, a placeholder. a measured
value names its measurement, quoted:

    source: "a thermocouple, 2026-10-05"

a bare word that is not eye says nothing and is refused. a body, a
motion and a camera have no source: where saturn hangs, how a thing
moves and where a camera stands are choices, not stand-ins for
anything measured, and a body's facts come from its template.

this is the caveat kept as data, so a placeholder can't pass for a
measurement once the comment above it has scrolled away.

## how it reads

1. parse: yaml, every node keeping its line and column.
2. keys: each block's keys read by a step that checks its value and
   refuses with the place it was written.
3. found: a name one piece uses must be another's: a camera's asset, a
   box's or a sphere's material, a sphere's motion.
4. built: the package each part belongs to builds it from the set. stage builds
   the world for the tracer, and refuses what the tracer cannot hold: a
   black body too cold to see, a box too thin to have faces, a comet at
   its own sun.

## limits

- lights have no real units. a lamp's strength is the kit's relative
  light, not candela, and a glow's is not candela per square metre.
  an exposure can't be an aperture, a shutter and a film speed.
- a set file does not place its assets. house -at says where one
  stands.
