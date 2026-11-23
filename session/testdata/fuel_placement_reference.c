#include <stdint.h>
#include <inttypes.h>
#include <stdio.h>

/* Authored replay fixtures, not measured fuel or botanical data.
   Seed 7, a 16-by-8 bed, 0.5 m cells, centre 1050E/8960N.
   Print final world positions to keep the Go assertion independent
   of the code that chooses jitter and assigns patch seeds. */
static uint64_t mixed(uint64_t seed, uint64_t lane) {
    uint64_t word = seed ^ (lane * UINT64_C(0x9e3779b97f4a7c15));
    word = (word ^ (word >> 30)) * UINT64_C(0xbf58476d1ce4e5b9);
    word = (word ^ (word >> 27)) * UINT64_C(0x94d049bb133111eb);
    return word ^ (word >> 31);
}

int main(void) {
    const int indices[] = {0, 1, 17, 63, 96, 127};
    for (unsigned k = 0; k < sizeof(indices)/sizeof(indices[0]); ++k) {
        int index = indices[k], x = index % 16, z = index / 16;
        double east = 1046 + (x + .5) * .5;
        double north = 8962 - (z + .5) * .5;
        /* Go's stir accepts a zero-based index; mixed takes its lane. */
        double dx = ((double)(mixed(7, index + 2002) % 10000) / 10000 - .5) * .5 * .7;
        double dz = ((double)(mixed(7, index + 4002) % 10000) / 10000 - .5) * .5 * .7;
        printf("index=%d east=%.17g north=%.17g flameSeed=%" PRIu64 "\n",
               index, east + dx, north - dz, mixed(7, index + 2));
    }
    return 0;
}
