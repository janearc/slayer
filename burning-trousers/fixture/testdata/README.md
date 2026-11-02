# Colour-matching fixture

`ciexyz31_1.csv` contains the CIE 1931 2-degree standard observer from
UCL’s Colour & Vision Research Laboratory, <http://www.cvrl.org>.
It covers 360–830 nm at 1 nm intervals, with columns for wavelength,
x-bar, y-bar, and z-bar.

The harness keeps its own copy so its expectations do not depend on the
tracer or the libraries the tracer uses. Treat the file as source data:
replace it from the published source rather than editing individual values.
This copy matches the one in libtheme at `e0d044e` byte for byte.
