# Miami as a reference

Miami has scenery and display type worth learning from. Use its looks and
capabilities as templates, and implement them within Slayer's existing
geometry, typography, HUD and frame pipeline.

This is a design note for the open Scene, TYPE and HUD backlog items. It does
not describe implemented Slayer features.

The source inspected is Miami at
`2666f6880ce79eb12f1ea9f79946003cccc85405`:

[letter placement][letters], [outline fonts][fonts], [figlet][figlet],
[overlays][overlays], [log text][logs], [worlds][worlds] and [looks][looks].
No Miami code has been copied into Slayer.

## Keep the character

The scenery can be beautiful and stupid: terran, arid, frozen and molten
worlds; ember, azure and quasar palettes; coloured gas, dust and particles.
Describe these as authored looks. A name such as quasar does not make a
palette a calibrated observation.

Miami's type has kerning, tracking, condensed letters, curved or sloped
baselines, weight adjustments and seeded hand-set variation. Its overlays
have fitted figlet text, blinking clock colons, progress bars and scrims.
These are useful capabilities to expose in checked YAML recipes.

## Build on what Slayer has

`typography.Layout` already measures advances and kerning with explicit
errors. `typography.Make` makes solid letters and shares repeated glyph
meshes. Extend placement and style around those owners; retain mesh sharing
when individual letters move, tilt or change size.

Measure and draw from the same placement result. State whether centring uses
advance width or visible bounds. Include tracking in that choice, and make
seeded variation stable across repeated frames and out-of-order samples.

Keep font selection explicit. Missing glyphs and unreadable font files need
useful errors. The recipe should name any fallback policy. A failed font
load must not quietly erase a caption.

World-space type participates in scene lighting, occlusion and GPU tracing.
A screen-space caption belongs to `hud` after scene development. Give each
route its own example so the distinction can be seen.

## Compose the HUD

Slayer already has frame-time `Wording` and `Gauge` interfaces, themed roles,
world labels, occlusion and a shared cell-coverage contract. Add display
text, clocks and richer styling through those pieces.

Take one immutable value snapshot for each frame. Keep layout and drawing
separate from the caller updating a value. Use the sequence clock for
scripted movies; an explicitly requested wall clock is a different source.

Fit type using actual measured text and the available rectangle. Keep caches
bounded and keyed by the font, text, size and style they represent. A changed
value or resized output must invalidate the relevant result.

Preserve uncovered pixels and cells. Test clipping, alpha, theme errors and
nonzero rectangle origins in both terminal and PNG output. Inspect the
pictures at their intended viewing size, including small terminal faces.

## Delivery

YAML and CLI inputs go through Puzzlemouth and checked schemas. Keep authored
palettes and variation limits in data with their provenance. Use shuriken and
Game lint, sweep, check and targeted mutations; record the exact scope and
existing baseline failures.

A scenery or world-text recipe also needs an actual GPU render. Keep PNGs,
settings and timing with the review evidence. A passing layout test alone
does not settle the appearance.

[letters]: https://github.com/janearc/miami/blob/2666f68/typeset/line.go
[fonts]: https://github.com/janearc/miami/blob/2666f68/typeset/outline.go
[figlet]: https://github.com/janearc/miami/blob/2666f68/typeset/flf.go

[overlays]: https://github.com/janearc/miami/blob/2666f68/app/overlay.go
[logs]: https://github.com/janearc/miami/blob/2666f68/app/log.go

[worlds]: https://github.com/janearc/miami/blob/2666f68/scenery/worlds.go
[looks]: https://github.com/janearc/miami/blob/2666f68/scenery/looks.go
