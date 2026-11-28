# Procedural irises

`bioforms/ocular` supplies an opaque eye texture in the globe's rest chart.
The chart origin is the globe centre, +Z is the gaze direction, and +X and
+Y span the iris. Supply the limbal radius and footprint in the same units
as the chart. The native human uses metres in its canonical chart. The CPU
tracer interpolates that chart at a posed mesh hit, so the pigment follows
the globe and turns with its gaze. Changing body scale or viewing the eye
obliquely changes the physical footprint relative to the chart. The
texture uses an approximate width and fades the finer marks with distance.
Metal does not draw meshes with this rest-point texture.

## Iris structure and pigment

- The pupil is a dark ellipse. Its half axes set a round opening or an
  approximate slit at an authored rotation. The pupillary margin has a
  thin, irregular dark ruff. The default human pupil is about 3.8 mm
  across in the chart, a picked aperture. A slit profile is a static
  approximation of one of the pupil shapes surveyed by Banks and colleagues.
  [Comparative pupil measurements](https://pmc.ncbi.nlm.nih.gov/articles/PMC4643806/)
- The inner field and outer stroma meet around an irregular collarette.
  Seeded fields vary both around and across the iris. The radial bundles
  change strength and break into thinner, mostly radial filaments with
  different spans. The borders of the inner pigment follow that structure.
  Crypts near the collarette have shaded centres and narrow rims modulated by
  the neighbouring fibres. Outer contraction furrows follow short,
  irregular arcs. The darkening and width of the limbus vary as it
  blends into the sclera. Sparse scleral vessel colour can be enabled
  independently. The marks use authored reflectance, widths and counts.
  [Observed iris texture traits and photographs](https://pmc.ncbi.nlm.nih.gov/articles/PMC3155193/)
- The brown, hazel, amber, green, blue and gray choices each use picked
  linear RGB for the stroma and its inner pigment. The two colours can
  also be supplied directly. Melanin measurements establish differences
  between light and dark irides. In-vivo hyperspectral measurements show
  variation with both position and colour. The RGB palette approximates
  their visible effect in ordinary light. There is no fitted melanin
  concentration, wavelength-dependent tissue scattering, or model of
  populations and genetics. [Stromal melanosomes](https://pubmed.ncbi.nlm.nih.gov/8602783/),
  [melanin measurements](https://pubmed.ncbi.nlm.nih.gov/16280011/),
  [in-vivo iris reflectance](https://doi.org/10.1117/1.3595710)

## Native human corneal surface

At reference scale the dome of the front globe joins the sclera with
an authored tangent. The horizontal centre radius is approximately 7.8 mm
in the analytic shape before tessellation. The default horizontal limbal
diameter after bulging is approximately 11.8 mm. These dimensions use the
anterior corneal curvature and white-to-white diameter measured in adult
samples as targets. The shaped globe has no measured aspheric surface or
limbal topography. [Corneal curvature](https://pubmed.ncbi.nlm.nih.gov/16266736/),
[white-to-white diameter](https://pmc.ncbi.nlm.nih.gov/articles/PMC4877715/)

The exterior supplies the shape and smooth normal for scene reflections
and GGX lamp highlights. Its effective dielectric index of 1.337 follows
human tear refractometry. Gloss and satin are authored for the opaque
approximation. Iris and pupil colour are read on the same surface as
sclera and corneal reflections. Corneal transmission and refraction, the
anterior chamber, a separate tear film and meniscus, iris microgeometry,
retinal effects and light-driven pupil dilation are outside this model.
[Human tear refractometry](https://doi.org/10.1002/j.1538-9235.1995.tb03963.x)

The manufactured human studies use the CPU renderer's lateral diffuse
response on the opaque globe. A share of 0.32 mixes neighbouring
illumination into the diffuse light, with linear RGB widths of 1.15,
0.86 and 0.66 mm. Human sclera strongly scatters visible light. The three
widths and share are authored approximations. The same response applies
on the exposed iris and sclera because they share the opaque surface.
Sampling begins at mesh vertices, so short widths can be under-resolved.
[Human scleral optical measurements](https://doi.org/10.1002/lsm.1900110404)

## Native lid and globe fit

The neutral opening is approximately 26 mm wide and 9.5 mm high through
the pupil in the unposed chart. Separate upper and lower contours place
their extrema on opposite sides of the pupil. The inner and outer corners
have different depths. These values are chosen for the reference adult
from the scale and variation of published observations. [Adult fissure measurements](https://pmc.ncbi.nlm.nih.gov/articles/PMC10371930/),
[three-dimensional canthal measurements](https://pmc.ncbi.nlm.nih.gov/articles/PMC6908328/)

The shallow lid roll follows the envelope of the aspheric cap over the
allowed gaze range at its exposed edge. The envelope returns to the
scleral radius where the moving cap cannot reach. The roll turns onto the
analytic neutral globe at its inner edge, which overlaps the globe
slightly inside the opaque surface and blocks views into the socket's
closure. The temporal corner ends nearer the sclera than the medial
corner. The eyelid patch blends into the broad orbital wall. This is an
authored contact approximation. It omits separate marginal mucosa, a
modeled caruncle and tear meniscus, and mechanical motion of the lids with
gaze. [Human lid margin histology](https://pubmed.ncbi.nlm.nih.gov/21413985/)

## Author an eye

`HumanPose.Eyes` accepts independent yaw, pitch and appearance for each
eye. A zero `Gaze` looks forward with brown irises and distinct seeds.
A non-nil `EyeAim.Appearance` supplies the complete texture. With a nil
`Appearance`, a nonzero `EyeAim.Iris` sets both base colours of the brown
profile and zero `Iris` uses the default palette. Aim turns the globe,
while the native lids remain static. To use a different pigment profile
or seed, build the texture once and reuse it as the figure poses.

```go
spec := ocular.Human(ocular.HumanIris{Colour: ocular.Hazel, Seed: 72})
left, err := ocular.NewSurface(spec)
if err != nil {
    return err
}
spec.Seed = 73
right, err := ocular.NewSurface(spec)
if err != nil {
    return err
}
gaze := primates.Gaze{
    Left:  primates.EyeAim{Appearance: left},
    Right: primates.EyeAim{Appearance: right},
}
```

Other bioforms can supply their own chart, radii, palette and feature
strengths to `ocular.NewSurface`, then use it as the material texture on
their own globe. Changing the species also requires choosing its globe
shape, dimensions, orientation and surface optics. The human mesh is
specific to the native human rig.
