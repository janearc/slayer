# authored appearance samples

appearance.txt records 120 local-coordinate absorption and temperature
samples from the unchanged production source at 90bc11d. These hold picked
noise and shape choices; they are not measured fire data or a physical
oracle. The contract test allows small floating-point differences.

From the module root, `game run sample-flame` prints the samples.
Regenerate only when reviewing an intentional appearance change, inspect
the resulting renders, and keep the independent physics checks intact.

ragged.txt adds fifteen fixed points at cooler soot patches and exact
continuous-flame borders. Their absorption and temperature are authored
appearance records from 69f294a, not laboratory data. The producer is
`game run sample-flame-rags`.
