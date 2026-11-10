/*
 * Oracle for the attributed SplitMix64 finalizer.
 * Sebastiano Vigna's 2015 public-domain implementation:
 * https://prng.di.unimi.it/splitmix64.c
 * It redirects to https://prng.di.unimi.it/splitmix.c.
 *
 * Output pairs are birth ID and top 53 bits for an initial state of zero.
 * This producer is kept beside the Go test vectors; it is not the emitter.
 */
#include <stdint.h>
#include <stdio.h>

static uint64_t state;
static uint64_t next(void) {
    uint64_t z = (state += UINT64_C(0x9e3779b97f4a7c15));
    z = (z ^ (z >> 30)) * UINT64_C(0xbf58476d1ce4e5b9);
    z = (z ^ (z >> 27)) * UINT64_C(0x94d049bb133111eb);
    return z ^ (z >> 31);
}
int main(void) {
    for (unsigned int birth = 1; birth <= 8; ++birth) {
        printf("%u 0x%llx\n", birth,
               (unsigned long long)(next() >> 11));
    }
    return 0;
}
