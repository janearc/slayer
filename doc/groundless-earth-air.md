# Groundless Earth air

`groundless-air` is an explicit opt-in for staging a terrain-free scene under
the existing standard-Earth atmosphere. It requires all three fields:

```yaml
air: earth
groundless-air:
  coordinate-origin: earth-mean-radius
  reference-altitude: {value: 12, unit: m}
  boundary: earth-standard-land
```

The coordinate origin is explicit: east and north are local tangent-plane
offsets in metres; up is metres above the Earth mean-radius datum used by
`airs/earth.yaml`. The reference altitude is also metres above that datum.

It selects the one atmosphere column sampled for the sky and direct sun; it
does not assert that every camera or object is at that altitude. It must lie
from 0 m up to, but below, the 86 km top of the existing atmosphere model.
This datum is not asserted to be a geodetic sea-level datum or a terrain
datum.

`earth-standard-land` deliberately acknowledges the current model boundary:
Earth's atmosphere uses the existing fixed 0.15 grass/soil albedo and its
continental aerosol approximation. These are not ocean reflectance, coastal
measurements, weather, or site-specific photometry. The choice does not
silently turn a groundless scene into an offshore or water model.

The scene must name `air: earth` and the explicit groundless reference. A
daylight scene uses `light: sun` and a sun direction. A full-Moon scene may
instead use `light: moon` and a `moon` azimuth/altitude; it follows the
existing `moonlight.Full()` source and Moonlight sky path at the declared
reference altitude.

This route does not treat the Moon as a solar spectrum or call airlight's
solar calculation. The Earth air source still identifies the tangent-plane
atmosphere and `earth-standard-land` lower-boundary approximation.

Every manufactured set thing must have an explicit absolute `position`;
terrain-relative stands, text, ecosystems, fire, lasers, terrain and local
atmosphere layers are refused. With a world block, `world.unit` must be 1 so
ocean, manufactured coordinates and local tangent offsets share metres.

A world ocean and other non-lighting world geometry may coexist. A world sky,
lamp, spot, sun or flood is refused because the selected atmospheric lighting
path supplies the scene sky and distant source.

In daylight, the sun azimuth and altitude remain the authored local direction;
airlight does not derive them from scene latitude, longitude or time of day.
Planet and space settings are also refused.

The old terrain-backed path still samples terrain and rebases absolute
positions against its terrain window. Ordinary world-only scenes still use
their authored world sky and lighting.

A world ocean is only an interface; this option adds no water volume,
underwater attenuation, in-scattering, shoreline, or coastal reflection model.
Daylight airlight needs its pinned solar spectrum on the configured source
shelf; a missing or invalid pin is an error. The Moonlight route uses its
separate full-Moon source and sky model.

The `groundless-air` key is available in both published ecosystem scene
schemas. It is scene-level configuration and does not modify the atmosphere
source data.
