# More than one sort of summit telescope

The Hawaiian families have distinct optical structures, mount silhouettes
and enclosures. They are scenery recipes. The apertures have source roles;
the supporting dimensions and parked poses are authored.

The [reference notes](summit-references.md) name the pinned institution
pages, technical papers and historical inventory. Source quantities live
in research/research-derived-data.yml, with exact excerpts and PDF pages.

Structural choices live in manufacture/summit-instruments.yaml and the
shared picked-number book.

## Maunakea

The optical families are cfht, gemini-north, irtf, uh-88 and ukirt.
They include yokes, open trusses, a closed tube, service annexes and a tall
vented dome. Gemini's enclosure height also comes from its source handout.

The radio families are jcmt, sma-antenna, sma-array and vlba-maunakea.
They use backed paraboloid shells, quadrupods and compact altitude cradles.
JCMT adds its broadly opened enclosure; the VLBA station adds a control
building. These do not simulate a radio receiver or interferometric signal.

The SMA array retains eight instances of each antenna's material prototype.
Its 6 m primary and f/0.4 ratio are source-backed. The compact array layout,
feed, mount and cabin are authored, not surveyed pad positions.

UKIRT remains available as a building family. UH announced the end of
science operations for September 15, 2026. Its presence in a scene must not
be read as a claim that it continues observing.

CSO and Hoku Ke'a are not silently placed in a present-day layout.
Their structures were removed in 2024; explicit historical recipes can be
added separately.

## Haleakala

- panstarrs-1 and panstarrs-2: compact reflectors and small domes;
- faulkes-north: an open reflector with a broadly opened clamshell proxy;
- dkist: an offset secondary, clear pupil and round-hole calotte proxy;
- atlas-haleakala: a short wide-field camera and counterweighted mount;
- tlrs-4: a small ranging telescope with an opaque corrector proxy;
- solarc: an off-axis instrument and sliding-roof housing;
- mees-solar: the facility family with a picked small solar-tube proxy;
- aeos: an open tracker with its enclosure panels parked low;
- mssc-1-6m: a closed reflector with an outer gimbal;
- mssc-twin-1-2m: two parallel tubes on a common central support;
- geodss and geodss-site: one tracker or three shared dome prototypes.

The Mees tube's 0.25 m aperture is explicitly picked. The references
establish the facility and its solar work; they do not establish that
particular instrument size. It is not the Mees observatory in New York.

DKIST's 4 m value describes its clear pupil, not its larger physical blank.
Its shallow display mirror does not reproduce the off-axis parent figure.
TLRS-4's ranging telescope is not a sodium guide-star launcher.

The GEODSS value is a nominal metre-class family. The historical inventory
also gives a 102 cm physical main aperture and an auxiliary instrument.
The three-dome layout here is authored; it is not a current site survey.

## Put them in a scene

Use a name as a manufactured thing's kind. The scene supplies placement,
terrain, lights and weather. Fixed default optical poses make comparisons
between surface details useful; tracking and mount motion are separate.

For a local specimen book:

    game run catalogue -- render cfht gemini-north jcmt sma-array dkist \
      --out out/hawaiian-observatories

The four details change circular tessellation. Each retains its declared
physical site envelope, aperture and authored array layout. Shared arrays
use the existing mesh-group CPU and GPU paths, without expanded geometry.

Optical objectives are opaque proxies. Mounts suggest their construction
families; they do not implement tracking kinematics. Except for the SMA
primary ratio, radio figures use an explicitly picked focal ratio.

These families supply the buildings and instruments. The separate
[yard equipment](yard-equipment.md) supplies cranes, vehicles, scaffold,
utilities and access structures for a working site.
