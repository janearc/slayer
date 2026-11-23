# Nickel-aluminium bronze

`nickel-aluminium-bronze` joins the shared material library beside titanium.
It is an authored C95800 nominal composition: Cu 81.5%, Al 9%, Ni 4.5%,
Fe 4%, Mn 1%, by mass. These values sum to one and fall within the
[manufacturer’s C95800 limits](https://www.concast.com/c95800.php).

They are a selected composition, not a certified sample analysis. The
manufacturer’s density 0.276 lb/in³ converts to about 7.64 g/cm³.
Atomic proportions are mass shares divided by each element’s atomic weight.
Cu and Mn atomic weights follow [CIAAW](https://www.ciaaw.org/atomic-weights.htm).

Its optical appearance is explicitly an estimate. Mullaney and Tatam,
[Metals 15(10), 1078 (2025)](https://doi.org/10.3390/met15101078), Figure 6,
measured near-normal reflectance of deposited UNS C63280 specimen K.

Approximate plot readings at 610, 550 and 465 nm are 0.46, 0.40 and 0.27.
The ±0.01 reading allowance is digitization uncertainty, not the authors’
measurement uncertainty. C63280 and C95800 are different grades; this
sample is an appearance proxy, not a measured C95800 optical table.

`ConductorFromReflectance` supplies a generic, nonunique closure: n = 1,
k = 2 sqrt(R/(1−R)). The conductor Fresnel equation then reproduces the
selected normal-incidence air reflectance. `Source.Estimated()` marks
these derived readings.

It does not identify actual complex indices,
recover a BRDF from integrating-sphere measurements, or establish how a
polished, oxidized or seawater-exposed propeller reflects at every angle.
Finish remains a separate choice through the existing material/finish API.

Existing bare-metal reflection currently evaluates air-relative constants
even inside water. This material addition does not fix or conceal that
transport limitation. It also does not establish the Alfa’s real screw
alloy. Existing titanium defaults and recorded renders stay unchanged.

Use the same constructors as other shared materials:

```go
alloy, err := materials.ByName("nickel-aluminium-bronze")
// Handle err before constructing the finish and look.
finish, err := thing.FinishOf("satin")
// Handle err before combining them.
look, err := thing.Made(alloy, finish)
```

`thing.NickelAluminiumBronze()` also supplies the shared conductor optics
for callers that construct a material explicitly. It reads the same
library values; no second bronze table exists in the renderer.

Run `game run check-bronze` for shared-material and finish checks, and
`game run check-bronze-metal` for actual CPU/Metal air-side reflection
at normal, oblique and grazing angles. The latter verifies the declared
estimate reaches both renderers; it does not validate a measured alloy BRDF.
