# Bodies and gravity: the grav design

This is an unimplemented design recorded on 2026-10-05, not documentation
of an existing `grav` package in this checkout. It follows the engineering
guide’s `gravBody` example: planets, brown dwarfs, stars, white dwarfs,
neutron stars, and black holes share a physical description.

Mass, composition, size, and age determine the body. `grav` owns the
properties determined by gravity; other packages own emission and appearance.

## Construction boundary

Values enter through validating constructors and expose private state
through methods. Distinct parameter types prevent accidental swaps;
private fields allow representation changes without changing clients.
Any public struct needs an explicit reason.

The design depends on:

- `measure`, using MeV and natural units with c, ℏ, and kB equal to one,
  and a measured Newtonian G. Locations use `measure.Coord`.
- `materials`, describing composition and exposing mass.

## Matter

```go
type Matter interface {
    Mass() measure.Mass
}
```

A star or planet carries a materials amount: composition and total mass.
A black hole carries `measure.Mass` directly, whose `Mass()` method returns
itself. The gravity layer asks for mass without inspecting composition;
material models supply density or structural information where needed.

## Spin

```go
func Turning(axis measure.Direction, day measure.Duration) (Spin, error)
func Momentum(axis measure.Direction,
    amount measure.AngularMomentum) (Spin, error)
```

`Turning` describes a finite-sized body by its measured rotation period,
which must be positive and finite. For a uniform-density sphere, angular
momentum follows from moment of inertia `2MR²/5`; without a density profile,
the body retains its period.

`Momentum` describes a black hole by angular momentum. With ℏ equal to one,
the quantity is a dimensionless count in those units, still represented
by its own type.

## Body construction

```go
func NewBody(matter Matter, age measure.Duration, spin Spin) (Body, error)
func NewHole(mass measure.Mass, spin Spin) (Body, error)
```

`NewBody` derives radius rather than accepting an arbitrary one:

- Bulk-density materials use mass and density.
- Self-gravitating gas uses published mass-and-age tracks.
- Unsupported material models return an error identifying the missing case.

The proposed track sources are Baraffe and colleagues, from about half a
Jupiter mass to above one solar mass, and MIST (Choi and colleagues, 2016),
from about 0.1 to 300 solar masses, including giants and white dwarfs.

The public constructor checks required values, positive finite mass, age
within the universe’s age, and the derived radius. A zero-value `Spin`
is invalid because it bypasses `Turning` and `Momentum`.

A derived radius within the body’s own horizon returns an error directing
the caller to `NewHole`. The private constructor then caches GM, horizon,
and density. Callers should not repeat those calculations or validations.

`NewHole` rejects nonpositive or nonfinite mass, period-based spin, and
angular momentum at or beyond the proposed `G × mass²` bound. This is the
design’s stated nonextremal boundary; it is not an implemented API here.

## Body properties

```go
func (body Body) Mass() measure.Mass
func (body Body) Radius() measure.Length
func (body Body) Age() measure.Duration
func (body Body) Spin() Spin
func (body Body) Density() measure.Density
func (body Body) GM() measure.Length
func (body Body) Horizon() measure.Length
func (body Body) Compactness() Compactness
func (body Body) EscapeSpeed() measure.LightFraction
func (body Body) SurfaceGravity() measure.Acceleration
```

In natural units, GM is a length. A nonrotating black hole has horizon
radius `2GM`; the rotating result belongs to the black-hole model.
Compactness is `GM/R`: roughly two parts per million for the Sun, about
one sixth for a neutron star, and one half at a nonrotating horizon.

Escape speed is typed as a fraction of c.

## Interactions between two bodies

```go
func Pull(one, other Body, apart measure.Length) measure.Force
func Hill(small, large Body, orbit Orbit) measure.Length
func Roche(small, large Body) measure.Length
func Tidal(star, hole Body) measure.Length
```

`Hill` estimates the region where satellites remain bound to the smaller
body. `Roche` gives the tidal disruption distance. `Tidal` applies the
same question to a star approaching a black hole: disruption outside the
horizon may be visible; inside it, the star is swallowed first.

The fluid Roche coefficient comes from Roche’s numerical solution and
needs a citation where used. The rigid-body approximation has a direct
closed-form derivation.

## Orbits

```go
func NewEccentricity(value float64) (Eccentricity, error)
func NewTilt(angle measure.Angle) (Tilt, error)
func NewNode(angle measure.Angle) (Node, error)
func NewPeriapsis(angle measure.Angle) (Periapsis, error)
func NewShape(apart measure.Length, eccentricity Eccentricity,
    tilt Tilt, node Node, periapsis Periapsis,
    since measure.Duration) (Shape, error)
func NewOrbit(primary, secondary Body, shape Shape) (Orbit, error)
func (orbit Orbit) Period() measure.Duration
func (orbit Orbit) At(when measure.Duration) Places
```

The three orientation angles have distinct types to prevent swaps.
Construction rules are:

| Parameter | Proposed rule |
|---|---|
| apart | Positive finite semi-major axis |
| eccentricity | Finite value in `[0, 1)`; open orbits are deferred |
| tilt | Zero through half a turn, avoiding duplicate orbit spellings |
| node, periapsis | Any finite angle, normalized to one turn |
| since | Any finite time, including negative; normalized by the orbit period |

Kepler’s third law uses both masses. `At` solves Kepler’s equation with
bounded Newton iteration to floating-point precision and reports failure
to converge rather than returning a guess.

`Places` supplies barycentric positions and velocities through methods.
The untilted orbital plane has north along y, matching the scene convention.

## Construction examples

These are sketches of the proposed API. Error handling is omitted for
readability; each error must be checked in working code.

