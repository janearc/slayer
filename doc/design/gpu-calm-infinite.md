# Exact calm infinite-water subset on Metal

An `InfiniteMeanContinuation` whose every core sample equals its authored
far-field mean is exactly one plane. Its bilinear core is constant and its
collar blends identical constant values. The existing CPU `flatAtMean`
invariant establishes this without a tolerance. `ExactPlaneLevel` exposes
that fact; a one-ULP core/mean mismatch still fails recognition.

The validated direct-Moon GPU route lowers only its exact tagged calm
continuation to the existing unbounded Plane primitive. The plane level is
transformed from local metres through the actual positive object scale and
world origin. The up direction must remain world up. The named optical
medium retains its authored IOR and metres-per-world-unit coefficients.

The existing source-plan `moon_plane.w` record explicitly uses -1 for this
unbounded case; positive values remain finite half extents.

Opaque lunar visibility excludes the uniquely validated water object. The
plan rejects the water medium on every other object, even opaque objects,
so a foreign same-medium object cannot gain the interface exemption. Local
propeller clouds must remain fully submerged. Incoming water/cloud Beer,
outgoing mixed integration and optional ray-spacing retain their meanings.

Generic finite collars and displaced infinite oceans remain refused on GPU.
There is no tolerance-based flattening or hidden wave omission. A production
recipe with waves must be explicitly changed to calm water before this exact
subset applies. Sky quadrature remains unsupported. The approved wavy CPU
recipes and published images remain separate from any calm GPU alternative.
