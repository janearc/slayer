# Residual gas bubbles reuse Fizz

The underwater shot separates two authored populations. `vapor-cavities`
represents short-lived rotor-tip vapor cores. `fizzes` can represent the
unresolved residual gas population in a downstream plume. Neither predicts
bubble inception, gas content, radius distribution or collapse dynamics.

This distinction follows the different dynamics of vapor and gas-bearing
bubbles discussed in Brennen, [Cavitation and Bubble Dynamics, chapter 3](https://media.library.caltech.edu/CaltechBOOK%3A1995.001/chap3.htm)
and [chapter 4](https://media.library.caltech.edu/CaltechBOOK%3A1995.001/chap4.htm).
Micron-scale gas nuclei and dissolved-gas effects are discussed in
[chapter 1](https://media.library.caltech.edu/CaltechBOOK%3A1995.001/chap1.htm).

These sources justify separate representations; they do not calibrate the
shot or establish that any particular operating state creates this plume.
Published [ship-wake observations](https://os.copernicus.org/articles/17/1285/2021/)
concern surface-ship conditions and are not substituted for a deep submarine
measurement.

## Existing constructor, oriented placement

`medium.Fizz` remains unchanged: its column doubles in radius from foot to
end, fades radially and longitudinally, and advects seeded churn along local
+Y. The fade is an authored density envelope, not a conserved gas-flux solve.

`world.fizzes.NAME.axis` optionally rotates that existing +Y axis into a
nonzero map east/north/up direction. Omission keeps the legacy upright pose
exactly. A downstream direction with a small up component gives a gently
rising trail without a new constructor.

For an illustrative starting recipe:

```yaml
fizzes:
  residual-gas:
    foot: [-1486.4218, -37.306, -55.226]
    axis: [0.342, -0.94, 0.03]
    radius: 0.6
    height: 20
    thick: 0.02
    rise: 10
    churn: 0.8
    seed: 705021
    colour: {srgb_linear: [0.84, 0.84, 0.84]}
    clear-water: true
    phase-g: 0.8
    source: "authored downstream gas-population approximation; not measured bubble dynamics"
```

With world unit one, lengths are metres. The legacy `height` names distance
along the oriented column; `rise` is churn advection along that local axis,
not an independently solved world-up terminal velocity. Tilt, footprint,
optical density, speed and persistence are explicit shot choices.

At 10 m range, an illustrative 2 mm diameter subtends about 0.11 pixel in a
960-pixel-high frame with an 80-degree vertical field of view:
`pixels = diameter * height / (2 * range * tan(fov/2))`. The 2 mm example is
not a predicted bubble size. Such unresolved populations need scattering
and extinction, not millions of individual sphere meshes or giant beads.

## Independent phase and native density

Optional `phase-g` on either `fizzes` or `vapor-cavities` selects the shared
explicit Henyey–Greenstein cloud model and requires `clear-water: true`.
The finite value is strictly between -1 and 1; explicit zero is isotropic.
Omission preserves inherited water-phase behavior, including its existing
zero-parameter Rayleigh behavior. This is a replaceable phase approximation,
not a radius-derived Mie calculation.

Native GPU Fizz is admitted only for an explicitly wrapped ClearCloud in the
supported lunar-water path. Standalone legacy Fizz keeps its baked density
representation and existing pixels. The native path evaluates the same Fizz
formula using one eight-word card (layout v1): radius, height, density, churn,
evaluated advection distance, canonical expansion factor, seed low and high.

The shared Volume supplies pose, finite AABB, material and optical metadata.
There are no particle meshes or sampled density grids on the new path.

Non-finite or subnormal nonzero float32 controls and non-normal squared
radii are refused. A numerical noise
budget limits the highest-octave coordinate magnitude to 2^20 cells, retaining
fractional precision and signed-index headroom; it is not a physical limit.

Zero optical density is a valid no-population control only with explicit
clear-water composition; standalone legacy zero-thickness refusal remains.
Field/phase omission, zero density and matched on/off images remain
separate comparisons.

A real underwater scene needs a supported render route and CPU/Metal volume
parity.
