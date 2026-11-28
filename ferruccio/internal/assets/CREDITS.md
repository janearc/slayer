# Prototype assets

The two landscape images are reference material. Neither is an image rendered
by Atelier or Slayer. Their presence does not imply that terrain data or an
atmosphere has been loaded into a set.

- Venus: NASA/JPL, Magellan, PIA00106. A computer-generated perspective of
  Maat Mons using radar and altimetry. Simulated colour; vertical scale 10x.
  Source: https://science.nasa.gov/photojournal/venus-3-d-perspective-view-of-maat-mons/
- Titan initial iteration: ESA/NASA/JPL/University of Arizona, Huygens,
  PIA06440. A merged view from the probe's surface images.
  Source: https://science.nasa.gov/photojournal/view-from-titans-surface/
- Titan current view: NASA/JPL/ESA/University of Arizona, Huygens, PIA07232.
  Processed colour surface view, incorporating reflection spectra data.
  Source: https://science.nasa.gov/photojournal/first-color-view-of-titans-surface/
- Display typeface: Cormorant Garamond Regular, Catharsis Fonts. The SIL Open
  Font License is bundled in OFL-Cormorant.txt.
  Source: https://github.com/CatharsisFonts/Cormorant
- Native UI typeface: GoGPU UI embeds Inter Regular and Bold. Their
  font metadata identifies Inter 4.001 at source revision 9221beed3.
  The app bundles its SIL Open Font License in OFL-Inter.txt.
  Source: https://github.com/rsms/inter/blob/9221beed3e5b521ce94050ca542f568755c17c22/LICENSE.txt
- The legacy Fyne frontend supplies its own fonts.
- Job schema: copied from this Slayer checkout, `job/job.schema.json`.

The prototype stores these files locally and makes no image request at runtime.

The catalogue plan, menu schema and menu are copied from this Slayer
checkout's `cmd/catalogue/plan.schema.json`, `menu.schema.json` and
`menu.yaml`.

The ecosystem contracts are copied from this checkout's
`set/ecosystem.schema.json` and `ecosystem-v2.schema.json`, including the
native `human-scale-2m` kind. `producer_contract_test.go` compares all six
bundled contracts against their producer files; `SHA256SUMS` records the
bundled bytes.

The active Ferruccio package takes its renderer executables from this
checkout's root `bin/slayer` and `bin/catalogue`. It does not use the old
private renderer checkpoints. Game records their actual build stamps and
hashes inside the bundle. During development a dirty stamp describes a
working tree; it is not release provenance. See
[the build contract](../../../doc/applications/ferruccio/BUILD.md).

The Subaru settings fixture is the existing authored night study from
the shared renders collection, with its camera and local lights preserved.
The legal mannequin fixture in `examples/oakland-larson` comes from Oakland
core c76ca61a, licensed CC0-1.0 alongside its source files. Its two-metre
normalization is a modelling choice. The native scale figure used by the
new playful scene is Slayer's `human-scale-2m`, a separate procedural kind.

Native construction disclosure copies producer data byte for byte:

- native-larson.yaml: manufacture/saucer-choices.yaml, authored saucer proportions.
- native-larson-landed.yaml: manufacture/saucer-landed.yaml, authored landing gear.
- native-human.yaml: manufacture/human-scale.yaml, authored scale figure.

These are the current producer recipe data, not measured engineering claims.
The producer-contract test compares each copy with its source. SHA256SUMS
records the bundled bytes. The recipe files do not include all construction
algorithms, materials or runtime state; native manufacture remains authoritative.
