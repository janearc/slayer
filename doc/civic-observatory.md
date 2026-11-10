# Griffith, on a smaller stage

Griffith is a place to visit as much as a place to observe. Its recipe gets
a wide terrace, two wings, a tall central rotunda, a low planetarium behind
it and an instrument dome on each end. The family is available as
kind: griffith through both scene readers and the catalogue:

    game run catalogue -- render griffith --out out/griffith

The [official visitor map](https://griffithobservatory.org/wp-content/uploads/2021/03/GOMap_03112021.pdf),
main level, supplies those building and instrument roles. The map was read
as a layout, not scaled into a survey.

Local +X is east; the entrance faces -Z. The east dome has a slit and a
long refractor. The west solar-instrument roof stays closed: there is no
coelostat mechanism or solar optical path.

The nominal objective is 12 inches, or exactly 0.3048 m using the
international inch. It comes from page 17 of the observatory's
[student guide](https://griffithobservatory.org/wp-content/uploads/2023/02/Module1_Student_Guide.pdf),
pinned in sources.yaml and checked against research-derived-data.yml.

Both original PDFs stay on the source shelf.

Every building dimension and material proxy is picked in
manufacture/ground-choices.yaml. So are the tube length, mounting height,
roof subdivision and objective thickness.

The small objective is opaque scenery with a polished surface, not a lens
that forms an image. Tube length does not stand in for a published focal
length.

All four detail levels keep the same envelope and objective size; they
change circular tessellation. Each box, drum, cap and slit shell is closed.

Some structural pieces overlap, as scenery assemblies do; this is not a
boolean solid suitable for architectural fabrication. The renderer receives
ordinary meshes and materials through the existing CPU and GPU routes.
