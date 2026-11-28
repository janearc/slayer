# Flowing giant moon-shadow observation

Azazel jobs 368–370 used CPU candidate `5f147e85d`; jobs 371–373 used GPU
candidate `53ce509a7`. Each backend rendered the same three committed set
files at 640 × 640 pixels, two rays per axis and exposure 25000. The giant,
its weather checkpoint, the disc camera and sun (azimuth 225°, altitude 20°)
were fixed. Only the opaque Mercury-template moon's position changed. Its
12,000 km radius is an authored diagnostic choice, not a named real moon.
Set positions in east, north, up metres were control [180000000, 0, 0],
sunward [-79730000, -79730000, 41040000], and foreground [-103366000, 0,
37620000]. The model uses a single directional sun and binary hard shadows.

The jobs now write pictures and linear PFM files under
`out/2026-11-27-flowing-shadow-{control,eclipse,foreground}/` for
CPU and the corresponding `-gpu` directories for GPU. Compared with the
control, an independent visual researcher found a sharply bounded black oval
on the giant and a separate white moon in the sunward image, on both
backends. The foreground image showed the moon occluding the giant near the
centre and a separate black oval extending from the left limb, again on both.
This visually supports distinct cast-shadow and foreground-occlusion behavior
for these authored placements. It does not establish spectral or volumetric
shadow transport, extended-source penumbrae, physical accuracy, or general
backend equivalence.

The moon clips to featureless white on both backends at this exposure. CPU
images have heavy speckle while GPU images are smoother, and the hard black
ovals look severe. These pictures are diagnostic rather than investor-ready.
