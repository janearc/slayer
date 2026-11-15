# Experimental GPU mist

The Metal mist pass computes its moon-reference selection and centroid, water and tree mask maxima, lattice noise, motion field, layered veil and final blend on the GPU. Images use the bottom-up PFM row order. The package also exposes a bounded GPU lattice-hash primitive for numerical checks.

The stage currently allocates its own Metal session and buffers for each call, and it uses float32 arithmetic for operations that the CPU stage performs in float64. It refuses unsupported dimensions, masks and nonfinite values. Its allocation limit covers Metal buffers, not all process memory.

The stage is experimental: historical-frame equivalence, 4K performance, cross-device behavior and use with shared device-resident intermediate buffers remain to be established.
