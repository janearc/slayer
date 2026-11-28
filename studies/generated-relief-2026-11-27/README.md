# Planet relief diagnostic

These five frames compare the experimental candidate ef0e1cee8, not a
released feature. They hold seed 1, quarter camera, exposure 0.003,
800 × 800, 2 rays per axis and clear clouds fixed. The authored height
scale is 0, 0.001, 10, 40 or 200 km per dimensionless terrain-height
unit. The first is a sphere; nonzero uses a fixed 128 × 64 global mesh.
The 200 km choice is deliberately exaggerated and made up.

The 1 m mesh is visually close to the sphere in this view. At 10 km a
dark square grid appears across central land; 40 km darkens the land with
broad shading; 200 km introduces angular wedges. The limb changes only
slightly and neither reviewer could confidently identify a cast shadow or
recognizable mountains. The 10 km grid blocks presentation. These outputs
show an effect, not accepted relief, and do not establish the grid's cause.
The five saved jobs write PNGs and linear PFM raws under
`out/studies/2026-11-27-relief-NAME/`.

The `10km-dense` control used candidate 773fad041: the same height field
and filter, scene, light and camera, but a 256 × 128 mesh. The previously
conspicuous square grid is no longer readily visible at this image size.
This comparison shows a dependence on mesh resolution; it does not prove
the method suitable at ground level or establish a complete cause. The
candidate remains experimental.

The first `grazing-*` pair used a closer camera but showed only a dark
atmospheric limb; it cannot assess ground or relief. The `lit-*` pair keeps
the eye at [-6297000, 2046000, 0] m and aims at [-5800000, 200000, 0].
Against the generated 6,421,960 m sphere, the centre ray intersects
roughly 379 km from the eye, and the hit's normal has a dot product of
0.0843 with the drop's direction (about 4.8 degrees above the local
horizon). Both views are nearly uniform brown; no land or mountain form is
discernible. The geometric calculation does not establish what hides it.

The `noair-*` pair reuses those exact lit sets with diagnostic binary
02c900394, which omits the generated air volume after deriving the planet.
The underlying green/brown ground becomes visible, establishing that the
air volume materially masked it in this setup. The sphere and 10 km mesh
still look almost identical at this view; no mountains or cast shadows can
be confidently identified. Removing air was a measurement only, not a
proposed fix or a reason to remove atmosphere from the generated worlds.
