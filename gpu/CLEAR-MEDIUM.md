# Metal clear-medium transport

The opt-in GPU API is `SampledWithInitialMedia` or
`Frames.DrawWithInitialMedia`. Both take `trace.InitialMedium` values.

Frames also provide `DrawOpenWithInitialMedia`, which refuses a shutter-moving camera:
this first implementation does not infer a new medium stack as the camera
moves.

Fixed set cameras can author the stack as `camera.initial-media`; the session
passes that validated stack through its ordinary fixed-camera GPU draw and
retained-frame paths, preserving the selected traversal mode.
The set-camera Metal path is currently pinhole-only.

It explicitly refuses a
set rig or job thin-lens aperture: their ray origins are spread over a physical
aperture, and a point camera's authored stack does not establish that all of
those origins lie inside the declared media. CPU tracing retains its existing
rig and thin-lens behavior. No rig/aperture Metal parity is claimed.

The packed scene assigns each named medium one local ID. `Material.info.w`
stores that ID; the field was unused by Material readers and the `int4` layout
is unchanged. Each table row is a 16-byte `float4` containing RGB extinction
per metre and metres per world unit. The per-draw initial state is 272 bytes:
one 16-byte count header and sixteen 16-byte held-medium rows. It is sent once
with the other scene inputs, not allocated per ray.

The shader's private `Held` grows from two scalars (8 bytes) to three (12
bytes) only for optical-state kernels. Its fixed 16-entry `Inside` stack grows
from 132 to 196 source-level bytes per tracing thread, a 64-byte increase.
Those numbers describe Metal struct storage, not compiler register allocation
or occupancy; those remain device-dependent and unmeasured.

An optical-state scene uses 19 common scene buffers instead of the ordinary
17. Native traversal appends its existing first-face buffer after those. The
ordinary input list remains 17 buffers. Clear HeightFields now carry a
separate geometric interface normal within the shader's private hit record.

Fresnel, refraction, total internal reflection and boundary offsets use that
normal; spectral slopes continue to control surface appearance. This also
updates the ordinary shader source because clear boundaries work without an
initial medium stack. The source-hash regression records that deliberate
change; it does not claim byte-identical shader text.

Real-device controls on Apple M3 Max cover an inconsistent spectral normal
that would otherwise cause false total internal reflection, and a genuine
60-degree TIR case. Both match CPU pixels exactly in those tests.

The existing
96-by-64 spectral-water comparison retains its original tolerance (mean
0.0002, maximum 0.05); observed differences were mean 0.0000636 and maximum
0.00560. A separate opaque spectral-shading control and the crinkled-blanket
comparison guard appearance behavior.

These are specific tested scenes, not
a general transport-parity claim. Infinite mean-plane continuation remains
CPU-only.

GPU packing refuses coefficients that cannot be represented as positive
normal float32 values; zero extinction remains exact. The shared trace
preflight still rejects named extinction combined with volume transport or
gravity.

GPU also explicitly refuses positive in-scattering, composite passes
with optical state, and an explicit initial stack on the separate gravity
kernel. Extinction affects the observed ray segments; it does not attenuate
lamp-to-surface paths or add in-scattering.

Set staging rejects a moving job camera with an authored fixed initial stack.
Composite passes and pixel replay refuse explicit initial media before
drawing. A static camera does not cause Slayer to infer or track which medium
contains it as animated boundaries move.

The packing, validation, source-generation and early-refusal tests run without
a device.

A coordinated headless 1×1 device batch passed on an Apple M3 Max at
source commit `4d29bd23045b364df20879af3cd8a39195631776` on branch
`metal-clear-medium`. The recorded test command was:
`game5 lane gpu -- go test -tags draw -p=1 -run '^TestMetalInitialClearMediumExtinctionMatchesCPU$' -count=1 -timeout=180s -v ./gpu`.

The test included the legacy no-optics control, a zero-extinction optical
control, camera-water exit, air-to-water entry, 60° water-to-air total
internal reflection (TIR), and rank-2 glass nested in rank-0 water.

The legacy
control was CPU `(0.9799407, 0.9799407, 0.9799407)` and Metal
`(0.97994065, 0.97994065, 0.97994065)`. The zero-extinction control produced
the same values. Optical Metal values matched CPU and analytic Fresnel ×
Beer-Lambert RGB within each stated tolerance:

| Case | CPU RGB | Metal RGB | Tolerance |
| --- | --- | --- | ---: |
| Legacy no-optics | `(0.9799407, 0.9799407, 0.9799407)` | `(0.97994065, 0.97994065, 0.97994065)` | `0.0002` |
| Zero extinction | `(0.9799407, 0.9799407, 0.9799407)` | `(0.97994065, 0.97994065, 0.97994065)` | `0.0002` |
| Underwater exit | `(0.60637105, 0.39841416, 0.16198312)` | `(0.60637105, 0.39841416, 0.16198312)` | `0.0003` |
| Air entry | `(0.77084905, 0.62483835, 0.39841488)` | `(0.770849, 0.6248383, 0.39841485)` | `0.0003` |
| TIR | `(0.5488119, 0.30119452, 0.16529915)` | `(0.5488119, 0.30119452, 0.16529916)` | `0.0008` |
| Nested water and glass | `(0.5558653, 0.5670945, 0.4372593)` | `(0.5558652, 0.5670945, 0.4372593)` | `0.001` |

Every device case reported cover `1` and depth `2`. The test passed in `0.19s`
(GPU-lane package time `0.419s`). The complete output, including CPU and
analytic values for every subtest, is preserved at
`metal-clear-medium-expanded-device.log` with SHA-256
`3c9b70f509eab17a19d71040655ca9e27a53db5889cec03d6f195cae501834c9`.

These single-pixel cases prove only the tested flat paths. The device run
predates the merge of water commit `79102c370c3c5240930c052b2855c9f19ea7cbeb`;
no Metal rerun was made after that merge. Moving cameras, direct-light paths,
scattering, volumes, fields, broad scenes and performance remain unvalidated.
The 64-byte per-thread figure above is source-level storage only; device
occupancy and spills were not measured.
