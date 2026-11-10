/*
 * Reuse the retained independent Vigna producer for one large birth ID.
 * The emitter's authored seed/ID packing gives packed state a473afe496151f15
 * for seed 0, ID 4934013240334081 and attribute 0.
 */
#define main splitmix_first_eight
#include "splitmix_reference.c"
#undef main
int main(void) {
    state = UINT64_C(0xa473afe496151f15) - UINT64_C(0x9e3779b97f4a7c15);
    printf("%llu\n", (unsigned long long)(next() >> 11));
    return 0;
}
