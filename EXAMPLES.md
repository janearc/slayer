# What Slayer draws

The numbered smoke scenes follow Slayer's rendering milestones. The larger
scenes show how those capabilities combine: a lantern in space, an orbital
landscape, a hillside, and a moonrise.

For a first look, [the demo menu](doc/demos.md) provides `game run demos`
and `game run demo N`, with PNGs and optional terminal output. The camera demo
also writes a short sequence from a checked YAML path.

Run `game build` from the checkout root first. Images go to `out/` unless
an output flag specifies another location. Application arguments after
`--` require Game's forwarding fix; the [workflow](doc/developing.md)
explains setup and the fixed runs available on older Game builds.

For architecture, see [OPERATION.md](OPERATION.md).

## Smoke scenes

Scenes live in `scene/smoke`. Their numbers remain stable so regressions can
be traced to a capability. Add new scenes rather than renumbering passing
ones; changes to existing scenes belong in separate, explained commits.

```sh
game run smokecheck -- smoke          # All scenes.
game run smokecheck -- smoke 9        # One scene.
game run smokecheck -- smoke 9 -out /tmp/nine
```

`-out` copies the terminal image, raster image, and comparison result to
the chosen directory. Cached results are keyed by the rendering binary;
`-fresh` bypasses the cache.

### Surfaces and direct light

1. **A ball on a floor.** A clay ball on chalk under one sun, with one
   shadow: the first render.
2. **Two lamps.** Red from the left and green from the right combine to
   yellow on a white ball. Each lamp lights the other's shadow; the region
   reached by neither remains black.
3. **A ball on a set.** Chalk and slate ground, a gradient sky, and haze
   that blends the distant floor into the horizon.
4. **RGB lighting.** Three lamps make white where their light overlaps
   and cyan, magenta, and yellow in the complementary shadows.
5. **A spotlight.** A warm tungsten pool with a fading edge and hard
   shadow, filled by a cool flood so shadows retain colour.

### Hexagons and terrain

6. **Hexagonal tiles.** Sea and moss on a sphere, with dark boundaries;
   sky tiles blend into the underlying sky at their seams.
7. **A planet.** Plate-based land, coasts, and polar snow on hexagons,
   suspended above the set and casting a shadow.
8. **A floor and ring.** Chalk-and-slate hexagons cover a floor and an
   upright ring, with flood light revealing its interior.

### Volumes

9. **Smoke.** A white puff beside the clay ball, with the floor visible
   through its thin edges.
10. **A gas giant.** Density increases below cloud tops until light no
    longer passes. Bands, turbulence, and a storm remain visible, with
    transmission at the limb.
11. **A ringed giant.** A tilted planet and icy rings shadow each other.
12. **Surface features.** A finer version of scene 7 with a volcano,
    canyon, ice shelf, and plateau placed on suitable terrain.

### Cameras and scale

13. **A long lens.** The ringed giant from 60 units away through a 200 mm
    lens, showing the flatter perspective of a distant viewpoint.
14. **Cloud detail.** A frame-filling giant with fine turbulence, jets,
    mottling, a large storm, and white ovals.
15. **The giant planets.** Jupiter, Saturn, Uranus, and Neptune at their
    relative sizes, including rings and Uranus's axial tilt.
16. **The rocky worlds.** Mercury, Venus, Earth, Mars, and Pluto with
    Charon; the last pair orbits its common centre.
17. **Orbiting moons.** Three satellites timed from the primary's mass
    using Kepler's law, viewed above the orbital plane.
18. **Hubble at Jupiter.** A narrow telescope view with four point-like
    moons, including a transit and its shadow. See the documented camera
    approximation in `film/lens.go`.
19. **A satellite view.** Fine Earth detail grows from a coarser world
    whose coastline stays fixed, without storing intermediate levels.

### Bands, film, and energetic objects

20. **A magnetar behind a giant.** False-colour gamma and X-ray renders
    show different transmission through a planet and moon.
21. **Radar Venus.** Surface backscatter seen by radar; synthetic-aperture
    processing is not modelled.
22. **Super 8.** A ringed giant fills a visibly grainy frame.
23. **IMAX.** The same scene through a 50 mm lens on a larger gauge,
    giving a wider field and finer apparent grain.
24. **Jupiter from Io.** A ground-level view past a volcanic plume that
    moves with Io; only camera placement is specific to the shot.
25. **A quasar.** A black hole bends visible light. The X-ray path omits
    gravitational fields, so that image has no lensing.
