# Region comparison sets

A set names one job, a positive finite isolated-residual threshold in linear
light, and a nonempty list of named regions. Each region names its observation
and its left, top, width and height in the complete frame's pixels. All fields
are required, including zero-valued left and top. Names are unique and have no
surrounding whitespace. The job path is read as written; a caller resolves a
relative path against the set file's directory.

```yaml
job: street.yaml
threshold: 0.05
regions:
  - name: water
    observation: bright points in reflections
    left: 4
    top: 0
    width: 8
    height: 6
```

Read is given the effective complete frame's width and height. It rejects
missing fields, unknown fields, multiple YAML documents, an empty list,
duplicate names, nonpositive size or threshold, and any rectangle outside
the frame. It checks containment without adding coordinates and so cannot
wrap an integer at the edge. A successful read returns data only; it does not
open the job, draw a region or establish a reference's identity.
