# Generating stars, planets and systems

## Authored atmospheric shell

A generated rocky world may set
`air_look: {height_km: 8.5, loss_per_km: 0.003, colour: [0.4, 0.6, 1], made_up: true}`.
These are artistic values for the existing shell: scale height in km,
surface extinction per km, and linear RGB scattering tint. Omission keeps
the derived default, and the draw states authored values. The shell has an
exponential vertical density and is truncated after six scale heights; it
is not a gas-composition model. It can be authored on a rock whose derived
air would otherwise be absent. Giants use a separate gas model.

Bruneton and Neyret's 2008 atmosphere work and Bruneton's documented
reference implementation show why density, wavelength-dependent molecular
scattering, aerosol extinction and absorption are separate inputs:
https://ebruneton.github.io/precomputed_atmospheric_scattering/atmosphere/model.h.html .
The single Slayer shell exposes fewer quantities and does not implement that
model, its spectral separation, phase functions or multiple scattering.
These controls are art direction, not an inference of atmospheric chemistry.
The original no-air comparison showed that the existing shell materially
obscured a low-angle view in one scene; it does not establish the physical
cause or justify removing atmospheres. Compare same-camera air settings at
ground, horizon and orbital scales and judge the actual images.

Status: design, 2026-11-26, with direction added 2026-11-27. Parts of
the planet and binary generation are now built; the sections below also
describe work that remains a design. It records the original system direction agreed the night of 2026-11-25, so the code can follow it.

## Authored cloud shells

Generated rocky planets can take an optional `clouds` block inside
`generate`, for example
`clouds: {base: 2, top: 9, cover: 0.6, loss: 0.5, made_up: true}`.
Heights are kilometres above the surface, cover is a share from zero to
one, and loss is extinction per kilometre at a cloud's heart. Zero cover
removes the layer. Omission retains the existing derived default. The
override can place a shell even when the water or air derivation would not
make one; it is an explicit artistic choice, not a weather or chemistry
prediction. Gas giants use their own gas model and refuse this override.
The set declares `made_up: true` and the draw prints the values.

The optional `weather: bands` chooses the existing latitude-based weather
for an authored shell, including on a locked planet; it does not change
the planet's rotation or orbit. Omitting it keeps the body's existing
weather, including the substellar blanket on a locked planet. That model
adds weather to the requested cover, so the cover value is not the final
fraction of the disc: in particular its authored blanket contributes 1.2
at the substellar point before the requested cover and storm hearts.
Choosing between these patterns is art direction, not a claim that a
locked planet's atmosphere behaves like a rotating Earth's.
The separate `weather: soft-bands` option keeps that latitude pattern but
reads its fine billow before the storm's turn. This is an artistic choice
to explore a softer treatment of whorls; it leaves the broad weather and
the established `bands` and default appearances alone.
An optional `storm_turn` sets the turn at a storm's centre in radians,
from zero to ten; omission retains the chosen weather's existing value.
The set declares these numbers made up. A zero-turn visual control removed
the concentric whorls but also removed the visible vortices; the choice
is an artistic control, not a more accurate storm model.

The reason to allow clouds independently of an ocean is observational:
NASA's [Venus cloud-top image](https://science.nasa.gov/asset/hubble/venus-cloud-tops/)
describes a global sulphuric-acid cloud cover. The existing procedural shell
is not a Venus model: its pattern, optical density and chosen heights are
art direction, and coverage does not guarantee the same visible fraction
from every camera. This control exposes the shell already drawn; it does
not introduce a cloud-type or atmospheric-composition model.

## Authored giant appearance

Generated giants accept `giant_look` in `generate`; omission retains the
existing appearance. The block is an artistic choice and requires
`made_up: true`. For example:

