# The sky, from the air

For the opt-in terrain-free Earth scene, its explicit datum and model
limitations are recorded in [groundless-earth-air.md](groundless-earth-air.md).

Earth's blue sky is mostly light scattered once. Under Venus's cloud,
light has been scattered hundreds of times: the cloud's optical depth is about 30, and the clear carbon
dioxide below it adds about 16 more at 550 nm.

## What this builds

One way to work out the light in any planet's air, from what the air is
made of, used for every sky slayer draws.

- Inputs are physical: the planet's radius; the air's layers by height,
  each with its gas scattering, its gas absorption, and its particles'
  extinction, single-scattering albedo and phase, all by wavelength;
  the ground's reflectance by wavelength; the sun's spectrum and its
  distance. Every number is read from a source or marked picked.
- Output is the radiance field: at each height, for each angle between
  the place and the sun (the local solar zenith), the light arriving
  from every direction, by wavelength, for unit sunlight at the top.
  Also the direct sun's transmittance to every height and angle, and
  what leaves the top: the planet seen from outside.
- The renderer reads a slice of the field at the camera's height and
  the sun's angle as its `sky.Sky`; the sun becomes a lamp whose
  strength is the sun's spectrum times that transmittance; the fill
  lamp goes, because `skytable` now lights the ground from the sky.

## The method, and why

Forward Monte Carlo transport in spherical shells.

- Photons enter across the planet's top sphere, facing the sun: over
  its whole disc for a walk of every sun, and over a band of it for a
  set's walk (below). Each walks a free path drawn by optical depth
  through the shells.
- At each event its weight is multiplied by the single-scattering
  albedo; it picks gas or particle by their scattering shares, and turns
  by that one's phase function.
- The ground reflects it as a Lambertian surface of its albedo. It ends
  when it leaves the top or its weight is rouletted out.
- Radiance is tallied where photons cross chosen heights, binned by the
  local solar zenith, the view's zenith and its azimuth from the sun.
  For a sphere lit by parallel light the field depends on nothing else.
- A set's run serves one pair of the sun's 5-degree zenith bins, the
  two its sun is read between, and no other sun. Its photons start
  only on the band of the disc whose light can reach the ground under
  such a sun, so those bins get as many more crossings as the disc is
  larger than the band: about ten times at dusk, four with the sun 17
  degrees up, two at 47.

  A sun outside the pair is refused in words,
  by the meter and by the sky; it walks again under its own key.
- The bins of the other zeniths in that run hold only part of their
  light. A run of every sun, the whole disc, still exists in the
  library (airlight.NewPlan without For), but no set asks for it.
- Each photon draws from a generator seeded by the run's seed and its
  own number, so it walks from the same numbers at every wavelength.
  Their noise then moves together, and the sky's colour, a ratio of
  wavelengths, keeps less of it than any one wavelength.

Why this and not another:

- It is indifferent to optical depth. Earth's 0.1 and Venus's 45 are
  the same code. Orders-of-scattering methods, the usual precomputed
  skies among them, need thousands of orders to converge under Venus's
  cloud.
- It is spherical, so the sky holds from any height and with the sun at
  any angle, below the horizon included. Plane-parallel solvers fail
  there, and that is the painted sky's failure: right from one place.
  A set's run holds this only for the pair of sun zeniths it walked.
- Its checks are exact: every photon is accounted for, a thin layer must
  give single scattering, and a thick conservative slab must give
  Chandrasekhar's diffuse transmission.

Its cost is time, paid once for each key: the field is cached under the
hash of everything that decides it, as every other slayer cache is.

## What a job pays for its sky

- One walk for each distinct air, ground height and pair of sun zenith
  bins. Two sets whose suns fall in the same pair, over the same air
  and ground, share one walk.
- A walk is a first walk of the middle wavelength, 580 nm, at the
  air's photons, which measures the crossings in the bins read; then
  every wavelength at as many photons as bring those bins to the
  crossings wanted (below), from the air's photons up to 32 million.
- Measured on four workers under a load of 9 to 12: 972.5 s for a sun
  17 degrees up over a sea-level ground, 917.7 s for a sun 47 degrees
  up over a ground 2.3 km below the sea, both at 32 million. The old
  walk of every sun at 4 million took 54 to 103 s. Read back from the
  cache, a walk takes milliseconds.
- A film whose sun moves walks again for each new pair of bins: about
  sixteen minutes each at these loads. A sun that sweeps 90 degrees
  crosses up to 18 pairs, about five hours once, then cached. There is
  no longer one table that serves a whole day.
- The first walk is thrown away, about a twenty-first of one walk at 4
  million. Its photons are not kept toward the count: the count is
  chosen from them, and counting them too would let the stopping rule
  lean on its own sample. It is walked from a seed of its own, so the
  walk's photons are not the ones it measured.

## The noise a walk keeps

- The colour noise wanted between neighbouring points of the sky's
  table is one 8-bit step at sRGB code 200: 2.4 times 1/255 over 0.78,
  1.2% of linear light.
- The colour noise, the median over the rows above ten degrees of log
  R/G's second difference round the azimuths, was 1.39 over the root
  of the median crossings at 240 crossings a bin and 1.32 at 1090: a
  sky over the open sea off Honopu, Kaua'i, earth's air, walked from
  the set's lowest ground, 49.8 m below the datum, sun 16.96 degrees
  up, counts at 540 nm.

  The walk reads them at 580 nm, a few percent
  fewer, so it asks a few percent more photons than needed. So
  the crossings wanted are (1.32 / 0.012) squared, 12,100. This is one
  sky: Venus's air and a sun under ten degrees are not measured.
- A run says what it kept: "the bins its sun is read from hold a
  median of N crossings, a colour noise of about X%".

## Checks that it is right

- Conservation: what leaves the top, plus what the air absorbs, plus
  what the ground absorbs, equals what came in.
- Vacuum: no air, no sky, full sun.
- A thin Rayleigh layer: the field equals single scattering worked by
  hand, and celestial's `Skylight` for Earth's air.
- A thick conservative slab on a very large sphere: diffuse transmission
  as Chandrasekhar's and van de Hulst's tables give it.
- Earth: a clear sky is blue at the zenith, white near the sun and the
  horizon, and its brightness falls toward the zenith as measured.
- Venus at the ground: about 14,000 lux at the Venera 9 site (Avduevsky
  and others, via Wikipedia, read 2026-11-15), and a spectrum that rises
  to the red as Venera 13 and 14 measured.
- Venus from outside: its Bond albedo, about three quarters. The same
  field that lights the ground must give the planet's measured
  brightness from space. A painting cannot be held to both.

## Sources

Venus's air: its structure by height (VIRA, Seiff and others 1985),
carbon dioxide's Rayleigh cross section (Sneep and Ubachs 2005), the
cloud's layers, particles and optical depths (Titov and others 2018,
Pollack and others 1993), sulphuric acid's refractive index, and the
sun's spectrum.

## Not in this

- Polarisation. Rayleigh light is polarised; the radiance here is not.
- Light from the air or ground themselves, which matters on Venus past
  about a micron and is not in the visible.
- Weather that is not layered: a cloud with edges is the volume tracer's.
