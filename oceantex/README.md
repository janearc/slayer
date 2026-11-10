# oceantex

a CPU reference for the sea, from a beach toward orbit. Source attributions
and authored choices are retained; several derivations still need review.
See [the integration status](../doc/phone-ocean-integration.md) before using it
in a scene. Colour goes through
[libtheme-css](https://github.com/janearc/libtheme-css): its 1931 observer,
its d65 and black body, its srgb.

    // a moderate sea off california on a november afternoon
    sea, err := oceantex.SeaState(4).Raise(300e3, 0, -0.4)
    scene := oceantex.Scene{Sea: sea, Salinity: 33.5, Celsius: 13,
        Chlorophyll: 0.5}
    scene.SunAt(37.5, -122.5, when)        // direction, colour, dimming

    // a camera is a lens at an eye, as slayer's is
    eye := oceantex.Vec{Z: 2}              // two metres above the water
    picture, err := oceantex.View(scene, oceantex.Camera{
        Eye: oceantex.Subspace{Origin: eye,
            Orientation: oceantex.LookAt(eye, oceantex.Vec{X: -100, Z: -5})},
        Lens:  oceantex.NewLens(35, oceantex.Gauge{}, 0), // a 35 on super 35
        Width: 960, Height: 540,
    })
    png.Encode(file, picture)              // a Texture is an image.Image

    // or a square, tiling texture from straight above, and its waves
    texture, field, err := oceantex.Render(scene, oceantex.Tile{
        Size: 512, Extent: 40, Altitude: 30,
    })

The original phone demo wrote shore, cliff and overhead views, sea states,
top-down tiles and water colour strips. It is not installed as a Slayer
command yet; that entry point will use the existing Game demo dispatcher.

oceantex is a package of slayer: `github.com/janearc/slayer/oceantex`.

## what is in it

    View          a picture from any camera: an eye and a lens
    Lens, Gauge   a focal length on a film gauge; Swath frames a strip from orbit
    Pinhole       a field of view in radians; Projection is the interface
    LookAt, Look  an eye's orientation; a pixel's ray out in the world
    Render        a square tiling texture from straight above, any altitude
    SeaState      wmo code 3700 (0 calm to 9 phenomenal); Raise finds the wind
    Beaufort      wind speed of a beaufort force
    Sea           wind, fetch, direction, depth
    Tide, Flood   a tide from its constituents; a sea with the tide in
    Scene.SunAt   the sun's direction and light for a place and a time
    Water         the colour of water holding some chlorophyll
    ColourAt      the likely colour at a place on a date, given a map
    Sources       every number used, and where it came from

## the shore

oceantex lives in slayer beside titan, and a `Scene` may have a `Shore`:
a square of titan's ground under and beside the sea. `NewShore` samples
titan once onto a grid; `Shore.Made` says how much of each height titan
generated. `View` then draws both, and the sea floor changes the waves:

- they shoal: taller as the water shallows, by the ratio of group speeds
  (linear theory, no numbers of its own)
- they are held to the depth in the surf: root-mean-square height no more
  than 0.42 of the depth (thornton & guza 1982)
- a crest is white where its wave is taller than 0.78 of the depth
  (mccowan 1894), and every crest is white inside the surf
- a `Wall` sends waves back, by mirroring the sea in a straight line

The original cliff demo used **synthetic ground**, standing in for a
provider of real lidar and soundings. That demo command is not part of this
import; its authored geometry must not be described as measured terrain.

what this is not: there is no spray up the cliff, no run-up, no foam left
behind a broken wave, and no bending or bunching of the waves as they
come in; one wave, the spectrum's peak, stands for the whole sea when it
is shoaled. the rock is one picked grey with no shadows. the wall is one
straight line however the ground wanders. the breaking numbers were
measured on sand beaches and are used on a rock shelf.

## the camera

it is patterned on slayer's (`ninjafilm/film` and `maths`, at v0.1.0): a
camera is a projection at a subspace. the names, defaults and conventions
are slayer's, so a shot framed for one frames the same in the other.

a `Lens` is a focal length in millimetres on a `Gauge` in front of a
`Pinhole`; the zero lens is a 50 on super 35; a pinhole of no width sees
60 degrees; `Wide` is edge to edge across the frame, in radians.

a `Projection` gives each pixel a ray down its own minus z, row zero at
the top; `Look` takes it out through the eye's `Subspace`; `Swath(across,
above)` and `Focal(wide, gauge)` are as slayer has them, and the gauge
table is slayer's.

it is a pattern, not an import. oceantex does not depend on slayer, so
slayer's `film.Lens` cannot be handed to `View` as it is. three things
differ:

- up here is z (x east, y north); slayer's is y. an eye with no turn
  therefore looks straight down, north up the picture.
- a `Projection`'s `Ray` takes no band. oceantex folds the spectrum into
  colour before the camera, so there is nothing for glass to split.
- a `Gauge` carries no grain and a `Subspace` no scale. there is no
  shutter, take or film stock: `Camera.Seconds` is one instant.

## sources.yaml

every measured or chosen number is in `sources.yaml`, with the publication
it belongs to, the url i read it on, the date, and whether i fetched that
page, read it in a search excerpt of it, or cloned the repository it is in. the code holds none of them: it
reads each by name, `num("cox-munk.along.slope")`, and
`TestEveryNumberIsSourcedAndUsed` fails if the code asks for a name the
file lacks or the file has one the code does not use.

three kinds of entry:

    sourced      from the publication named
    definition   exact by definition or convention
    glue         mine, from no publication: floors, grid sizes, step counts

what stays in the code is arithmetic: pi, the 2 in a gaussian, fresnel's
halves and squares.

things a checker should know:

- most numbers were read in a secondary copy (nasa source code, lecture
  notes, a later paper), not the original. each entry's url says which.
- fetches go through a tool that summarises the page. where it said text
  was garbled or reconstructed, the entry's note says so: jonswap's 1/3
  power, the square root in the tma depth factor, banner's 10^.
- smith's shadowing function has no measured numbers in it, so it has no
  entry. i read it in ross & dion (spie 5572, 2004), where the fetch tool
  reported the equation text as partly garbled and filled in the standard
  form. treat its attribution as partly checked.
- cooley & tukey 1965 (the fft) and mastin, watterberg & mareda 1987
  (fourier synthesis of seas) are cited from memory as the origin of
  methods; they contribute no numbers.

## how one sea is seen from every height

a pixel shows the waves at least as long as the sea it covers. shorter
ones are not dropped: their slope variance, which is what cox and munk
measured less what the picture already shows, widens the sun's glint. near
the camera that is sharp facets; toward the horizon and from orbit it is
the smooth glitter pattern. there is no switch between the two.

`View` builds the sea from three tiled patches (4 km, 512 m, 64 m), each
carrying the waves too short for the next larger, and marches each pixel's
ray to the first wave it meets, so near waves hide far ones. crests are
sharpened by tessendorf's sideways displacement.

## the colour of the sea at a place

`Water(lamp, chlorophyll)` is sourced end to end: pope & fry's absorption,
morel's scattering, the classic case 1 model for pigment and dissolved
matter, gordon's and lee's reflectance. chlorophyll is the green pigment of
phytoplankton, in milligrams a cubic metre: least in the clear mid-ocean
gyres, most where the water is rich. the demo's strip runs it from 0 to
10.

`ColourAt(lat, lon, date, chart)` needs a map of typical chlorophyll by
place and month, and **this package does not contain a real one**. the
hosts that serve them could not be reached from where this was built, and
i would not type one in from memory.

`ReadGrid` reads one as rows of `latitude, longitude, month, chlorophyll`;
nasa's modis-aqua monthly climatology, exported to that, is the thing to
give it. `Even(0.2)` is the same value everywhere.

what is here instead is `data/chlorophyll-points.csv`: 23 points of open
ocean, by month, that a research subagent read out of nasa's modis-aqua
product one pixel at a time and averaged over three years. the readings
are real; the averages are its arithmetic, it is far too sparse to be a
map, and its header says everything that is weak about it. nothing uses
it unless you load it.

for scale, o'reilly & werdell (2019, remote sens. environ. 229, 32-47)
call water below about 0.1 mg/m^3 oligotrophic, 0.1 to 1.67 mesotrophic
and above 1.67 eutrophic. the subagent read that in the abstract at
https://modarch.gsfc.nasa.gov/sci_team/pubs/abstract_new.php?id=31786 on
2026-11-09; i did not read it myself.

## what is mine

the joints between the sources. each is marked GLUE in the code, and its
numbers are `kind: glue` in sources.yaml.

- sub-pixel roughness = cox-munk less what the picture resolves, with a
  floor
- the glint's slope density is centred on the pixel's resolved slope
- the sky reflects off the pixel's mean slope only
- shadowing toward the eye and toward the sun are multiplied
- a sea stops growing at the fetch where jonswap's peak falls to pierson
  and moskowitz's; jonswap's own formulas never stop
- the 10 m wind is given to cox and munk (12.5 m) and to pierson and
  moskowitz (19.5 m)
- a sea state is drawn at the middle of its range of heights
- the sun above the air is a black body at 5772 k; the air takes out only
  what its molecules scatter
- the backward share of particle scattering, measured at 550 nm, is used
  at every wavelength; so is the 550 nm refractive index, and the
  reflectance for looking straight down
- water is black outside 400-700 nm
- the wave patches tile, and their sizes; a pixel's footprint is taken
  between its width and its length along the line of sight
- choppy displacement at full strength, held short of folding
- the camera's defaults are slayer's choices, and its gauge sizes are
  slayer's table, which slayer says it inherited and has not checked
  against a publication. neither have i.

## what is missing

left out because i did not check a source for it, not because it is small.

- **the sky.** `Scene.Sky` takes one; the package has none. the demo's
  `-sky` is a flat grey placeholder. most of what a calm sea looks like
  from a beach is the sky in it, so this matters.
- **the atmosphere between the sea and the camera**, which is most of what
  you see from orbit, and ozone, water vapour and dust on the way down, so
  a low sun here is brighter and less red than a real one.
- **the curve of the earth.** the sea is flat: the horizon is at eye level
  and an orbital view has no limb.
- **a real shore.** see "the shore" above for what the shore does and
  does not do. `Render`'s top-down tiles know nothing of it.
- **tide tables.** `Tide` adds up constituents with noaa's speeds. it
  leaves out the astronomical zero each published phase is counted from
  and the 18.6-year swell of each amplitude, so it gives a tide of the
  right size and rhythm, not the right hour.
- **a chlorophyll map**, as above, and any water that is not deep and
  clear: mud, blooms, the bottom showing through.
- swell from elsewhere, capillary waves, where foam sits (it is an even
  veil), cox and munk's skewness and peakedness, the sun's width,
  refraction of the sun near the horizon.
