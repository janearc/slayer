#include <stdint.h>
#include <inttypes.h>
#include <stdio.h>

/* Authored Slayer seed-lane examples; not measured fire data. */
static uint64_t mixed(uint64_t seed, uint64_t lane) {
    uint64_t word = seed ^ (lane * UINT64_C(0x9e3779b97f4a7c15));
    word = (word ^ (word >> 30)) * UINT64_C(0xbf58476d1ce4e5b9);
    word = (word ^ (word >> 27)) * UINT64_C(0x94d049bb133111eb);
    return word ^ (word >> 31);
}
static uint64_t inverse(uint64_t odd) {
    uint64_t value = 1;
    for (int k = 0; k < 6; ++k) { value *= 2 - odd * value; }
    return value;
}
static uint64_t unxor(uint64_t word, int shift) {
    uint64_t value = word;
    for (int k = shift; k < 64; k += shift) { value ^= word >> k; }
    return value;
}
static uint64_t unmixed(uint64_t word) {
    word = unxor(word, 31) * inverse(UINT64_C(0x94d049bb133111eb));
    word = unxor(word, 27) * inverse(UINT64_C(0xbf58476d1ce4e5b9));
    return unxor(word, 30);
}
int main(void) {
    printf("manual fire seed7 request3 index0: %" PRIu64 "\n",
           mixed(mixed(UINT64_C(7), UINT64_C(4)), UINT64_C(1)));
    printf("zero first-target hillside seed: %" PRIu64 "\n",
           UINT64_C(2) * UINT64_C(0x9e3779b97f4a7c15));
    printf("zero first-target result: %" PRIu64 "\n",
           mixed(UINT64_C(2) * UINT64_C(0x9e3779b97f4a7c15), UINT64_C(2)));
    uint64_t target = 0 - UINT64_C(320) * UINT64_C(0x9e3779b97f4a7c15);
    uint64_t hillside = unmixed(target) ^
        (UINT64_C(2) * UINT64_C(0x9e3779b97f4a7c15));
    if (mixed(hillside, UINT64_C(2)) != target) { return 1; }
    printf("deadline first-target hillside seed: %" PRIu64 "\n", hillside);
    printf("deadline first-target source seed: %" PRIu64 "\n", target);
    printf("deadline birth320 hash input: %" PRIu64 "\n",
           target + UINT64_C(320) * UINT64_C(0x9e3779b97f4a7c15));
    return 0;
}
