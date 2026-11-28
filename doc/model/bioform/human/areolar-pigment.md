# Areolar and papillary pigmentation

The areola and papillary centre use the chest's epidermal and vascular
colour model. Areolar absorption starts from the adjacent skin response.
The authored reference gives more contrast from melanin as overall
melanin increases, with less visible vascular tint. The papillary centre
has its own gradual change in pigment. A specimen's settings can produce
low or high regional contrast at the same skin tone, with independent left
and right colour. The underlying RGB coefficients are illustrative.

Histology in 20 women found more melanin per length of basement membrane
in the sampled areolar skin than in adjacent breast skin. That quantity
does not directly give a reflectance or a universal contrast ratio.
[Dean et al.](https://pubmed.ncbi.nlm.nih.gov/15710115/)

Reflectance measurements at the areola, breast, and inner arm found that
regional reflectance ratios varied differently from overall skin
reflectance across the study samples. The constructor offers individual
controls and assigns no population, ancestry, or reproductive-state
presets. Qualitative photograph references informed the range of edge
softness and visible contrast. Lighting, exposure and colour balance
prevent using those images as optical measurements.
[Pawson and Petrakis](https://digitalcommons.wayne.edu/humbiol/vol47/iss4/5/)

## Author an individual

`HumanSpec.WithAreolarTone` and `Torso.WithAreolarTone` accept the same
profile. Left and right refer to the specimen's own sides. All
adjustments run from −1 to 1, with zero selecting the authored reference.
`Areola.Melanin` and `Areola.Vascular` scale the regional additions over
adjacent chest skin. At −1 the corresponding areolar addition is zero.
`Papilla.Melanin` and `Papilla.Vascular` vary the papillary centre
relative to its areola and can reduce or increase local contrast.
`Chroma` shifts red, gold and olive while preserving local luminance.
`EdgeFeather` varies the breadth of the transition within the same
geometric footprint.

```go
left := primates.AreolarSideTone{
    Areola: primates.PigmentShift{
        Melanin: -0.35, Vascular: 0.22,
        Chroma: primates.Chroma{Red: 0.18},
    },
    Papilla: primates.PigmentShift{Melanin: -0.20},
}
right := left
right.Areola.Melanin = -0.23
right.EdgeFeather = 0.16
spec := primates.AdultHuman().WithAreolarTone(primates.AreolarTone{
    Left: left, Right: right,
})
```

The default material from `HumanPose.Surface(nil)` reads the region in
the same rest chart as the existing pigment and uses the areola and
papilla landmarks and dimensions shared with the chest geometry.
Shallow irregular colour variation follows that chart into every pose.
The colour controls alter no geometry. The same material model serves
both adult anatomy families. Individual controls require the typed
constructor and do not add scene configuration keys. The model has no
fit to measured spectra and the new parameter combinations have yet to
be visually evaluated in renders.

## Matched plate observations

In the `e1d165ac` chest plates, the small changes in colour on the light
specimen were comparable to the shading variation. The dark specimen
showed a dark areola with a diffuse border and a weakly distinguished
papillary centre. Both pigment centres sit on the visible relief. At
the 528 × 480 framing, the lens puts roughly one millimetre of the
front surface across a pixel. The old reference feather spanned about
10 mm at an 18–19 mm radius. A 4 mm papillary radius also limits the
visible separation of relief and colour at this framing.

As a check on relative contrast, weighted RGB display medians from both
sides were 123.7, 107.2, 99.1 and 104.0 on the light plate and 39.6, 33.1,
32.1 and 43.6 on the dark plate. These summarize pixels within distances
of 0–3, 6–11, 13–18 and 23–29 pixels of the approximate centres
`(154, 256)` and `(375, 256)`. These are display values after studio
lighting, shadows and development.

The CPU lateral diffusion pass spreads illumination and then applies
the local pigment at the surface hit. The
matched lighting is deliberately balanced and the visible material
uses one satin value. Those choices limit the surface detail that can
be judged from these frontal plates.

The refinement shortens the reference feather to about 7 mm while
retaining a soft, irregular edge. It adds subtle pigment variation
inside the areola at a few millimetres of scale. Small fluctuations of
the pigment and edge fade as the pixel footprint increases. The
papillary colour follows the existing relief and uses a smaller
melanin increment on darker skin with a subtle independent chroma
shift. Individual pigment and feather adjustments still apply.
The revised output has yet to be compared in matched renders.
