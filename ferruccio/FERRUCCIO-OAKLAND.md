# Ferruccio and Oakland

Ferruccio is the separate GoGPU/Metal editor application. Slayer remains
the renderer; Grissom remains a separate viewer. The Fyne prototype is
retained as an earlier checkpoint. This slice provides still-image editing,
camera navigation and a first static imported-geometry study. Film planning,
music and USD remain later work.

## Open the study

Choose **Larson + visitor**, then **Render preview**. The app copies the
bundled scene and its CC0 mannequin resources into an independent working
folder. Save chooses a durable destination and carries those resources
along. An existing asset with different contents causes a refusal before
the YAML is saved; the app does not replace it.

Observe shows the YAML beside the completed image. Compose gives the image
more room and provides camera buttons. Drag to orbit the camera around its
target; scroll to change its field of view. The figure's placement is the
`at` and `heading` under `scene.world.imports.visitor`. The Larson's is under
`scene.set.things.larson`. Editing either and rendering changes the actual
Slayer scene. Object selection and transform handles are not implemented.

Camera changes are undoable YAML edits. Command-Z and Command-Shift-Z while
the image has focus restore the camera draft and request its next render.
The viewport retains the previous completed image while rendering. It never
rotates that bitmap to pretend a new view has been rendered.

The theme and text-size controls are in the footer. Colours come from pinned
Chroma CSS, parsed by Chroma's own packages. Vaporwave is the default. Themes
are offered only when their exact source swatches meet the established
contrast and hue requirements; refused themes and role choices are recorded
in [theme provenance](native/internal/appearance/themes/PROVENANCE.md).

## Data flow

`native/internal/workspace` owns the UI thread and document selection.

`internal/draft` owns schema-checked YAML edits. The editor keeps its undo
history; the original file changes only on Save. Catalogue plans use their
own schema and producer. Fixed set cameras use explicit east/north/up
`location.at` and `orientation.look_at` coordinates. Indirect job files,
relative cameras, moving cameras and lens rigs remain outside the gesture
controls. They can still be edited as YAML.

`internal/preview` freezes the draft beside its original file, preserving
relative paths, and starts the appropriate Game-managed renderer process.

Each navigation event replaces the pending request. After 300 milliseconds
without another gesture, the workspace cancels its previous render, waits
for that process to finish, then draws the newest view. Path, text, document,
specimen and generation checks reject stale results. Stop cancels only the
window's own work.

In Slayer, `set` validates the import declaration and loads Oakland's
checked asset. Oakland translates the supported static glTF/GLB subset into
existing `mesh.Mesh` geometry. `stage` binds explicit native materials and
adds ordinary `thing.Thing` instances. The CPU tracer, Metal renderer and
manufactory do not import Oakland. Repeated geometry is shared; unsupported
features and excessive input or expansion are refused.

An import-bearing `job` holds the checked set and exact decoded source
hashes together. Rendering consumes that snapshot. Command-line overrides
keep it; a new job rereads the files and gets a changed render identity when
the geometry, normalization or material definitions change. The ordinary
light, picture and resume caches therefore see imported dependencies.

The renderer writes an `.imports.json` receipt beside each still. It names
source files and digests, rights declarations, native material definitions,
normalization choices, placement, job identity and build. **Render log**
shows the import and receipt path. A two-metre mannequin is a geometric
reference, not a claim about scanned anatomy or measured optical material.

## Scope of this build

Static triangles, checked node transforms, shared instances and explicit
native materials form the first Oakland slice. Textures, animation, skins,
morphs, glTF extensions, USD composition and export are refused or remain
unimplemented. The supported input contract is documented beside the
Oakland source. This is a local review build with recorded tests and actual
render proofs, not a release or independent review verdict.
