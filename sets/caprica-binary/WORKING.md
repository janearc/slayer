# Working system revision, not yet rendered

`working-system.yaml` is a proposed revision of the preserved
`system.yaml`. The original records the wisps, lights, tails, comets,
cameras and Caprica appearance inputs as handed off. Shot selection and
rendering for this revision are pending.

- The existing `the-hot-giant` key is renamed `amber`, without changing its
  position, `hot-jupiter` template, tilt, tail, illuminated crown wisps or
  cameras. Its authored position is about 0.29026 AU from the primary at
  the selected epoch. This maps the close giant in the brief to the existing
  body in the draft. The template's bands are cream, ochre and rust.
  Whether the result reads as the intended yellow Amber remains open.
- `azure` is a new generated blue, ice-rich giant at a fixed position. Its
  provisional map center is [-3830749.61307, 2213848.79561, 0] in the
  scene's million-metre units: 20 AU east of the binary's primary at the
  selected epoch. This is an invented composition choice, not a fitted
  orbit or a claim that Azure moves with the primary. Its 17.15-Earth-mass
  input, ice/rock/metal/gas fractions, age, rotation and blue palette are
  artistic choices; `giant_look.made_up: true` records the appearance. The
  input primary mass and luminosity are 2.45 solar masses and approximately
  50.629244 solar luminosities as reported by `system-anchor`. Both actual
  binary-star lamps and Fresno remain staged by the existing system.
- Azure uses the existing generated **painted-band gas** constructor, not
  the separate flowing-weather bridge, which refuses binary lights. The
  earlier standalone blue palette was reused as art material, not cited as
  evidence of Azure's name or history. The selected colors, scattering,
  effective climate and appearance have not been shown in this scene.
- The proposed `azure-orbit` camera is authored near Azure; neither it nor
  the planet has been rendered. The dozen soaring Hatzegopteryx called
  for in the brief and an in-atmosphere camera with both suns are not yet
  present. Those are the next coordinated model and staging tasks, not
  behavior already implemented by this set.
- The `caprica` eye and target are moved from the draft's inconsistent
  north -2052-million-metre position to the prior Caprica look-book's day
  deck eye [9.08, -2.43, 3.5] and target [5.261, -3.684, 3] in this set's
  units. `drop-from-caprica` now uses the prior cloud-skim eye
  [6.2, -1.6, 2.3] and points toward Fresno. These are proposed camera
  corrections based on the anchor calculation, not evidence that the new
  atmosphere, wake and lights compose or expose correctly. The original
  coordinates are preserved unchanged in `system.yaml`.

The scene is a still layout: the hot-Jupiter and Azure centers are authored
fixed positions; no orbit around the primary is implemented for either.
The pair and Caprica's existing orbital constructor remain as in the draft.
The 48-AU system places Azure at roughly millions of scene units while an
11-metre wingspan is only 0.000011 units. If represented in global float32,
the horizontal coordinate's spacing near Azure is on the order of 250 km;
whether the relevant renderer rebases all needed geometry and lighting is
not established. A usable close animal shot requires a measured local-origin
strategy, not a promise that this snapshot already preserves metre detail.

Fresno's long comet inundation is part of the system's fictional history.
The preserved haze and wake are artistic proxy wisps, and no deposition or
atmospheric evolution is simulated. The four comet fragments are still
static hand placements, and the luminous crown remains spotlighted
scattering wisps rather than a simulated aurora. The draft's wind stream and
captures are retained, not yet assessed. This revision retains the prior
documented source and artistic provenance in `DIRECTION.md` and the model
source notes under `doc/model/bioform/hatzegopteryx/`.
