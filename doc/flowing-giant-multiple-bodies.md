# Flowing giants with other bodies: a staged path

2026-11-27. Design only; no claim that this is implemented or visually
validated. The present bridge is deliberately restricted to one body and
one solar-luminosity directional sun, a fixed still and metre world units.

## The current light boundary

`session/flowing_bodies.go` makes a spectral `planet.Face` for one chosen
sun, current eye and a fixed body-frame meridian, then puts it in a globe's
`thing.Material.Shine`. `Shine` is surface emission to the tracer. In
`trace/surface.go`, that emission is added to the lamp-lit and sky-lit
terms; it is not multiplied by the tracer's shadow visibility. Thus adding
another planet or spacecraft can occlude the globe *from the camera* while
failing to cast its shadow on the face. The current draw record says this.
The fact that a cloudy face looks three dimensional is not evidence of
inter-body illumination or shadows.

Each `planet.Face` also binds a particular eye, sun direction and spectrum.
Its own `Chart`, where used, represents this view and illumination; it is
not a body-space material independent of lights or camera. The simulation's
weather checkpoint has a separate identity: `weather/cache.go` includes the
solver version, grid and radius, rotation, forcing, seed and compute backend;
the requested moment names its retained checkpoint. Optical columns and
camera are inputs to the face, not to the weather checkpoint.

## First implementation slice: multiple bodies, one sun, CPU

Keep the existing one directional-sun restriction. Permit multiple fixed
system bodies only after adding an explicit tracer material contract for a
**precomputed direct-sun face**. At each shaded surface point, sample that
face as today, then gate that direct-sun contribution by the visibility of
the *same* selected sun from the surface point. Reuse the tracer's existing
shadow/transmission walk, raised-origin convention and finite blocker rules;
do not cast a second diffuse light contribution on top of the prelit face.
Ordinary bodies continue to occlude rays from the camera and to cast shadows
using their normal geometry. Do not silently attach the face to the ordinary
unconditional `Shine` path. The new contract must identify the sun it
represents; a different lamp must never gate it.

For an opaque blocker this can darken the direct contribution through a
binary eclipse. A colored/translucent blocker's current RGB transmission
applied to an already-integrated RGB face is an approximation, not a
wavelength-by-wavelength spectral calculation; either explicitly refuse it
in this first slice or declare that approximation by name. Do not call it
spectral eclipse transport. The present face already computes lighting
internally; a blocker can cover part of the angular extent of an extended
star or part of the atmosphere along a slanted path. The simple single-ray
gate only handles the chosen point/directional-light approximation and does
not model penumbra, volumetric atmospheric shadows or extra scattered light.

Stage each flowing planet separately after the eye is known, using its own
name, center, radius, fixed meridian, optical columns, weather and cache key.
Continue to construct other bodies via `stage.Add`, without adding the old
giant volume for a flowing body. Preserve the explicit refusal in world-only
and placement APIs that have no eye. Keep films, camera paths, shutters,
zoom, lens rigs, moving orbital bodies, additional lamps and binary/multiple
stars refused until their view/light lifecycles are implemented. Permit
several static bodies and opaque occluders only when the new tracer contract
is active. Never replace an unsupported case by an unshadowed picture.

Implement the first slice on CPU. The GPU must either implement the same
explicit contract with its native shadow query, or refuse the new material
by name; it must not fall back to unconditional emission. Chart the
unoccluded face if the chart is otherwise valid, then apply visibility at
the actual surface hit, so occluder changes cannot make a stored chart's
shadow stale. A future multi-star design needs a separate incident spectrum,
direction, face contribution and visibility for *each* star; multiplying a
combined prelit RGB image by one shadow is insufficient.

## Visual work

The implemented flowing appearance also accepts optional `shadow_fill`
from zero to one; zero is the default. With multiple bodies, a blocked
directional-sun contribution retains that fraction of its prelit face.
This is an explicitly invented appearance control, not an implementation of
penumbra, indirect illumination or scattering. It changes the shadow's
interior brightness and leaves the point-sun shadow boundary hard. A
positive fill without multiple bodies is refused.

`flowing.sun_disc_degrees` optionally gives the full angular diameter of a
uniform distant source, up to 10 degrees; zero keeps a point-direction
shadow. The face remains prelit for the disc center and is modulated by the
fraction visible along a fixed golden-angle pattern; `sun_disc_samples`
chooses 8–64 directions and defaults to 8. Sixty-four uses up to eight times
the visibility rays per hit. The deterministic pattern does not itself prove
animation stability, and flowing films remain refused. This is a bounded
extended-visibility approximation, not finite-star, limb-darkened, spectral
angular or volumetric transport. Ordinary surfaces still use the scene's
directional lamp. See the authored penumbra study for limitations and the
deliberately enlarged comparison size.

Construct a fixed system with one flowing giant and one opaque moon under
the same directional sun. Choose a camera sequence of stills with the moon
beside the disc, between the sun and the giant, and between the camera and
the giant. Record placement, weather checkpoint, sun and exposure. The
sunward transit should show the moon's cast shadow on the giant; the
camera transit should show foreground occlusion independently. Include a
control with the moon displaced out of the sun path, then vary its size and
sun direction. Inspect full-sized frames and keep linear raw outputs; if
the shadow is missing, detached or double-counted, return the slice to the
builder. These images evaluate the named point-light approximation; they do
not establish extended-star penumbrae, multiple scattering, general dynamic
systems, or GPU parity.

Separately, preserve the existing two-camera weather landmark observation:
the geometric markers agreed visually with projections, but no preselected
weather feature could be matched in the shaded images. A cached-field
visualization may make the fluid more legible, but does not itself verify
that the shaded renderer uses identical body-space coordinates.
