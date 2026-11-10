# Summit references beside the recipes

The source shelf now has pinned local copies of the references below.
Apertures belong to their optical roles: a clear aperture, mirror blank
and enclosure diameter are different quantities. The recipes must keep
those roles when they start using these numbers.

## Maunakea

- CFHT: the observatory manual gives a 3.58 m usable primary and describes
  the yoke mount. Source key cfht-observatory-manual.
- Gemini North: the observatory handout gives an 8.1 m primary and
  enclosure/structure dimensions. Source key gemini-north-handout.
- IRTF: the IfA facility inventory describes the nominal 3.0 m telescope.
  The separate IRTF technical page was unreachable during this fetch;
  do not substitute its mirror-blank diameter for effective aperture.
- UH 2.2 m: the facility inventory supplies the nominal telescope size.
  Source key uh-maunakea-telescopes.
- UKIRT: the facility description gives its 3.8 m primary. Source key
  ukirt-telescope. UH [announced][ukirt-end] a September 15, 2026 science
  closure; a standing building is not proof of continued science.
- JCMT: the observatory description gives its 15 m submillimetre dish.
  Source key jcmt-telescope.
- SMA: the CfA description distinguishes the array from an individual
  6 m antenna. Source key sma-telescope.
- VLBA: NRAO describes each station's 25 m dish and control building.
  Source key vlba-telescope.

CSO and Hoku Ke'a are [removed structures][removed], suitable for explicitly
historical scenery. They do not belong in a current summit layout.

## Haleakala

- The Pan-STARRS project page describes two 1.8 m telescopes. Use this
  over the older facility page's prediction of PS2 operations in 2017.
  Source key panstarrs-telescopes.
- Las Cumbres describes the 2 m Faulkes family, its alt-az mount and
  Ritchey-Chretien layout. Source key faulkes-telescopes.
- NSO distinguishes DKIST's 4 m clear aperture from the larger physical
  mirror. Its enclosure and support-building pages describe separate
  functions. Source keys dkist-telescope, dkist-enclosure and
  dkist-support-buildings.
- ATLAS specifies a 0.5 m Wright-Schmidt telescope on a German equatorial
  mount. Source key atlas-specifications.
- NASA's TLRS-4 station log specifies a 0.28 m receiving and transmitting
  Schmidt-Cassegrain telescope and common coude path. Source key
  tlrs4-station-log. This does not make it a sodium guide-star instrument.
- IfA describes the 0.5 m off-axis SOLARC instrument at the Zodiacal Light
  Observatory. Its Maui history gives context for Mees and the original
  twin instruments at MSSC. Source keys ifa-haleakala-facilities and
  ifa-maui-history.
- Boeing's AEOS upgrade account gives the 3.67 m telescope diameter.
  Source key aeos-upgrade. This is a contractor's 2012 description,
  not a current inventory of every instrument at MSSC.

The older inventory pages are historical evidence. Current operations
and small-instrument membership still need their own review; the available
building and instrument families do not imply continuous operation.

## Fetch record

The normal Game source command downloaded twelve initial references.
UKIRT, the IfA Maunakea inventory and the IfA Maui history failed Go's
certificate-chain check. System curl with verified TLS fetched their
actual bytes, which were imported with ordinary source sidecars and pins.

Both Go and curl timed out connecting to the IRTF technical host. A public
Space Command article returned HTTP 403 through both tools. Neither is
represented as a successful fetch; use the pinned IfA inventory for IRTF
and obtain another primary GEODSS reference before adding measured data.

All imported files keep their actual URL, hash, size and fetch time.
No TLS verification was disabled. Source bytes stay on the private shelf.

[ukirt-end]:
https://www.hawaii.edu/news/2026/07/02/uh-continues-ukirt-decommissioning/
[removed]:
https://www.hawaii.edu/news/2024/07/02/second-observatory-removed-maunakea/

## Technical inventory details

The SMA technical paper's table on PDF page 6 supplies eight antennas and
a primary focal ratio of f/0.4. Its paper is pinned as sma-technical; the
individual 6 m aperture also has the CfA facility reference.

The 2009 Haleakala environmental study supplies a historical facility
inventory. Its table on PDF page 115 calls GEODSS a three-telescope,
metre-class installation. Elsewhere it gives a 102 cm physical main
aperture and a smaller auxiliary instrument.

The factory's GEODSS class value is consequently labelled nominal, not
a claim that every installed or upgraded optical surface is exactly
1.000 m. Keep the historical and current inventory contexts separate.

These two PDFs and the other institution pages now support 21 quantities
in the research ledger. Each has a role, source key and exact excerpt;
the PDF entries retain their physical file page numbers.

The government PDF returned HTTP 403 to the Go fetcher but downloaded
through system curl with verified TLS. Its real bytes, hash and fetch
sidecar were imported, then kept by the normal Game fetch command.
