# spacecraft mission census

compare the [mission survey](../../../cmd/catalogue/mission-census.json)
with the factory:

```sh
game run catalogue-census -- jimo new-horizons cassini huygens
```

the command emits JSON with source references, launch status, model
availability and component counts. an absent model has a null count and a
`not-built` review. existing models report their count and a `pending`
review. the report includes a 600-part reference so readers can compare
model detail without inferring accuracy from triangle count.

this is a bounded survey. its seed covers 60 science missions and concepts
from NASA, ESA and JAXA, with later weather satellite entries. unlaunched
concepts such as JIMO and Darwin are included. the input records which
mission or concept each entry represents. Cassini and Huygens have separate
records because they describe different spacecraft.

for a model’s physical bounds and known simplifications, see the
[model inventory](model-maturity.md). the survey references identify the
mission. model dimensions and finishes have their own provenance.
