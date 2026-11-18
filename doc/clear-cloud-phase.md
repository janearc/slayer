# Independent clear-cloud phase

`medium.NewClearCloud(field)` retains the legacy inherited water scattering
law exactly. `medium.NewClearCloudWithHenyeyGreenstein(field, g)` selects an
independent, authored Henyey–Greenstein cloud law.

The dimensionless parameter must be finite and strictly between -1 and 1.
Positive values scatter forward; negative values scatter backward. Explicit
zero is isotropic, `1/(4π)` per steradian. Omitted phase remains distinct:
Slayer's existing water parameter zero means Rayleigh. Density, bounds, albedo,
absorption and clock are unchanged.

`PhaseAsymmetry()` returns `(g, explicit)` and `Tick` retains that choice.
Each volume owns its override, so shared material or water coefficients cannot
couple different clouds. This is a normalized bulk optical approximation;
it does not infer a Mie solution from bubble radius or pressure.

CPU source evaluation selects the cloud law directly for local cloud samples,
including refracted sky quadrature where already supported. Water gaps retain
their own law.

Source visibility, incoming water/cloud Beer and outgoing mixed attenuation are
shared; no water-weighted source is divided to recover light. Phase selection
uses a small value rather than a capturing callback per occupied cell.

The supported Metal mixed route packs cloud asymmetry in `Volume.scatter.w`
with an explicit selection flag in `Volume.air.y`. Named water optics remain
untouched. The kernel uses the same source direction convention as CPU:
toward source dot camera-ray direction. Explicit HG zero never calls the
legacy Rayleigh-at-zero helper. Positive and negative lobes use a stable
denominator and `(1-g)(1+g)` numerator near the endpoints.

A double-valid parameter that casts to a float32 endpoint is refused on GPU,
rather than clamped. Nonzero parameters that would flush to zero are refused
too. These are backend precision limits, not changes to the constructor's
mathematical interval. Existing GPU density/source eligibility is unchanged.

The normalized HG model and isotropic limit follow the primary
[PBRT phase-function treatment](https://www.pbr-book.org/4ed/Volume_Scattering/Phase_Functions).
PBRT points both directions away from the sample; Slayer's existing cosine
convention yields the corresponding minus sign in the HG denominator.
