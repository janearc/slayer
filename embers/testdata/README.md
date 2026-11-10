# authored release samples

appearance.txt records six firebrands from the unchanged production source
at 90bc11d. It holds the authored mapping of seed, birth and attribute to a
release, separate from the published SplitMix vectors and RK4 motion oracle.
It is not measured firebrand data. Position and heat allow small roundoff.

From the module root, `game run sample-embers` prints the samples.
Regenerate only when reviewing an intentional change to release appearance.

distant_birth_reference.c reuses the independent C finalizer for a large
authored birth ID. It prints the phase integer 8962399066576436. Compile it
with cc embers/testdata/distant_birth_reference.c -o /tmp/distant-birth
from the module root, then run /tmp/distant-birth.

The Go expiry check uses that phase and exact rational arithmetic. At the
recorded large clock the candidate is already expired by 240 ns. Ordinary
floating nanoseconds round its birth 503 ns later and make it appear alive.
The existing candidate window correctly excludes it; a wider window would
admit it. This is a numerical boundary regression, not measured fire data.
