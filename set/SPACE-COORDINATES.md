# signed native space placement

A version 2 scene with explicit `setting: space` places manufactured
things in a local metre frame. Its east and north values may be signed,
within the existing schema coordinate cap of one billion metres in each
direction. `height` retains its nonnegative metre quantity contract.
This permits an object edit across the local origin without changing
terrain placement or allowing a non-finite native transform.

Version 1, unlabelled legacy scenes, and version 2 scenes with ground or
no setting retain nonnegative thing east and north values. Version 1 and
legacy files still cannot use the version 2 `setting` key. No terrain,
stand, world geometry or camera coordinate contract changes.

The version 2 thing definition holds the signed cap. A scene conditional
adds the ground minimum unless the scene explicitly says `setting: space`.
This shares the existing thing definition and all its other validation.

The version 1 schema stays unchanged. The reader determines the scene's
coordinate frame before reading its children so YAML key order cannot
change the accepted range. Its numeric reader continues to reject NaN,
infinity and values outside the frame's bounds.

Tests pair signed space reads with ground refusals, both coordinate
edges, non-finite and oversized values, height refusal, and a setting
written after the set. The embedded Ferruccio contracts must match the
producer schemas after this change.
