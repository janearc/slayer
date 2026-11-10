# A working summit has a yard

The next hero scene waits for the instruments and the things around them:
service buildings, mirror-handling equipment, vehicles and access structures.
Golden hour was requested, so the next composition uses low sunlight and
colour across the terrain, painted walls, metal and vegetation.

This expands the manufactory, not just one decorated render. Each kind needs
YAML placement, a catalogue entry, closed ordinary GPU meshes, four useful
detail levels and a stable physical envelope.

Dimensions picked for scenery stay labelled as picked. Instrument apertures
need pinned primary sources; the three personal telescope apertures are
requested design choices.

## Guide-star pointing

The old hero scene deliberately gave its four beams different azimuths and
several degrees of separation. Replace those display choices with a named
common pointing and an optional tangent-plane offset in arcseconds.

[ESO's GALACSI layout][galacsi] describes its wide-field guide-star radius as
64 arcseconds. That is an example pattern, not a universal observatory rule.
The [MUSE manual](https://www.eso.org/sci/facilities/paranal/instruments/muse/doc/ESO-261650_MUSE_User_Manual_P113.pdf) also shows the launchers on the telescope structure.
The metre-scale near field should consequently look almost parallel.

Preserve perspective: parallel world-space axes can meet at a vanishing point
in an image. Keep launchers, telescope pointing and the shared sky direction
consistent. Display beam radius and brightness remain explicit choices.

Finite-range sodium-beacon parallax, focus, refraction and sodium return
must not be claimed from an outgoing Gaussian beam alone.

[galacsi]: https://cdn3.eso.org/sci/facilities/develop/ao/sys/galacsi.html

## Maunakea

Keck I, Keck II and Subaru already have templates. The remaining named
families are CFHT, Gemini North, IRTF, UH 2.2-m, UKIRT, JCMT, the SMA and
the Maunakea VLBA antenna. Distinguish a telescope, its enclosure, its
support building and an array of separately placed antennas.

The [IfA facility index][ifa] and [observatory directory][mk] establish the
site inventory; each instrument still needs its own dimensional reference.
The older summit table is historical evidence, not a current site manifest.

Historical CSO and Hoku Ke'a recipes can remain useful scenery, explicitly
labelled as historical. UH reports both structures [removed in 2024][removed].
Do not silently place them in a present-day summit recipe. Proposed
instruments are a separate catalogue category.

[ifa]: https://www.ifa.hawaii.edu/
[mk]: https://www.maunakeaobservatories.org/about/
[removed]:
https://www.hawaii.edu/news/2024/07/02/second-observatory-removed-maunakea/

## Haleakala

Use the [IfA facility inventory][haleakala] as the starting list:
Pan-STARRS 1 and 2, Faulkes Telescope North, DKIST, ATLAS, TLRS-4,
the Zodiacal Light Observatory / SOLARC and the Maui Space Surveillance
Complex.

Include the Mees solar-observatory family with its historical context
checked. MSSC needs distinct public instrument families rather than one
dome bearing every instrument's name.

Do not infer an engineering layout or an unlisted instrument from a photo.
The inventory and status of the smaller instruments need source review
before this list can be marked complete.

[haleakala]: https://about.ifa.hawaii.edu/facility/haleakala-observatories/

## Three personal telescopes

- A 36-inch Dobsonian: open truss, mirror box, upper cage, altitude bearings
  and rocker base.
- A 5-inch apochromatic refractor: long tube, objective cell, rings, focuser,
  diagonal and a usable mount and tripod.
- A 24-inch Schmidt-Cassegrain: short tube, corrector/secondary assembly,
  focuser, mounting and pier.

These are instrument scenery recipes. The words apochromat and
Schmidt-Cassegrain describe the intended construction family; an opaque
display objective is not a calibrated lens or a ray-traced prescription.

## Rubin is a facility

The original recipe had the internal telescope and a generic shell. It now
adds a faceted upper enclosure, base and three-tier service wing, with doors
and a separation joint. Structural dimensions remain picked scenery.

Rubin's [summit-facility description][rubin] identifies the service building,
maintenance and coating facilities, clean room, stepped wind-guiding exterior
and separation between buildings. Read those as functional and visual cues;
do not turn estimated photo dimensions into measurements.

[rubin]:
https://rubinobservatory.org/explore/how-rubin-works/technology/summit-facility

## The equipment of astronomy

Reusable first-pass kinds:

- mobile crane, mirror-handling gantry and scaffold tower;
- service truck, utility pickup and equipment trailer;
- shipping crate, equipment cabinet, generator and cooling unit;
- stairs, work platform, handrails, cable tray and antenna/weather mast.

Site recipes should place these with access lanes, telescope clearances
and useful variety. Clutter should look as though work happens there.
A service truck may be parked; its recipe need not simulate a drivetrain.

## Work order

Common guide-star pointing first; personal telescope assemblies and reusable
yard primitives alongside the named summit families; Rubin's full building;
then a cluttered site composition. OWL remains requested.

Reuse radio-dish work for JCMT, SMA, VLBA, ALMA and SKA while preserving
their different feeds and enclosures. The next hero stills come after
these additions, at golden hour, with the new factory contents visible.


Prepare one reviewable slice at a time. Run small relevant checks when the
slice is ready for merge. No broad test or mutation sweep is implied.
All commits stay local on branches.

## Current study

The [Hawaiian families](hawaiian-observatories.md), three personal telescopes,
Rubin facility and reusable yard kinds are in the catalogue. The
[golden-hour site](golden-hour-site.yaml) brings them into a working campus.

The [launchers](guide-launchers.md) share the resolved beam axes and exits.
The main instrument has an explicit zero heading and a northward 55-degree
boresight. Four enclosure-clearance rays and the independent arcsecond
checks hold the intended pointing. A drone flyby remains separate work.
