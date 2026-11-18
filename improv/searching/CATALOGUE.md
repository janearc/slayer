# Where the camera goes

The catalogue study is the first measured-position stage of heard.'s
pullback. `studies/heard-catalogue-720.yaml` pins a small product made by
celestial-data's `gaia-previs` from the retained Gaia DR3 radial-velocity
table. It has 4096 source IDs, each with its original row and measurements.

Galactic directions rotate into Slayer's right-handed y-up frame. Distances
are inverse positive parallaxes at signal-to-noise of at least 20, with no
zero-point correction. The source epoch is retained. These are approximate
display distances, not a new scientific distance catalogue.

The camera moves logarithmically from its near distance to its far distance
between two song cues. Each frame uniformly rescales the world and camera
for GPU precision. Relative positions and angles do not change. Marker radii
grow to stay visible; their glow, colour and travel time are authored.

The cyan marker stays at the Sun. This sample is incomplete and explicitly
ends at 500 parsecs. It cannot establish an empty region outside that radius,
the full Milky Way, Local Group or Laniakea. Those next stages need their own
catalogues or explicitly identified models before they enter this shot.

The earlier flow study remains separate, with wholly authored geometry.
Neither this Gaia product nor the retained Tully paper supplies a numerical
reconstructed galaxy velocity field. Pulses must not acquire that claim by
being placed beside a measured-position layer.

The render copies the pinned product beside its YAML and score, with `.txt`
views for the browser. The source table stays in its existing data location;
there is no second multi-gigabyte copy.
