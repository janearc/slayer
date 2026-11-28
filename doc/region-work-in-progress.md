# Street region measurements

see [draw a region](frame-regions-plan.md) for the current interface
and [repeatability control](region-control.md) for the small scene.

the older study kept five street rectangles, crops from the original
image and fresh redraws. the original build and GPU mode were
unknown, so those pairs do not establish equality.

At one ray per side, a fresh water region and a crop of a same-build whole
street frame produced identical delivered pixels in a BMP byte comparison.
Raw-light equality, other sampling rates and the remaining street regions
still need same-build checks.

A smaller water patch was drawn at 1, 2, 4, 16 and 32 rays per side. Raw-light
scores against the 32-ray reference are recorded in
[water report](region-water-linear-report.txt); they are not monotonic. The reference is
finite and not proven converged. Repeat developed images matched at each
setting, but raw-light variation was not measured.

The wall patch has the same raw sampling sequence, recorded in
[wall report](region-wall-linear-report.txt). Its mean absolute difference falls as
sampling rises, while its isolated-positive count does not. The reference is
finite and not proven converged. Haze, lamps and the street end still need
the sampling sequence and raw comparison.

The original street render is recorded in the render notes as four rays
per side, sixteen samples per pixel, on the GPU with `street-2k.yaml` and
its chosen exposure. The renderer's exact GPU mode and complete input closure
still need confirmation. A request for 64 samples per pixel corresponds to
eight rays per side; 64 per side would instead mean 4096 samples per pixel.