```go
// Solar-mass hydrogen and helium: the tracks determine its structure.
solar, err := materials.SolarAbundances()
sunMatter, err := materials.NewAmount(solar, measure.SolarMasses(1))
sunSpin, err := grav.Turning(measure.Up(), measure.Days(25.4))
sun, err := grav.NewBody(sunMatter, measure.Years(4.6e9), sunSpin)

// The same constructor for a Jupiter-mass hydrogen body.
hydrogen, err := materials.NewElements(map[string]float64{"H": 1})
gas, err := materials.NewAmount(hydrogen, measure.Kilograms(1.898e27))
giantSpin, err := grav.Turning(measure.Up(), measure.Hours(9.93))
giant, err := grav.NewBody(gas, measure.Years(4.6e9), giantSpin)

// An uncompressed planet made entirely of forsterite.
formula, err := materials.NewFormula(map[string]float64{
    "Mg": 2, "Si": 1, "O": 4})
olivine, err := materials.NewMaterial("forsterite", formula,
    measure.GramsPerCC(3.27))
rock, err := materials.NewAmount(materials.Only(olivine),
    measure.Kilograms(5.97e24))
rockSpin, err := grav.Turning(measure.Up(), measure.Days(1))
planet, err := grav.NewBody(rock, measure.Years(4.6e9), rockSpin)
```

The rock example yields a radius near 7,600 km, versus Earth’s 6,371 km.
That difference follows from the chosen simplification: Earth has a dense
iron core and pressure-compressed rock; this example has neither.

## Gas giants and interior models

Giants use `NewBody` rather than a separate constructor. Composition and
age distinguish them, using explicitly named interior models:

```go
saturn, err := grav.NewBody(materials.Planet("saturn"),
    measure.Years(4.6e9), saturnSpin)
uranus, err := grav.NewBody(materials.Planet("uranus"),
    measure.Years(4.6e9), uranusSpin)
```

The design’s reference expectations are hydrogen and helium with a diffuse
rock-and-ice core for Saturn, following Mankovich and Fuller (2021), and
ice-rich interiors for Uranus and Neptune, with Helled and colleagues
(2011) as one of several competing models.

The rough proposed fractions are a tenth to a fifth of Saturn’s mass in
the diffuse core, versus about three quarters in ices and roughly a tenth
in a hydrogen-helium envelope for the ice giants. These are model choices,
not settled unique compositions.

## Physical and data bounds

| Bound | Rejection | Basis |
|---|---|---|
| mass | Nonpositive or nonfinite | Numeric validity |
| age | Negative or older than the universe | Planck age estimate |
| spin | Period shorter than rotational breakup | Density |
| horizon | Radius inside its own horizon | `2GM`; use `NewHole` |
| gas retention | Envelope lost over the body’s age | Jeans escape |
| tracks | Outside all supported tracks | Published data coverage |
| shares | Composition does not sum to one | Materials validation |
| layers | Layers not ordered by density | Interior-model constraints |

For the simple self-gravitating model, breakup period is `√(3π / Gρ)`:
it depends on density rather than size. Unsupported tracks fail explicitly.
The proposed coverage spans about half a Jupiter mass through 300 solar
masses; the actual tables must establish the supported domains.

## Fusion and evolution

The design derives stellar behaviour from evolutionary tracks instead of
hard-coding a separate star constructor. Near 0.075 solar masses, tracks
settle into sustained hydrogen burning; below that, bodies continue cooling.
Roughly 13–80 Jupiter masses can burn deuterium temporarily.

Composition, mass, and age choose a track. External heating can inflate
close-in giants and belongs to a later systems model. Rotation-dependent
flattening requires an internal density profile.

MIST lifetimes would replace the current stars model’s `burnedShare`
approximation, including the noted factor-of-four short lifetimes for
massive stars. Its giant and white-dwarf stages would extend the existing
`ended` handling.

Black holes retain separate construction because their
formation history is outside this structural model.

## The lower limit for gas retention

Small bodies may lose their lightest atmospheric molecules. The proposed
check compares Jeans escape over the body’s age using escape speed and
the gas temperature and molecular mass, rather than a fixed size threshold
or an escape-speed rule of thumb.

Earth retaining nitrogen while losing hydrogen, and cold Titan retaining
a thick nitrogen atmosphere, illustrate why composition and temperature
matter. An isolated hydrogen body may freeze rather than boil away;
external heating is therefore important.

Super-puffs such as Kepler-51 are useful stress cases: youth, contraction,
haze, and rings are competing interpretations. Proposed hydrogen-covered
ocean worlds such as K2-18b require layers and remain scientifically
contested; they do not establish a completed model here.

## Quantity types and coordinates

The proposal adds `Acceleration`, `Force`, `Angle`, `Density`, `Direction`,
`Coord`, and `Displacement` to measure. Solar reference quantities become
methods such as `Sun().Mass()`.

Dimensionless values still need distinct types:

| Type | Interpretation |
|---|---|
| Eccentricity | Focus offset divided by semi-major axis |
| Compactness | GM divided by radius |
| LightFraction | Speed divided by c |
| AngularMomentum | Angular momentum in units of ℏ |
| Direction | Vector divided by its magnitude |

A coordinate contains three positions, not three lengths. Subtracting two
coordinates produces a displacement; adding a displacement to a coordinate
produces a coordinate. Adding two coordinates should not compile.

Frame identity—solar, body-local, camera—belongs in a future location layer
above these quantities.

## Outside this package

Emission belongs to `emit` and stellar models. Strong fields belong to
`holes`, in front of libholemath. Gravitational-wave inspiral can layer
Peters’ formula above `Orbit`. Three-body systems and shape-changing tides
are deferred until their models and scene requirements are specified.