26. **A blazar.** Looking down a jet with 200-fold beaming, whose bright
    end overwhelms the disc.
27. **A pulsar.** Crab-like beams tilted 45 degrees from the rotation
    axis sweep around about 30 times per second.
28. **A magnetar flare.** A faint blue visible source becomes a bright,
    nearly transparent fireball in gamma rays.
29. **The Pillars of Creation.** Emission-line light mapped through
    Hubble's palette.

### Earth and beyond

30. **Earth at night.** Cities and clouds on the dark side. A daytime
    test checks that city emission fades and clouds shadow the ground.
31. **The Bullet Cluster.** Visible galaxies, X-ray gas, and a separate
    mass-colour layer. The mass uses a thin luminous volume as an explicit
    illustration, not a physical mass-lensing calculation.
32. **Aurora from orbit.** Curtains in an oval around each magnetic pole.
33. **A world beside a magnetar.** The Earth aurora model with exaggerated
    settings, rendered in X-rays.
34. **An escaping atmosphere.** A hot Jupiter about 40 planetary diameters
    from its star trails gas away from the star and back along its orbit.
35. **Hale-Bopp.** A slightly green coma, broad curved dust tail, and
    straight blue ion tail near the Sun.

### Lensing and reflection

36. **The photon ring.** A black hole lenses the far side of an accretion
    disc above and below its shadow. Nine rays per pixel resolve the
    thin ring where rays orbit near the hole.
37. **Falling in.** An infalling camera sees the sky crowd forward and
    the disc's light shift.
38. **The Blue Marble.** Earth from an image map, Africa facing the camera
    and sunlight from the upper right. This scene reads
    `$SLAYER_MARBLE/blue.jpg`.
39. **Glass reflecting sky.** Fresnel reflection rises from about 5% at
    near-normal incidence to about 80% at a grazing angle.

## The lantern

```
game run swarm
game run swarm -- -gpu -wear 0.5 -out out/lantern.png
```

A star wrapped in a Dyson swarm: thousands of small mirrored panels,
each on its own tilted orbit, all facing in. The star's light escapes
between the panels in shafts, like light through a pierced tin lantern,
and catches in a thin warm haze round it.

The panels are set in rings. A ring that turned only about its own pole
would leave its gaps in one place for ever, so each ring also tumbles,
and the shafts sweep round.

With wear the panels are battered: bitten at the edges, dented,
punched through. They can be black glass inside and out, or matte.

`swarm` draws it alone, from a little above the plane of the rings, 26
star radii out. The star is a hot body at 4800 kelvin, with faint
stars behind. `-gpu` draws it on the graphics card.

The flags set the number of rings and panels, the wear, the glass, the
spread of the rings, how fast and narrow they are, and the eye's place.

The lantern is a package, so any scene can place one: the orbital
scene puts one at its middle. doc/the-lantern.md tells how it came to
be, day by day.

## The orbital scene

```
game run orbital
game run orbital -- -far
game run orbital -- -cells 120x40 -frames 0
```

Two Culture orbitals round a lantern, inside a nebula. The orbitals are
two great bands sharing an axis, turning opposite ways so their spin
cancels, each leaning 45 degrees from the lantern's axis.

The outer band is at the radius where its floor feels one g, and the
inner is four fifths of that.

Their floors are tiled in hexagons painted with seas, coasts, deserts,
forests, rivers running downhill, lakes, volcanoes, and cities on the
coasts and rivers that glow warm or neon at night. The walls facing the
gap carry the lights of cities.

- **The near view**, the default, puts the eye inside, 500 km above
  one floor near the gap, looking along it as it curves up into an arch.
  A far sun lights it from one side, so the floor near the eye is in
  day, and further round the arch turns to night and the cities light.
- **The far view**, with `-far`, puts the eye 3.6 million km out, 25
  degrees above the plane of the rings, looking at the middle. It is lit
  only by the lantern: a 9,000 km star in twelve rings of worn panels
  of blue-black glass.
- In the far view the inner rings are wide and slow, the outer narrow
  and quick. Two beams sweep from the star's poles, thirty times its
  light, and the nebula is swept clear along the rings' path.

The orbital can draw a film as numbered PNGs, with `-frames` and
`-fps`, and resume one with `-first`. With `-cells` it streams frames
to the terminal instead; with `-keys` as well, it draws the next frame
at each key press.

It draws on the CPU only, because the GPU cannot yet draw its tiled
textures, its nebula or its beams.

doc/the-orbital-scene.md tells the story of the scene.

## The ring

