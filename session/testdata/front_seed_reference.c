#include <inttypes.h>
#include <stdint.h>
#include <stdio.h>

/* Fixed authored staging fixture: hillside seed 7, four fuel targets.
 * The established front lanes are 2, 3, 4, 5. This is an independent
 * uint64 C calculation, not experimental or botanical reference data.
 * The mixer constants and XOR convention are Slayer main.go choices;
 * they use the SplitMix64 finalizer, not its additive state sequence.
 */
int main(void) {
    for (uint64_t lane = 2; lane <= 5; ++lane) {
        uint64_t word = UINT64_C(7) ^
            (lane * UINT64_C(0x9e3779b97f4a7c15));
        word = (word ^ (word >> 30)) * UINT64_C(0xbf58476d1ce4e5b9);
        word = (word ^ (word >> 27)) * UINT64_C(0x94d049bb133111eb);
        word ^= word >> 31;
        printf("%" PRIu64 "\n", word);
    }
    return 0;
}
