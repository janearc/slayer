# placing things in space

use the [v2 schema](ecosystem-v2.schema.json) and set `scene.setting` to
`space`. a manufactured thing can use `east` and `north` from minus one
billion to plus one billion metres. set both. `height` is an optional
nonnegative metre quantity. use `position` with metre quantities for
`east`, `north` and `up` when you need all three axes to be signed.

```yaml
position:
  east: {value: -15, unit: m}
  north: {value: 20, unit: m}
  up: {value: -3, unit: m}
heading: {value: 90, unit: deg}
```

use one position form per thing. `heading` turns clockwise from north.
the fixed camera’s `location.at` and `orientation.look_at` triples are
`[east, north, up]`. the corresponding axes in Go are x east, y up and z
south.

legacy scenes, v1 scenes and v2 ground scenes keep nonnegative `east` and
`north` in the older placement form. their `position` coordinates stay
signed. see [sets](../doc/sets.md#places) for terrain rebasing and
[Webb at L2](../sets/webb-at-l2.yaml) for a complete space scene.
