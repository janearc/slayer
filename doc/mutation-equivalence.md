# mutation equivalence notes

These arguments apply only to the exact mutations listed in `.mutants`.
They do not excuse untested changes to authored appearance or physics.

## placement tolerance

`Len()` returns binary64. For a length in [0.5, 2], subtracting one is
exact by Sterbenz's lemma and gives an integer multiple of 2^-53.
Outside that interval, the distance from one is at least 0.5.

The binary64 value of 1e-9 divided by 2^-53 is exactly
4835703278458517/536870912, which is not an integer. Equality at that
length tolerance is therefore unreachable.

This argument does not apply to the dot-product checks: their equality is
reachable and the
tests exercise it with the next representable values on either side.

The handedness check follows three unit-length and three orthogonality
checks with tolerance 1e-9. Their Gram matrix has diagonal greater than
(1-1e-9)^2 and off-diagonal magnitudes less than 1e-9. Its least
eigenvalue exceeds (1-1e-9)^2-2e-9 by Gershgorin's bound.

The determinant is consequently near +1 or -1, separated from zero by much
more than
binary64 arithmetic error. Replacing its positive test by a nonnegative
test cannot admit a singular basis that passed the preceding checks.
