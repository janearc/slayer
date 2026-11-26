# Chroma colourway sources

These CSS files are copied byte-for-byte from `janearc/chroma` commit
`284aab1761ca9202052caee4e3b65e4e9c0a456f`, under
`colourways/sources/`. The Atelier appearance loader reads their named colour
roles with the `css` package of `github.com/janearc/libtheme-css`, at the
version `native/go.mod` and `native/go.sum` pin.

The default `vaporwave` colours, and every selectable theme, come from these
source files. `LICENSE.chroma` is the license from that source commit.

SHA-256 of each embedded CSS source:

| Theme | SHA-256 |
| --- | --- |
| aqua | `72e85072677b00df6b8e195bad630864d74e87b1ec793cad7b3c9cdbeea457b1` |
| fermi-long | `7ac313ed7269f07227a8bd082f4c7d2c319ff2420bc69e5deb17130b0b613a3b` |
| fermi-midnight | `7bfff458e0f19a12ee78b87511d168185ab33996285b213c11fbc7b4f004833d` |
| neptune-voyager | `b240b8a88d28cbf534e3d9bdc8aee7b22c2d9511f8ca7bdc9213d51b3f1a0a34` |
| pastel-invert | `b45eee51021f41e837e113f4cbdfec45df0e47214b3fd8c13052046f1c10b2a5` |
| pastel-rose | `cc800485e73663c912d9b4bb5244a76f750f5eb65c1489dc69f7eee95966059c` |
| pastel | `4f6f08bbd74897b391c4f94fd4dbc980499ab01385d1ab0dd1615fcb3a0db29b` |
| pluto-coffee | `35e03f9be7057a2ea995af7657bcfe24bcd30306842860cf56f8737833789bfc` |
| pluto-night | `e3feee7f7014b2cfdfccb3f2b755fedc48f17cc6a1d97637704c5e33829d7617` |
| pluto | `583a6d8c68d5cdab01c99659af2405a9b15a50e5d1c177272ce33fba9424bbf2` |
| twilight-burnt | `311f2d0734196db4a541be973aeaecd619c2ea7f2913ee341b4b4b6145a89380` |
| twilight-deep | `c259d1c598c844023719ab4f96b33de20b6721b77b97198edeff499d9887df7b` |
| twilight-low | `2be32bcd30d657404dc9cc345517d50fa201eed033c9903da24035d8975c201b` |
| twilight | `c5aac56355cfab0f425131ed3a7d0c753a162b930c2f4e8dbe271b00b0cdadc6` |
| vaporwave-dusk | `d984b4094a5fea053c3626348a082ce38649de429afdcda55b1d32ea9db889b3` |
| vaporwave | `1dbf52c864aca50a9ecdd91df00765d26fb6ca378511fc8d904a27bcb8c76d86` |

## Role mapping and readability

The appearance code reads CSS roles with Chroma's pinned `libtheme/css`
reader and fills absent general roles with `libtheme/colourway.Resolve`.
Resolved roles retain Chroma's `From` lineage. The source CSS is never edited.

Atelier keeps the owner's 10–12.5 contrast band and 40-degree hue gap rule
for text on saturated grounds. It checks only exact swatches from the source
or Chroma's documented role resolver. `AvailableThemes()` lists themes that
meet both gates; `UnavailableThemes()` reports the source themes that do not,
and `SelectTheme` rejects them without changing the active theme.

For Vaporwave, source `--ink` has 11.00:1 contrast on `--ground` but only a
10.6-degree HSL hue gap. The readable source role selected for body text is
`--blue` (`#94c3ff`): 10.22:1 contrast and a 44.5-degree gap. `InkSource()`
reports that role and `InkAdjusted()` reports that the semantic ink role was
aliased; the color value itself is unchanged. `--heading` and `--link` remain
the action and link colors.

The usable themes are `aqua`, `fermi-long`, `fermi-midnight`,
`neptune-voyager`, `pluto-coffee`, `pluto-night`, and `vaporwave`.

`pastel`, `pastel-invert`, `pastel-rose`, `pluto`, `twilight`,
`twilight-burnt`, `twilight-deep`, `twilight-low`, and `vaporwave-dusk` are
unavailable because no exact source or resolved Chroma role meets both text
gates on the source ground. Those names remain in the provenance table, and
the runtime API returns their precise refusal reason.

Text-bearing DevTools surfaces use the selected ground token. The exact
resolved `panel` and `surface-1` roles are exported as `DecorativePanel` and
`DecorativeRaised` for non-text framing. This keeps their source colors
available without putting prose on surfaces that fail the contrast floor.

The CSS is copied, not built from Chroma, so the build does not need Chroma
as a module. `LICENSE.chroma` is copied from the same source commit.