```
game run ring
game run ring -- -dress -night -bands
```

A ring of triangles hung round a flaring magnetar, "the kind of thing
the Culture would hang round a flaring star for the fun of it". The
ring is 10,000 km across, mirror on its inner face and dark outside.

The eye is outside and above, looking in across the near rim at the
mirror on the far side.

The magnetar, SGR 1806-20, flares one second in, and the fireball
reaches the ring about a third of a second later. The picture is drawn
by default at 1.2 seconds, as the flare lights the mirror. A small far
sun gives the hull its shape.

- `-dress` tiles the inner face in hexagons, painted with rivers,
  lakes, volcanoes and cities, and cast into triangles.
- `-night`, with `-dress`, puts the sun out, so only the cities and
  the lava glow.
- `-bands` draws a second picture beside it in false colour: x-ray in
  green, gamma in violet, infrared in deep red. The ring has no x-ray
  or gamma of its own, so in those it is a dark shape.

## The hillside

```
game run hillside
game run hillside -- -night -alight 1
game run hillside -- -set sets/kilauea-iki.yaml
```

A hillside of things from the manufactory: spruces, ferns, boulders,
cobbles, logs, a can, a campfire and a spotlight, on dark, weathered
basalt under a painted sky.

Each thing grows from its own seed at a seeded place, so the same flags
and seed always draw the same picture.

- By day the sun lights it. By night the moon is faint, and the
  campfire lights the ground a few metres round it.
- `-alight N` sets N spruces on fire, with flame and smoke from
  measured numbers for burning trees.
- `-above` looks down from high over it, and `-from M -tilt D` looks
  down from M metres at D degrees.
- `-variety N` makes only N of each kind and reuses them, so a large
  forest keeps few faces. A command refuses more than `-faces`
  triangles in all, a million and a half by default.
- `-gpu` draws on the graphics card.
- `-prove` draws nothing. Given the flags that drew a raw, it draws 64
  of its pixels again from scratch and proves each to the bit.

With `-set`, hillside stages a whole set file instead. A set can name
measured ground, such as Kilauea Iki from the USGS lidar, with a full
moon or twilight, weather, and a hud drawn over the picture.

The sets are in `sets/`: three of Kilauea Iki (alone, with a forest,
and at moonrise), melancholia, planetary turbidity, the shoebox, and
three of the black widow. [doc/sets.md](doc/sets.md) says what a set
file may say.

## Moonrise

```
game run moonrise -- -out out/moonrise.pfm -preview out/moonrise.png
```

A full moon rising behind one tree, through a very long lens. The tree,
a Norway spruce by default, stands on the earth. The sun is low behind
the eye.

The eye is 1.7 metres up and 1,100 metres back, looking at the moon a
third of a degree above the horizon, at its true size and distance.

The lens is two degrees wide, about a 2,000 mm lens on IMAX film, and
the frame is IMAX's shape. The picture is kept as a raw, so it can be
developed later.

Where the moon stands and the earth's colour are picked, not measured.
The light is white, since reddening is the air's doing and the tracer
has no air here.

## The house

```
game run house
game run house -- -camera road,garden -at "the box=-1,0,0"
```

The shoebox: a house under a sky with Saturn, drawn from its set file.
`sets/shoebox.yaml` holds the world, piece by piece, its cameras, and
its film and print.

One camera is drawn whole, the front by default, and its raw is kept
beside the picture. Several are drawn side by side on one page; `-at`
places the asset a camera stands off.

## Weather

```
game run weather -- -gpu
```

A planet's weather run from still air, drawn every so often as it
grows. The defaults are a gas giant's: 60,000 km in radius with a day
of five hours.

Storms stir it from below and friction slows it, and jets and vortices
grow by themselves, as they do on Jupiter. Each step it says the
fastest wind, and how much of the energy is in the jets. Jupiter's
share is 0.93.

Each moment it writes three pictures, each a map of the whole planet:

- The air's spin;
- Its bands;
- The air's age, which turns haze from white toward brown.

## Other commands

- **Slayer**: `game run slayer -- set sets/widow.yaml` draws a set
  file as a stranger would, from outside, and checks its terminal
  picture against a real one.
- **Develop**: `game run develop -- -out x.png a.pfm b.pfm` adds raws
  together and develops them at an exposure, with an optional white
  balance in kelvin.
- **Sources**: fetches and checks the published sources. OPERATION.md
  has the details.
- **Lightbox**: `game run lightbox` serves the smokes folder at
  http://localhost:8765.
