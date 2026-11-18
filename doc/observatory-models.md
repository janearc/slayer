# Ground observatory catalogue models

The named ground observatories share checked optical and enclosure
construction, then keep their recognizable structural families. Keck I and
Keck II are separate catalogue specimens with the same family recipe. Their
individual scene poses can move together without collapsing the catalogue
entries into one object.

## Physical references and authored geometry

The visual cues come from primary observatory references:

- [Keck telescopes](https://keckobservatory.org/our-story/telescopes/):
  domed enclosures, open telescope structure and segmented primary mirrors.
- [Subaru structure][subaru-structure] and
  [Subaru enclosure airflow][subaru-airflow]:
  the cylindrical enclosure and instrument arrangement.
- [Rubin summit facility][rubin-summit]:
  the enclosure attached to the low, long service facility.
- [ELT dome](https://elt.eso.org/telescope/dome/) and
  [ELT structure](https://elt.eso.org/telescope/structure/):
  the enclosure mechanisms, segmented primary and layered instrument support.

The model retains source-backed apertures and panel inventories from
`research/` and the source shelf. Enclosure envelopes, ribs, galleries,
shutter tracks, access decks and mirror-cell beams are authored scenery.

Their ratios and provenance are recorded in
`manufacture/ground-choices.yaml`; they are not engineering drawings or
measurements from photographs. Rubin retains its shared M1/M3 blank and
camera roles. Its primary is not segmented.

## Structural detail

The domed shell interiors gain segmented meridian ribs and tracks alongside
the observing slit. Subaru's corresponding supports are vertical; nothing
bridges its slit. Access decks have posts with the same authored 1.1-metre
rise for each instrument, independent of the primary's diameter.

Rubin keeps its faceted enclosure and service wing, with roof members and
posts placed within each tier. Mirror cells gain concentric supports and
radial members behind the primary.

These are ordinary manufactured meshes with the existing material roles
and CPU/GPU paths. Detail levels retain the same aperture and physical
envelope. The closed members leave small authored joint clearances to avoid
coincident edges when material surfaces are grouped into a mesh.

Exterior panel divisions, ventilation equipment, working shutters, detailed
mount mechanisms and articulated scene groups are not modelled.

[subaru-structure]: https://prc.nao.ac.jp/3d/subaru_structure_e.html
[subaru-airflow]:
  https://subarutelescope.org/en/news/topics/2012/04/18/2537.html
[rubin-summit]:
  https://rubinobservatory.org/explore/how-rubin-works/technology/summit-facility

## Subaru exterior study

The Subaru recipe models a narrower tall drum above a wider equipment
belt, a projecting walkway, a pale fixed base and an attached service
tower as separate forms, with raised seams, shutter-edge housings,
ventilation panels and guard rails.

[Subaru's enclosure description](https://subarutelescope.org/en/about/features/8.html)
and its [airflow note](https://subarutelescope.org/en/news/topics/2012/04/18/2537.html)
describe the cylindrical enclosure, shutters and ventilators. These are
qualitative references. The existing pinned overall dimensions and primary
aperture remain unchanged; the new tier proportions, counts and finishes
are authored interpretations in ground-choices.yaml.

This pass has static open shutters. It does not establish mechanical
tracking, a measured panel layout or a thermally accurate enclosure.

## Keck enclosure

The Keck enclosure uses a curved body down to a low plinth, parallel opening
edges, substantial shutter housings and a broad lower shutter panel.
The optical pivot is raised within the declared physical envelope.

The [Keck II permit cross-section](https://dlnr.hawaii.gov/occl/files/2018/01/CDUP-2509-Keck-2.pdf)
and [Maunakea enclosure workshop](https://www.eaobservatory.org/jcmt/wp-content/uploads/sites/2/2016/10/2016-12-8_Maunakea-Operations-and-Engineering-Workshop-Domes-and-Shutters_intro.pdf)
support the distinction between Keck's up-and-over shutters and Subaru's
lateral shutters. The proportions are authored; they are not measurements
taken from those references.

The warm catalogue study uses schema-checked local-lights in specimen
coordinates, with point-source RGB strength at one metre. These scene
lamps illuminate the interior and reflective mirrors. They do not alter
the mirror's emission or pretend to represent observing operations.
The connected facility and actual shutter motion are not modelled.

## Keck instrument and clearance

Keck has its own pale angular space frame, diagonal members, spider
supports and a secondary cage with depth. The primary stays a reflective
36-segment surface. Member counts and proportions are authored scenery,
not a reconstruction of engineering drawings.

The [Keck control-system poster][keck-controls] identifies the upper tube,
spider, secondary and mirror cell. [Berkeley Lab's account][keck-lbl]
describes a roughly 36-metre-wide, 30-metre-high dome and a secondary
about 15 metres from the primary. These rounded descriptions expose the
current authored 30-by-28-metre enclosure's scale mismatch; they do not
establish exact shell thickness, pivot height or shutter clearance.

The default placement and narrower top frame keep the optics inside the
curved interior at every detail level. A measured enclosure and full
swept-volume clearance across an articulated pointing range are not
established.

[keck-controls]: https://proceedings.jacow.org/ICALEPCS2015/posters/mopgf040_poster.pdf
[keck-lbl]: https://www2.lbl.gov/Science-Articles/Archive/keck-telescope.html

## Night studies

Keep a night view beside the daylight and cutaway studies for models with
attached lamps or authored interior lighting. Preserve the lamp settings
in the saved recipe, and distinguish model-attached sources from lights
placed for the shot. Check that the lights illuminate nearby surfaces;
a glowing mirror is not a substitute for interior illumination.

## Interior studies

Catalogue plans may supply an `interior` block with `at` and `look-at`
in model metres, X east, Y up, Z south. This overrides the catalogue's
bounds-relative framing. The enclosure is retained. The block also declares
a procedural star field, bounded participating air, and optional beam
sources. Every pose and display strength is authored in checked YAML.
