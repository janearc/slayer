# Native adult torso study

The public typed factory is `manufacture.AdultTorso(primates.Torso)`. Start with `primates.Adult()`, change the desired dimensions, call `manufacture.Make(kind, manufacture.Sow(seed))` and place the returned parts with `Things`. The result is a single closed skin extending from a transverse slice through the pelvis to the base of the neck. The kind uses its own parametric anatomy. It has no imported surface, vertex weights or external maps.

## Geometry and posture

Sixteen trunk sections describe the ribcage, waist, pelvis and shoulder girdle in metres. Cubic interpolation produces continuous lateral width, anterior depth and posterior depth. The anterior surface includes the sternum, clavicular arches, anterior axillary pillars, lower costal wall, abdomen and navel. Each soft-tissue envelope has independent breadth, projection, vertical landmark position and areolar radius. The inferior pole moves down under load across its attachment footprint. Its lower turn blends into the chest wall before the medial and lateral ends. Papillary relief, the surrounding plateau and fine relief are vertices on the same surface as the trunk.

Rehnke and colleagues describe distributed fascial adherence around the breast and variation in the inferior attachment. Khatam and colleagues tracked the nipple, lower fold and axillary landmarks as orientation changed. These observations informed the locations and directions of the controls. `-measure` prints seven posed landmarks for direct comparison. Dimensions, stiffness and displacement amplitudes are authored choices. They have not been fitted against an individual scan or a measured population. [Rehnke et al.](https://pmc.ncbi.nlm.nih.gov/articles/PMC6211786/) · [Khatam et al.](https://pmc.ncbi.nlm.nih.gov/articles/PMC4297751/)

Arm elevation lifts the ipsilateral axillary wall and lateral envelope. Axial twist and forward pitch rotate transverse stations about the lower trunk. Gravity and compliance are independently adjustable. The age control applies a small increase in displacement under the same downward load, motivated by measured age-related changes in breast skin elasticity. The age series is illustrative. Morphology and pigmentation are separate controls. Physical posture prediction would require tissue-specific material and boundary data of the kind used in MRI-based finite-element models. [Coltman et al.](https://pubmed.ncbi.nlm.nih.gov/27800637/) · [MRI-based biomechanical model](https://pubmed.ncbi.nlm.nih.gov/30408760/)

## Appearance

The native material transmits RGB bands twice through a superficial pigment field over a blood-sensitive diffuse dermal return. Pigment varies across the areola, surrounding skin and fine-scale surface markings. Seeded submillimetre displacement gives the mesh a small amount of real relief. The study keeps the outer dielectric gloss low. This optical construction is informed by separate epidermal and dermal absorption in layered skin models and by measurements of regionally varying albedo and surface reflectance. Coefficients have not been fitted to spectrophotometer data. [Donner and Jensen](https://doi.org/10.2312/EGWR/EGSR06/409-417) · [Weyrich et al.](https://vcg.seas.harvard.edu/publications/20060730-analysis-of-human-faces-using-a-measurement-based-skin-reflectance-model/paper)

The optional `-tissue` witness samples direct diffuse irradiance and occlusion at vertices over the entire posed surface. It spreads a fraction across connected neighbours, using red, green and blue length scales of 2.40, 1.08 and 0.52 mm. The lengths of adjacent vertex intervals limit transfer across a fold. The material retains the native baseline lighting and adds the local diffusion correction. Final visible hits also interpolate two traced dielectric roughness responses using the rest-space regional roughness field. All three camera views share the same rest coordinates and pigment. This study handles a stationary directional source and a uniform two-sided fill. It omits through-thickness transmission, multilayer spectral transport and time-dependent mechanics. The smallest blue profile is below the spacing of some rings. [Jensen et al.](https://graphics.stanford.edu/papers/bssrdf/)

## Reproduction

Use Game's tests lane, with `GOMAXPROCS=2` and the local Game home. The native runs are in `.game5/runs/torso.game`.

```sh
game5 lane tests -- game5 run torso-study -- -view 34 -width 380 -out oblique.png
game5 lane tests -- game5 run torso-study -- -view 34 -width 380 -tissue -out tissue.png
game5 lane tests -- game5 run torso-study -- -view 90 -clay -out profile.png
game5 lane tests -- game5 run torso-study -- -form full -view 55 -arm 115 -clay -out elevated.png
game5 lane tests -- game5 run torso-study -- -form full -gravity 0 -measure
game5 lane tests -- game5 run torso-check
game5 lane tests -- game5 run torso-kind-check
```

The fixed review framing is a 0.41-radian lens from a 1.55 m orbit, with a 380 × 475 frame and two rays per pixel axis. The seed is 431. Compare geometry in clay first. Use frontal, three-quarter and profile views to distinguish a change in projection from a change in surface colour. Compare load, unilateral arm elevation and trunk rotation separately. The native output is a torso section. Upper arms, breasts' interior structures, the full skeleton, respiration, collision/contact, volume conservation and wardrobe attachment are outside this implementation.

### Witnesses and checks

At the review sampling, the skin has 119,394 vertices and 238,784 triangles, including its two end caps. The family test checks enclosure, finite control ranges, separation across the sternum, material registration, unilateral axillary motion and the direction of the load response. The manufacture test traverses the public factory, seed, manufacture and placement calls. A separate study test renders the native material and its optional diffusion witness.

In the original native geometry witness, for the full morphology at age 38, removing the downward load raised the left papillary landmark from 1.14017 to 1.17750 m. The central lower attachment moved from 1.10600 to 1.11412 m. At 115° left arm elevation the left anterior axillary landmark rose from 1.27674 to 1.30801 m. The right axillary landmark remained at 1.27715 m. These differences measure the generator's behaviour. Clinical error against matched observed subjects remains unmeasured.

The output directory contains individual PNGs and sheets for geometry, movement, morphology and optics. Values shown beside the age series also vary morphology and melanin, so those three renders cannot establish an isolated age effect. The pose, load and optics series each hold the other controls fixed.

The isolated age witnesses use the full morphology, clay material and a 55° camera. Only the 25-, 38- and 68-year age control changes. In this study age adjusts the vertical compliance response gradually. It does not select a body shape or pigment.

## Chest geometry and material controls

The revised native chest has a broad superior attachment over the thorax, a local lower-pole return and an independently modeled areolar plateau, papilla and low irregular tubercles. The lateral-attachment option projects tissue forward and outward from the curved chest wall. This revision changes geometry and optical appearance. Joint definitions and control values are held fixed.

The melanin control changes diffuse lightness. Three independent chroma controls redistribute linear RGB reflectance and renormalize it to the original Rec.709 luminance at each material sample. Red, golden, olive and cool are named review settings of these axes. They are artistic chromatic controls, without any assignment to ancestry or demographic group. Tests cover four rest-space locations at each of the three melanin levels and assert the luminance stays within 1e-12 of the neutral material.

Small pigment differences across shoulder cap, clavicle, axillary wall and chest articulation share the torso's rest coordinates. Areolar pigmentation has a soft, irregular margin, finer mottling and a distinct papillary centre. None of these fields receives the light direction or visibility. In the pigment-only witness two opposed uniform dome lights sum to constant illumination with the sky disabled. The lit view adds cast shadows and surface response. Elbows and the face lie outside this section.

The review skin matrix uses one specimen, seed 431, a 24° camera and the same area light, fill and whole-surface transfer. The three row values of melanin are 0.12, 0.50 and 0.95. All five columns use the same pose and tissue geometry. Render an individual cell with the following command.

```sh
game5 lane tests -- game5 run torso-study -- -width 370 -view 24 -lateral-attachment -melanin .50 -undertone olive -tissue -out mid--olive.png
```

For the frontal, oblique and profile geometry witnesses use `-width 440 -lateral-attachment -clay` and views of 0°, 34° and 90°. For the areolar detail plates use `-chest-detail`, whose 0.232-radian lens targets the chest rather than the entire torso. The color controls are not fitted to an individual measured reflectance spectrum.

The revised profile has a rounded lower turn. The frontal clay render still shows an extended medial groove beside the sternum. The smooth shoulder termination and cropped neck also identify the model as a section. These remain visible limits on anatomical fidelity.

## Thoracic wall and proximal upper arms

This revision separates the bilateral pectoral bed from a compact mobile lobe on each hemithorax. The bed crosses the midline continuously. The lobe's medial boundary and inferior return follow its own footprint. The prior geometry multiplied both by one 52 mm lateral gate, then subtracted a separate stripe beneath a downward-shifted lobe. That construction made a long trough next to the sternum and an independently drawn lower fold. The current lower turn comes from differential displacement between the lobe and its restrained perimeter. The seventh-landmark report includes the steepest forward rise on each inferior sagittal profile as a surface proxy. It does not establish contact or diagnose a clinical inframammary fold.

The mobile load is evaluated relative to world gravity in the pitched and twisted chest frame. Mobility varies across the tissue footprint. Arm elevation on the torso-only form affects the axillary wall and ipsilateral surface. Papillary radius and projection are separate metre inputs on each side. Relief projects along the native posed surface normal. Nine small, irregularly spaced elevations use independent widths, radii and amplitudes on the areolar plateau.

Use `-legacy-contour -lateral-attachment` for the previous masked wall and `-legacy-contour -lateral-attachment -undertone olive` for its previous optical mapping. The default wall is the engineering revision. `-sections` writes the anterior surface depth in metres at 2 mm lateral intervals on three horizontal planes. At each height, the average of the two points 40 mm from the midline lay this far forward of the sternum:

| Height | Prior | Revised |
| --- | ---: | ---: |
| 1.18 m | 29.39 mm | 22.66 mm |
| 1.22 m | 15.81 mm | 6.59 mm |
| 1.26 m | 0.63 mm | −3.78 mm |

The upper trough terminates on the supporting wall. Lower shadow pockets and angular boundaries from the finite disc-light samples are still visible in the clay views. The matched review set holds the camera, lamps, skin seed and sampling fixed in frontal, profile, three-quarter and grazing-light comparisons. The shallow and full morphologies use the same upright gravitational load.

`-proximal-arms` adds a neutral-pose shoulder continuation. Two patches of lateral torso faces are removed. Their boundary vertex IDs become the first arm cross-sections. The next cross-sections match the boundary surface tangents and turn from the shoulder through the deltoid toward the upper arm. The arm meshes end above the elbow. The coronal charts retain the actual torso perimeter, distal section and silhouette rails during interpolation. The chart orientation check addresses local foldovers found in the first tube construction.

The topology check counts exactly two oppositely oriented uses of every edge, traverses every vertex from one component, and checks Euler characteristic 2. Rays through the armpit must pass through air, while nearby rays must strike the arm. In an 800-ray frontal probe the initial tangent-joined arm had 22 of 740 occupied rays with extra surface layers. The corrected front chart produced zero extra layers at both review resolutions. At −15° and +15°, the 208 × 143 trunk grid produced 2/729 and 0/754 extra crossings, while the 416 × 286 grid produced 0/731 and 1/756. Oblique rays can cross a concavity more than twice. This local sampling does not constitute a global self-intersection proof.

The optional material study now integrates over connected edges of the posed triangular mesh, including the arm graft. Triangle areas weight the diffusion kernel. Its three radii come from the optical response. The approximately 2.29 mm vertical grid interval exceeds the 1.08 and 0.52 mm green and blue radii on much of the trunk. Edge distance approximates geodesic distance. Regional roughness is a per-hit interpolation between two native dielectric renders. The skin study omits volumetric, spectral and through-thickness transport. The kernel is interpolated at visible hits and uses the study's uniform fill and one directional source.

The shoulder is a topology and shape demonstration in neutral stance. It has no scapulothoracic, clavicular or glenohumeral joint solver; non-neutral poses with arm extensions are refused. The remaining body requires a connected craniofacial/neck surface, arm-to-elbow/hand anatomy, and a bifurcated pelvic floor continuing through hips and legs. The present neck, trunk and arm ends are closed study sections. Tissue positions are authored load responses rather than material stresses or a volume-conserving biomechanical solve. Neither the matched images nor the transverse and socket checks establish population or clinical accuracy.

## Figure-reference comparison

The later geometry study follows the flank over the lower ribs, the shallow upper pectoral plane into the axillary tail, and the different turns of upper and lower soft tissue. Five supplied photographic references were viewed locally. Their postures, lighting and viewpoint differ from the controlled neutral study. They informed these general observations only. The generated surface and material use native inputs.

The transverse wall gains a flatter anterolateral costal section, a narrower caudal flank, paired neck-to-clavicle transitions, lower abdominal convexity and a pelvic-crest transition. Tissue follows circumferential distance on the thorax. A cubic spline separately describes anterior projection and mobility at each vertical station. Superior and inferior boundaries change across the arc. The projected vertical coordinate can reverse locally under a sufficiently high load. This is an authored shape response without contact or tissue stress solving.

Generate the four matched views with the same seed and factory reference form using the existing `torso-study` Game run. Use `-width 360 -proximal-arms -clay` for the whole series. Set `-view 0`, `-view 90`, `-view 34` and `-view 34 -lateral`. Use `-vertical-sections` to sample anterior depth every 2 mm at world lateral offsets of 0, 40, 80 and 115 mm. Each ray selects the first surface. A separate direct chart test measures the inferior return under load, including portions hidden from frontal rays.

The prior and revised renders, the two section CSV files, and a plot are in the local output directory `out/reference-thorax/`. The four revised individual views have the prefix `pole-`. A separate `asymmetric-rotated.png` uses `-form asymmetric -twist 24 -pitch 9 -arm 42 -view 34 -clay`; it omits the neutral-only upper arm continuation. The clavicular ends and axillary junctions remain simplified and the neutral inferior contours show closely matched bilateral heights. The section checks measure the shape change in this constructor, without a population or clinical accuracy claim.
