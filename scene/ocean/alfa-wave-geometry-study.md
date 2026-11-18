# Alfa and bounded wave geometry study

`TestAlfaWaveGeometryValidationDraw` is a small, CPU-only contact study. It
constructs the catalogued submarine with `manufacture.KindNamed("alfa")`,
`manufacture.MakeWithComplexity(..., "quick")`, and
`manufacture.Floating("alfa")`.

It lowers the actual named surfaced waterline (6.24 m above the model's keel
in the checked source data) to the generated field's mean height, zero. A
finite 128 m square wave-height field then intersects the surfaced hull; the
Alfa's 81.4 m model bounds fit inside it.

The sea inputs (wind 12 m/s, fetch 100 km, toward 0.2 radians, depth 200 m,
seed 41, time 18 s) are synthetic test values, not measurements of San Onofre,
Hawaii, or any real coast. The field is the existing vertical-only,
one-instant adapter from `NewHeightField`; it does not include horizontal
choppy displacement, a shore, or complete wave optics.

For this geometry view, the test copies each made Alfa material and replaces
its authored metal/coating response with a synthetic opaque diffuse color.
The field uses another synthetic opaque color. This keeps the model's generated
geometry and placements while deliberately removing optical-material behavior
from the study.

It does not apply Alfa's reserved `wet hull` look. There is no clear interface,
water volume, absorption, scattering, or underwater camera transport. No
movement or animation is implied.

To trace and retain the 256 × 144 one-ray CPU image, set
`SLAYER_ALFA_WAVE_STUDY_OUT` to a new `.png` path and run this focused draw test
through the Game tests lane:

```sh
SLAYER_ALFA_WAVE_STUDY_OUT=/path/to/new-alfa-wave-study.png game lane tests -- \
  go test -tags=draw -count=1 -v ./scene/ocean \
  -run '^TestAlfaWaveGeometryValidationDraw$'
```

The model's sourced dimensions and explicitly chosen geometry remain in
`manufacture/alfa.yaml`; its references are recorded in that file and the
catalogue. The test is an inspectable scene recipe in Go because it composes
the public construction API and sampled shape directly; it adds no scene YAML
loader, parser, water material, or renderer path.
