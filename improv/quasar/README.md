# quasar

Short films about quasars, blazars and photon rings, hosted by zappy,
and the arithmetic behind every number they put on screen.

The point of the series is honesty about the pictures. Slayer's hole
pictures are drawn from an impossible place with an impossible field of
view, in colours chosen to be looked at. These shorts say so, and say
what is computed and what is painted.

## Layout

- `rings/` is the library for the photon ring short. It follows light
  past a hole that does not spin, with celestial's hole, the march the
  tracer uses. It says which disc image a ray brings back (the direct
  one, the one that went round once, and so on), where on the eye's sky
  each image lies, and how thin each is in pixels for a camera of a
  given width. The top-down diagram of two rays is drawn from its paths.
- `facts/` is every number a short says, each either a registered value
  from `research` (read at its source) or computed from registered and
  defined values by a function with a test. A sentence that names a
  number neither gives is refused.

## What the photon ring short may claim

From the code, read on 2026-11-12:

- The bending is computed. Inside the hole's reach (twenty
  schwarzschild radii) every ray is a kerr geodesic marched by
  celestial's hole. An outgoing ray past twelve masses is let go
  straight, which `trace/field.go` measures at up to eight degrees short.
- The tracer has no limit on how many times a ray goes round. It stops a
  ray after forty thousand steps, and the disc is gathered at every
  crossing, so the second and third disc images are in the sums. Whether
  a picture shows them is down to pixel size, and `rings` measures that.
- The disc's colour is a black body by radius. Its hottest temperature
  is picked, its brightness is the whole light's and not the visible
  light's, and the picture is tone-mapped.
