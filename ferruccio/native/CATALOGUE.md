# Catalogue settings

A catalogue picture has two inputs: a specimen name and its rendering plan.
The saved `settings.yaml` contains camera, lighting and sampling controls; it
is not a Slayer job and does not contain the specimen name. The companion
`render.json` records that name under `specimen.entry.kind`.

Atelier checks the settings against the catalogue plan schema and selects the
recorded specimen when the receipt is present. A standalone settings file asks
for a selection in the Specimen menu. Saving under another name preserves the
selection in the current window; reopening a copy without its receipt requires
selecting the specimen again.

Render preview invokes the bundled catalogue producer through Game5. It uses a
new snapshot with 640 × 360 pixels, one ray per side and the first detail listed
in the plan. Lighting, camera and all other settings stay as authored. The
original YAML and previous pictures remain intact. The new output directory
contains the catalogue book, specimen image and producer receipt.

The plan schema and catalogue producer are pinned to Slayer commit
0b0d207e0290098e49392570291eb2567567e0c6. The specimen menu is unchanged from
a89af6be1f1079ed540e532c7d5650260917a928. Unknown fields and unknown specimen
names are refused. The catalogue producer also checks camera and allocation
constraints before drawing. Atelier does not convert these files into scenes.

Long diagnostics are presented in bounded pages with a Copy action for the
complete text. The inline diagnostic is a short preview; Check opens the full
validation details, and Render log opens the renderer output.

The producer is Game-stamped from the clean source commit above. That commit
uses a local Celestial source replacement without its own versioned checkout.
The packaged executable is frozen by SHA-256; this is not a claim that all
transitive source dependencies are independently pinned. Build information is
retained with the validation records.

Catalogue previews support camera navigation.

Drag to orbit around the specimen,
or scroll to change field of view. The changes appear in the draft as `orbit`
and `wide`; Save retains them.

The YAML editor's undo history includes camera
edits. A new preview starts after input settles. The prior image remains visible
and labelled while the requested view is computed. This is a still-preview
workflow, not continuous 3D drawing. Mission references and opened image files
do not have camera controls.

The orbit producer's SHA-256 is
cee0a127dd7694d3499f08fedcd5c21237d1f8be954cec4d9ec48eda00ff8834.
