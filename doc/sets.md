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
refused by hillside by name, with the file, the key and where it stands,
before anything is staged.

two are drawn. `setting: space` is drawn by hillside's stage for a scene
in space (hillside/afloat.go), and a film's `fps` is read and has no use
in a still. sets/webb-at-l2.yaml is the example: smoke 50 as a set.

a scene in space:

- has no ground and no air. its sky is black unless its world half says
  one.
- stands each thing, a kind the manufactory makes, at its east, north
  and height in metres from the scene's origin, and turns it by its
  heading or not at all.
- is lit by `light: sun`, one distant lamp from `sun:` at any altitude,
  as bright as sunlight at the earth at the film's exposure; or by
  `light: none`, no lamp.
- is seen by a fixed camera, `location.at` and `orientation.look_at`,
  or one placed off a thing.
- refuses by name stands, texts, ecosystems, fire fronts, fuel beds,
  layers of air, a planet, a burning thing and a recipe.

every other version 2 key is still refused by name: the date, the
frame, an anchor, a globe, a sky of stars, an orbit, an attitude, an
eye, a camera's moves and duration, and the kinds point, star, lantern,
culture-orbital and nebula.

## what a set says

everything is under scene:

- name: the scene's name.
- light: ambient, moon or sun.
- location: lat and lon, in degrees. timeofday: hhmm, on a local clock.
- terrain: the ground, a window of a published elevation model. the
  set names it: `from` is the dataset's key in sources.yaml, `file` the
  file of it that is read from the shelf.

  slayer downloads nothing unless asked; a set whose terrain or moon
  face is not on the shelf fails and names the flag. `--fetch-terrain`
  fetches what is missing, checks it against sources.yaml and draws.
  `--refresh-terrain` fetches again and replaces a file only with a
  good one. `--check-terrain` downloads nothing, says whether the shelf
  is the source's and draws nothing.
- film: the picture's width, height and rays, its exposure, how wide it
  sees; or a rig, a frame, a lens and a shutter. and a print.
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
- moon, sun: where each stands in the sky, by azimuth and altitude.
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

    middle: [-5, 0, 2.5]

is five units west of the middle, at the middle's northing, two and a
half up. a size is said the same way, east to west, north to south,
and up, and every side is above nought.

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
  glow: {swatch: tungsten}.
- boxes: material, middle, size; source. the material is one the world
  holds, or the set is refused.
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
  rise of nought it stands still. sets/fizz.yaml is one.

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

## open

- lights have no real units. a lamp's strength is the kit's relative
  light, not candela, and a glow's is not candela per square metre.
  until they are real, an exposure can't be an aperture, a shutter
  and a film speed, the way a stock was rated.
- a colour is srgb_linear or a swatch. oklab is the next way to say
  one.
- a spot and a lamp make a black body's light two ways, libtheme's
  colour and celestial's spectrum. one way for both would move the
  widow's pictures, so it waits for a look at them.
- a set file does not place its assets yet. house -at says where one
  stands.
