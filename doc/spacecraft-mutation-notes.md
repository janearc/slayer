# Spacecraft mutation review

The first complete instrument scope against the frozen `68f7ba3` snapshot
finished with 303 kills, 13 unbuildable mutations and 22 survivors. Its raw
result remains separate from any replay with the added regression tests.

The first replay had 10 kills, eight equivalents and four survivors. It
exposed a default-matching optical test; a distinct 32 m enclosure now
detects all three substitutions. The final four-case replay killed all four.

The regenerated guard scope passed: 16 kills and two equivalents. Mapping
the actual reports back to the original scope resolves all 338 cases as
315 kills, ten equivalents and 13 unbuildable mutations. No survivors or
timeouts remain in that scope. This is not a second full run.

## Changes the tests must detect

An assembly must preserve its first construction error and return no
specimen after that error. Invalid tube radius followed by a different
invalid run exercises the two rod guards and the final refusal together.

The public maker must refuse an unsown seed. A very small aperture must
also fail cleanly if floating-point geometry collapses; it must never
dereference a missing assembly result. These are public error contracts,
not tests that change embedded application data.

`hexRings` promises the inverse of complete rings around an empty centre.
The shipped eighteen panels alone do not distinguish every discriminant
fault. Counting successive rings independently exercises that mathematical
contract over several populations.

The optical recipe adapter must construct its requested dimensions instead
of substituting the named default. A zero-value recipe must return an error
and no construction. The legacy spacecraft reader must require an explicit
kind, even when the remaining spacecraft fields happen to be valid.

## Exact changes that appear equivalent

These arguments apply to ten exact sites entered in `.mutants`.
The other observed faults have regression-test kills in the replay.

The two rod error guards originally had the same textual mutation name.
The orientation error now has a distinct name, `placementError`, so its
equivalence entry cannot also excuse the reachable tube error.

Replay the original survivor report and the regenerated complete `rod`
scope; the original report cannot select the renamed mutation.

The spacecraft reader had the same ambiguity between its key-reading and
required-field guards. `requiredError` now names only the latter. Replay
that whole function too; its equivalent must not excuse a key-reading fault.

* The `spaceAlong` error guard in `instrumentAssembly.rod` follows a
  successful `newTube`. The latter already checked the positive finite
  length derived from the unchanged `run.Len()`. That is `spaceAlong`'s
  only refusal; failed scene-unit conversion is caught by `newTube`.
* `Spacecraft` adds two vector lengths, each computed by `sqrt(v.Dot(v))`.
  Finite lengths are at most `sqrt(MaxFloat64)`; their sum cannot overflow.
  Subtraction is finite too; infinite or NaN lengths make either form
  non-finite.
* Spitzer's immutable housing-radius proportion is one and its panel-width
  proportion is two. Therefore the two operands of the observed max/min
  mutation are equal for every accepted aperture. Apertures that overflow
  the multiplication already fail the envelope length check either way.
* Dividing by the panel-width proportion instead of multiplying makes
  the other operand at most one quarter of the aperture. The housing
  radius still supplies the same maximum. This argument depends on the
  current embedded proportions and must be revisited if they change.
* The two `spacecraftValue` panic guards read immutable embedded bytes and
  names supplied by the shipped templates. A malformed document or missing
  name requires a source edit; no public input changes that document or
  those names. Keep both guards for future source mistakes.
* Each of the two observed Webb bus-coordinate changes replaces
  multiplication by the exactly represented `spaceOne` with division by
  that same one. It preserves finite values, signed zero and infinities.
* `scenePositiveInfinity` is used as the sign argument to `math.Inf`.
  Replacing positive one with positive two selects the same infinity.
* The reader reaches `spacecraftRecipe` only with `kind: space-observatory`.
  Omitting that kind prevents dispatch. Omitting the nested spacecraft
  block leaves zero parameters, refused by `Construction` immediately
  after the required-field check. Keep both checks and their distinct errors.

No appearance change is excused by these arguments. Independent aperture,
silhouette, closed-mesh and CPU/Metal comparisons remain required.
