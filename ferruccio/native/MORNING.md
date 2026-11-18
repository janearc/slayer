# Start here

Open `bin/Ferruccio.app` in the parent directory. This is the current native
GoGPU/Metal application. Earlier Atelier bundles are kept as checkpoints.

Choose **Larson + visitor**, then **Render preview**. The scene places an
imported two-metre figure beside Slayer's manufactured Larson. Drag the image
to orbit the camera; scroll to change its field of view. Command-Z restores
the previous camera draft and queues its render. Observe keeps the source
beside the picture; Compose gives the picture more room and camera buttons.

Use the footer's theme menu to change Chroma themes. The YAML editor and
surrounding controls use the same theme. Text size is adjustable. Edit the
figure's `at` under `scene.world.imports.visitor` to change its placement;
object selection and transform handles remain future work.

Save the study to a folder of your choice. The app carries its glTF, binary
buffer and licence along, and refuses an existing asset with different
contents. The original example stays intact. **Render log** gives the new
image and import-receipt paths. Stop cancels the window's current preview.

The shared renders gallery contains every captured iteration. Captures 12–14
exercise orbit, zoom and undo; 15–16 show the imported study and its new camera
view. The Oakland subgallery has the matched CPU/Metal images and receipts.
Window images are exports of the real widget tree, not macOS screenshots.

Read [the architecture](../FERRUCCIO-OAKLAND.md),
[build provenance](BUILD-PROVENANCE.md) and [validation](VALIDATION.md).
No independent review was requested before this reading.

The catalogue-settings fix adds another check: open the Subaru night study
`settings.yaml` beside its `render.json`. The document should say `catalogue`,
with `subaru` selected above the editor. Render preview uses the matching
catalogue producer and displays a fresh image. A copied settings file without
its receipt asks you to select a specimen. See CATALOGUE.md for the format.