```yaml
giant_look:
  made_up: true
  paints: [[0.08, 0.22, 0.4], [0.7, 0.55, 0.32], [0.15, 0.35, 0.55]]
  roil: 0.06
  shear: 7
  grain: 0.08
  fine: 0.05
  mottle: 0.5
  jets: 10
  spots:
    - {lat: -0.3, lon: 1.2, size: 0.16, stretch: 2, turn: 2, paint: [0.9, 0.3, 0.1]}
  streets:
    - {lat: 0.4, count: 6, size: 0.05, turn: 1.5, seed: 3, paint: [0.9, 0.8, 0.6]}
```

Paints are linear RGB, latitude is the sine of latitude, and longitude
and turn are radians. A spot is a coordinate swirl and tint, not a fluid
simulation or a promise of vertical relief. Streets place seeded spots in
a row. The planet seed controls roil and the street has its own seed; the
same body is consequently stable across camera positions. These invented
values and the complete appearance are printed when the set is drawn.
The block is refused on rocky worlds.

An individual spot may also say `detail: 0.8` and `seed: 19`. Detail
breaks the otherwise smooth storm tint with seeded colour streaks sampled
in the swirled coordinates; zero or omission preserves the old appearance.
The frequency, number of noise scales and blending are made-up artistic
choices, not a forecast of storm motion or cloud height. NASA's JunoCam
[view of a Jovian vortex](https://science.nasa.gov/photojournal/suitable-for-framing/)
shows internal structure, while its [swirling cloudscape](https://science.nasa.gov/photojournal/jupiters-swirling-cloudscape/)
shows bright clouds and shadows in a more complex scene. These images inform
the desire for structure, not the numerical parameters or an assertion that
this colour-only model reproduces Juno's clouds. The former image uses colour
processing; the linked NASA captions credit the image and processing.

## What this is for

The immediate goal is a visually compelling, general constructor for
investor demonstrations: describe a system and pick or place cameras
throughout it. Caprica and the lava drop are examples of what the
constructor can make, not special cases that define its limits. Physical
realism and scientific plausibility are not requirements for these sets.
Published sources inform the models; invented values and artistic choices
are declared as made up in the set and printed when it is drawn.

Appearance needs controls that compose across worlds and remain coherent
from different cameras. Planetary cloud research separates coverage, cloud
type, vertical profiles and detail (Schneider and Vos, *Nubis*, 2017,
https://www.advances.realtimerendering.com/s2017/Nubis%20-%20Authoring%20Realtime%20Volumetric%20Cloudscapes%20with%20the%20Decima%20Engine%20-%20Final%20.pdf).
That separation is a useful authoring idea, not a claim that our clouds
implement Nubis or simulate weather. Chosen shapes, colours and parameters
are artistic unless a source says otherwise. World-space seeded structure
should persist as the camera moves. A visual assessment should include a
disc, a grazing terminator and close views of the same world, plus another
seed and visibly different settings.

Sources used by software belong in `sources.yaml` and on the shelf, with
retrieval and byte verification through the existing sources commands.
Reference imagery must retain its credit and any disclosure of processing.

### Comets

A system may describe a vast comet population by one population number,
without tracking each member. A camera estimates how many are expected in
its view; when one is selected to appear, its orbital elements, size and
appearance seed are written as YAML with the set. Existing canonical comets
are read before selecting any new one, and their later positions follow
Kepler. The population model and selection method must state their sources
or identify their assumptions as artistic choices. Around Caprica, comets
can glint near the terminator horizon with tails; that is a scene direction,
not a restriction on the constructor.

The 0.1.6 changelog states the goal: to make anything from a pebble to a
planetary system from fewer than sixty-four parameters, through common
constructors, and draw only what is asked for when it is asked for. A
system may hold ten planets around a binary while the frame shows one moon
of the third.

Today slayer has the parts of a system but no way to make one from a
handful of numbers. This design adds three constructors, a star, a planet
and a system, and the orbits and resonances that bind them.

## What already exists

- `scene/bodies.Body`: a checked core (name, kind, radius, mass) with
  parts hung on it, each there or not: Ground, Air, Clouds, Gas, Rings,
  Moons, Glow, Flare, Cities and more. A body takes a seed, spins, tilts
  and has an orbit. Its kinds are Rock, Giant, Hole and Star.
- `scene/bodies/template`: named bodies, among them Earth, Jupiter,
  Saturn, a hot Jupiter, the Sun, a pulsar, a magnetar and a quasar.
  Their own doc says the physical values were entered from memory and are
  unverified; they are starting points, not sources.
- `celestial/stars`: a star's life stage, surface temperature, luminosity
  and radius from its birth mass and age.
- `celestial/orbit`: bodies stepped through the galaxy's potential, with
  their nearest pass to the centre.
- `celestial/hole`: Kerr geometry and geodesics for black holes.
- `celestial/radiation`: light by wavelength, for a star's colour from its
  temperature.
- Terrain comes from Providers, today measured surveys; a city comes from
  a seed (`doc/city-clustered-10000.yaml`, seed 31).
- `kind: star` in a scene is read and not yet drawn (`session/undrawn.go`).

## The constructors

Each constructor takes numbers and a few named kinds, never prose. A
person's story about a system is turned into these numbers by an agent;
the constructor only ever sees the numbers.

### Star

Birth mass, age, metallicity and spin, and a seed. `celestial/stars`
gives the rest: life stage, temperature, luminosity, radius. A star that
has finished is a remnant: a white dwarf with its cooling age, a neutron
star, a pulsar, a magnetar. A star is a body that glows and lights every
other body in its system; nothing in the sky is painted.

### Planet

Mass, composition, spin, age, its orbit, and a seed. From those come its
radius (from a mass and radius relation for its composition), surface
gravity, the temperature its star's light gives it, whether it keeps an
atmosphere, and whether its water is ice, ocean or vapour.

Then the parts: Ground from a procedural terrain Provider that answers
the same questions a survey does, Air, Clouds, Gas, Rings, and Cities
where there is flat land near water in a temperate band.

The kinds a generator must be able to make, each a choice of composition
and parts rather than its own constructor:

- a temperate Earth, and a super-Earth, the same parts larger
- a hycean world: a global ocean under a hydrogen atmosphere
- a hot Jupiter, tidally locked, with a hot day side
- a Jupiter and a Saturn
- a rogue planet, lit only by its own heat
- a lava drop: a body of stellar mass made of dust and rock, formed in the
  dusty torus of an active galaxy (Mishra et al. 2026; nilgai, firang 2)

### Orbit

What the body goes around (its primary), and one of the semi-major axis
or the period: given the mass inside it, Kepler's third law fixes the
other, and a constructor given both checks they agree. Then eccentricity,
inclination and phase. A plunging orbit is mostly eccentricity.

Orbits name their primary, so a list of them is a tree: planets around a
star, moons around a planet, a captured body around the centre of mass of
everything inside it.

### Resonance

A resonance is a statement about two orbits: this period is p over q of
that one. The system derives the outer period, and so its distance, from
the inner. A chain such as 1:2:4 is resonances in a row.

### System

The centre, a list of orbits, a list of resonances, the system's age and a
seed. The centre is one of: a star, a binary, nothing (a rogue planet's
system) or a black hole with its torus. The same constructor serves all
four; what forms around the centre is set by the physics of that centre.

## History

Planets do not stay where they form. A system is made in two steps:

1. Formation: where bodies are born, from the centre's physics: the disc
   around a star, with its frost line and habitable zone from the star's
   luminosity, or the torus around a black hole.
2. History: what the system's age has done since: migration, scattering,
   ejection, capture, accretion, being swallowed.

Outcomes are drawn from published rates rather than integrated orbit by
orbit, so they are cheap and citable. Each body keeps a record of where
it formed and the events that moved it, from a short fixed list: formed,
migrated, scattered, ejected, captured, accreting. The record is output,
not input; it is how a hot Jupiter says why it is close, and how a rogue
planet names the system it left.

## Drawing

Every body's seed is its parent's seed and its place among its siblings,
hashed. A system can know all its bodies and make only the one in frame,
as Scrooge already does for detail.

For positions, nested Keplerian ellipses with fixed resonant ratios give
every body's place at a moment in closed form, and look right for almost
everything. A stellar-mass body on a plunging orbit, or a close
encounter, needs orbits integrated step by step; that is an option for
the systems that need it, not the default.

A system spans ten orders of magnitude, from its widest orbit to a
building, and a GPU's 32-bit floats hold about seven. Positions stay in
64-bit on the CPU with the camera as the origin; only offsets from the
camera go to the GPU. Before this is built, read how slayer places large
distances today: `Body.Place` takes a scale, and the orbital scene runs.

What is drawn follows how many pixels a body covers: below a pixel, a
point source with its true brightness; resolved, a sphere with its parts;
close in, terrain, then a city. At true scale a system seen from outside
is mostly points, as a real camera sees it.

Light comes only from bodies: a star by its temperature, a white dwarf, a
nebula of ionised wind glowing in its emission lines, the faint heat of a
night side or a rogue. The range is far wider than a frame, so exposure is
the darkroom's: exposed for a planet, the star burns out, as on film.

## Cameras

A system places its own cameras, from the numbers it already has. Each is
repeatable from the seed, can be overridden, and is what Ferruccio offers
first. `--camera overview` for a city is the same idea, smaller.

- System: above the orbital plane at about thirty degrees, far enough back
  that the widest orbit fills the frame.
- Portrait, for any body: the body fills a chosen share of the frame, seen
  about forty-five degrees from the line to its star, so it is lit three
  quarters with its terminator showing. A rogue is seen from the side its
  own heat makes brightest.
- Pair: two bodies, such as a white dwarf and its star, seen square to the
  line between them, far enough back to hold both.
- Sky: on the ground at a city, at the hour its star sits low, looking
  toward the brightest thing in that sky.

## A worked example

A lava drop formed in the Milky Way's centre during the active period of
an old merger, was thrown out on a plunging orbit, and was captured on its
way back in by a young, large star with a temperate Earth, a Jovian
planet, and an old white dwarf that has cooled a long way. The white dwarf
sits far enough out not to have novae, near enough that gas passes
between the two stars and lights the system.

The system constructor checks three things and says which fails:

- A stellar-mass lava drop is in effect a third star. The system must be
  hierarchical, the Earth close to its star and the lava drop on a wide
  orbit, or the Earth is thrown out.
- An old, cold white dwarf cannot have been born with a young star. Either
  it was captured too, which the Jovian and the white dwarf also make
  possible for the lava drop, since a capture needs a third body to carry
  off the energy, or the star only looks young because it has taken its
  companion's mass.
- At a comfortable distance the white dwarf takes the star's wind rather
  than its surface, and the wind it ionises glows, as symbiotic stars do.

## Sources to put on the shelf

Before code relies on a number, its source is on the shelf and cited:

- the mass and radius relations for rock, water and gas worlds
- the stability of hierarchical systems and Hill spacing
- the rates of migration, scattering and ejection
- hycean worlds (Madhusudhan et al. 2021)
- lava drops (Mishra et al. 2026), as nilgai cites them
- wind accretion onto white dwarfs in symbiotic systems
- verified values for the named templates, which today are from memory

## Order of work

1. Draw `kind: star`: a glowing body that lights the scene, from
   `celestial/stars`. It is the smallest piece that shows on its own.
2. The planet constructor and a procedural terrain Provider.
3. Orbit, resonance and the system constructor, with the stability checks.
4. Formation and history, with the record.
5. Lava drops and black-hole centres.

## Open questions

- How composition is given: a named kind, fractions of rock, metal, ice
  and gas, or both.
- Where the sixty-four parameters fall across the three constructors.
- When integrated orbits are worth their cost, and who asks for them.

## Found while drawing

On 2026-11-26, at rc-v0.1.9 397fcb299, a space set whose only contents
are `world.bodies` draws every body unlit: the hot Jupiter and Saturn
templates came out as dark discs. The same set with one thing under
`set.things`, out of frame behind the camera, draws them lit. A
generated system is nothing but bodies, so this is fixed before step 1.
