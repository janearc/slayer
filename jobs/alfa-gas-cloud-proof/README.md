# Separate vapor cores and residual gas

These 540×480 stills examine a synthetic propeller flow under moonlight, at
two rays per axis. Residual Fizz is a separate authored population. The
renderer does not convert vapor into noncondensable gas.

`vapor-only` retains the approved stern camera and Moon azimuth of 230°.
The four `backlit301` controls use a separate composition: Moon azimuth 301°,
altitude 40°, the existing below-stern detail camera, and a 65° vertical field.
This brings the view closer to the direction of transmitted moonlight. The
source strength, water, material, clock of 0.7 s, and ten-stop gain stay fixed.
No lamp or emission is added.

The chosen pressure recovery of 300 Pa/m permits a 40 m field, replacing the
earlier 1000 Pa/m and 12 m visual lifetime. This is an authored lifetime, not
measured pressure. Core radius 0.12 m and density 0.8/m stay fixed. Wander of
1.2 m at a scale of 3 m disturbs the aft cores. Spreading dilutes them by the
area ratio instead of increasing cross-sectional optical density. These
controls are not CFD.

`backlit301-vapor-only` uses spread 0.18 m/m. The matched residual-on/off pair
uses spread 0.10 m/m: `backlit301-gentle-vapor-only` and `backlit301-wide004`.

The latter adds residual gas with radius 2.1 m, height 50 m, density 0.04/m,
churn 1, rise 10 m/s, HG asymmetry 0.8, seed 705021, and neutral 0.84 albedo.
Its birth is 0.2 m aft of the screw, inside the near-core region.
`backlit301-wide` lowers only that residual density to 0.012/m. These are
chosen optical footprints and persistence, not measured bubble radii,
counts, gas fractions, or cavitation noise.

The stronger matched candidate shows localized aft haze and contrast loss.
Near-core patterns remain visible and the hull stays dark. Fizz still has a
sharply bounded birth plane. Placing it at the screw avoids a detached plane
in this composition; it does not introduce a new fade-in algorithm.

Run the named `alfa-gas-cloud-*` Game targets. Outputs land under
`out/alfa-gas-cloud-proof`. The actual proofs used clean source `4dca3141`.
Exact jobs, raw images, and broader density sensitivities are retained beside
the shared proofs. Original approved images and the orbit film are unchanged.

Blade geometry retains the fixed pose from the approved comparison scene.
The clock advances the density fields; these stills are not a rotor-motion
proof. Separate CPU/Metal blade-motion controls cover that feature.
