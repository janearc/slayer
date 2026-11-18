# Spitzer maturity work

The solar assembly has two chevron leaves with 392 separate cells on
each. The body, bus, instruments and thermal surfaces remain simplified.

The [mission description](https://www.spitzer.caltech.edu/mission/the-solar-panel-assembly)
specifies the cell count and 5.5 by 6.5 cm cell dimensions. Section 2.1 of
[Werner et al. (2022)](https://doi.org/10.1117/1.JATIS.8.1.014002)
specifies 330 by 72 cm active panels and an approximately 160 cm upper
extension. Both sources are pinned in `sources.yaml`; their measured
numbers live in `research/research-derived-data.yml`.

Seven columns and 56 rows fit the published count and dimensions. That
packing, the 20-degree chevron fold, panel thickness and mounting offset
are authored choices. Each cell is a separate closed box. Each leaf has
one substrate, one upper extension and eight continuous reflector strips.
The strips do not claim a measured segmentation of the real coating.

The selected panel dimensions sum to 4.9 m, while the paper's observatory
table lists 4.5 m overall. No undocumented overlap was invented to remove
that discrepancy. The mission page describes half the panel area as cells;
the paper says 60 percent. The specified count and cell dimensions occupy
about 59 percent of the specified active panel area. These differences
remain open for reconciliation against drawings or another primary source.

Cells use measured bulk gallium-arsenide optical constants from Aspnes,
Kelso, Logan and Bhat (1986), DOI 10.1063/1.337426. The unchanged CC0 table
is in `materials/testdata/main/GaAs/nk/Aspnes.yml`, obtained from the
refractiveindex.info database at tag `v2026-05-24`. This is an optical
proxy.

The real GaAs/Ge cell, cover glass and antireflection coating need
their own layered model. The reflector strips still use the existing
thermal-shield appearance; aluminized Teflon is not yet represented.

The inventory is independent of tessellation: 784 closed cell components
at every detail level.
