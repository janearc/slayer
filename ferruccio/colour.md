# Color and type

Atelier uses the Chroma Vaporwave colourway. The source is
`colourways/sources/vaporwave.css`; its Git blob is
`4742e0f1813aa8723ff1be2c267e3b07121d43ec`.

The theme keeps the source's deep violet hue, muted lavender ink, bright
magenta and cyan. Its prose-bearing violet ground and layers are derived
desaturated shades (HSL saturation below 0.35) so lavender prose retains a
clear hue edge while meeting the contrast floor. Source panel and surface-1
values are too bright for the prose contrast band.

| Atelier role | Value | Source |
| --- | --- | --- |
| Window ground | `#14111a` | Desaturated shade of `--ground` |
| Prose | `#cdc3d8` | `--ink` |
| Primary-action fill and selected text | `#ffb5ff` | `--bright-magenta` |
| Links | `#00dff9` | `--link` |
| Panel and focus fill | `#1b1623` | Desaturated shade of `--cursor-line` |
| Hover and selection | `#1f192a` | Desaturated shade of `--surface-1` |
| Control edge | `#432c6b` | `--border` |

Prose colors and primary-action labels are tested at 10:1 to 12.5:1. Their
hue gaps are also tested against every prose-bearing surface. The action
label uses the source ground over a bright magenta fill. Buttons use the
panel layer; the UI renderer adds a visible border so their shape reads as a
control without relying on color alone.

`Theme()` defaults to 18-point text. `ThemeWithSize` supports 16, 18, 20 and
24-point body text, with headings scaled above it and captions kept at 16 or
larger.
