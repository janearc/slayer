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

- Photons enter across the disc of the planet's top sphere, facing the
  sun. Each walks a free path drawn by optical depth through the shells.
- At each event its weight is multiplied by the single-scattering
  albedo; it picks gas or particle by their scattering shares, and turns
  by that one's phase function.
- The ground reflects it as a Lambertian surface of its albedo. It ends
  when it leaves the top or its weight is rouletted out.
- Radiance is tallied where photons cross chosen heights, binned by the
  local solar zenith, the view's zenith and its azimuth from the sun.
  For a sphere lit by parallel light the field depends on nothing else,
  so one run serves every time of day and every camera height.

Why this and not another:

- It is indifferent to optical depth. Earth's 0.1 and Venus's 45 are
  the same code. Orders-of-scattering methods, the usual precomputed
  skies among them, need thousands of orders to converge under Venus's
  cloud.
- It is spherical, so the sky holds from any height and with the sun at
  any angle, below the horizon included. Plane-parallel solvers fail
  there, and that is the painted sky's failure: right from one place.
- Its checks are exact: every photon is accounted for, a thin layer must
  give single scattering, and a thick conservative slab must give
  Chandrasekhar's diffuse transmission.

Its cost is time, paid once: the field is cached under the hash of
everything that decides it, as every other slayer cache is.

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
