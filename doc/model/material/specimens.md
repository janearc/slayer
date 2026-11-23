# Porous material specimens

`materials/specimen` accounts for the current solid phases and pore water
in a physical specimen. It is the first foundation for composition-driven
construction materials. It supplies mass, volume and density, not an optical
shader or a hydration solver.

## Specify the state being rendered

A batch recipe describes what was mixed. A hardened specimen describes what
exists at the chosen age and conditions. Cement reacting with water changes
the phase assemblage. Water chemically bound in a solid phase belongs to
that phase's mass; free liquid water belongs to the pores.

Supply each identified solid's current mass, intrinsic density and basis.
Supply total porosity, pore saturation and water density separately. A basis
is a measurement citation or an explicit assumption. The constructor checks
that it is present; it does not verify the evidence behind the text.

Solid names are case-sensitive and must not have leading or trailing
whitespace. Saturation is specified over total porosity, including closed
pores: it describes water already present, not how much water can enter
from outside. A separate accessibility or transport model is still needed.

Water density remains required even at zero porosity or saturation, and
saturation is retained as supplied when porosity is zero. Those inputs
contribute no water mass in that case; this is an explicit input contract.

All specimen units are SI: kilograms and metres. This differs from the
parent `materials.Material.Density`, which uses grams per cubic centimetre.
Do not pass the latter directly without conversion. An apparent density that
already includes pore volume would count pores twice in this model.

For solid masses m_i, intrinsic densities rho_i, porosity phi and saturation S:

```text
V_solid = sum(m_i / rho_i)
V_bulk  = V_solid / (1 - phi)
f_i     = (m_i / rho_i) / V_bulk
f_water = phi * S
f_empty = phi * (1 - S)
rho_dry = sum(m_i) / V_bulk
rho_wet = rho_dry + f_water * rho_water
```

These are volume and mass identities, not an empirical appearance fit. Empty
pores have zero mass here: neglecting air mass is an explicit approximation.
Dry density excludes free pore water without simulating oven drying or
dehydration. Changing saturation describes another specified state; it does
not simulate swelling, shrinkage, water transport or reaction.

## A numerical specimen

This is an analytic fixture with invented constituents, not a building mix.

```go
made, err := specimen.New([]specimen.Solid{
    {Name: "matrix", MassKG: 2, DensityKGPerM3: 2000,
        Basis: "analytic fixture, not a measured material"},
    {Name: "inclusion", MassKG: 3, DensityKGPerM3: 3000,
        Basis: "analytic fixture, not a measured material"},
}, specimen.Pores{
    VolumeFraction: 0.2, Saturation: 0.25, WaterDensityKGPerM3: 1000,
    Basis: "analytic fixture, not measured pore conditions",
})
if err != nil {
    return err
}
```

Each solid occupies one litre. The two litres of solid become a 2.5-litre
bulk specimen. Each solid occupies 40% of bulk volume; water occupies 5%
and empty pores 15%. Dry density is 2000 kg/m3 and wet density 2050 kg/m3.

The constructor rejects missing evidence labels, duplicate names, invalid
fractions, nonpositive masses/densities, nonfinite values and unrepresentable
bulk calculations. Readback cannot mutate a constructed specimen.

Numerical range checks reject nonfinite results and zero solid volumes or
dry density. They do not promise relative accuracy for subnormal volumes;
liquid water mass can also round to zero at extreme scales. A zero-value
`Specimen` is unconstructed and returns NaN densities, not a physical zero.

## What a cement-and-aggregate material still needs

The specimen does not infer phase identity from oxide analysis, infer grain
sizes from a maximum aggregate size, or infer hardened pore water from batch
water. Those require additional evidence and physical models.

Spatial structure also matters. Garboczi and Bentz's
[multiscale microstructure model](https://tsapps.nist.gov/publication/get_pdf.cfm?pub_id=860125)
distinguishes aggregate, bulk paste and an interfacial transition zone with
different microstructure. Their paper concerns ionic diffusivity; it is not
an optical scattering law.

A list of volume fractions alone cannot supply
the missing geometry or visible reflectance. No authored RGB fallback is
supplied by this package.
