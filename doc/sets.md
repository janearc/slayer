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
- boxes: material, middle, size; source. the material is one the world
  holds, or the set is refused.
- ocean: one finite, static height-field sample from the existing wave
  cascades. It says the name of a world material with a clear interface,
  a centre `[east, north, up]`, a square extent, samples per side, and
  every sea input, seed and time explicitly:

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
  resolves more than Cox and Munk's total. CPU only: the GPU refuses it.

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
  [the direct-sun limitations](ocean-mean-plane-direct-sun.md).
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
