# Staged working view

`Session.WorkingViewShared` inspects a staged frame without drawing it.
It returns the frame's camera, dimensions, shared surface meshes,
placements and authored origins. Repeat placements can share object-local
geometry. `Session.WorkingView` returns an aggregate mesh for the
same inspected scene.

A physical rig uses a labelled pinhole approximation at its focused
angle of view. Blur and shutter exposure belong to the offline render.
The view reports volumes, fields and lights outside its surface
representation. The shared view can show the sea as a labelled mean-level
proxy.

A finite height field is triangulated at the original sample locations.
The triangles approximate the bilinear surface between samples. The
builder reports that approximation and rejects fields above 262,144
cells.

Staged geometry can carry an authored reference. A nearer unannotated
surface still occludes a reference behind it. The caller checks that the
source placement supports the requested edit. Native hit targets can
refer to staged surfaces, so keep the owning session open while using
them. Ferruccio's [working-view guide](../../doc/applications/ferruccio/WORKING-VIEW.md)
describes its detached display and editing behavior.
