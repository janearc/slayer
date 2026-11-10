package embers

import "testing"

// These are the high 53 bits of the first eight SplitMix64 outputs from
// state zero. At attribute zero, our birth IDs select those same states.
// The independent C producer is testdata/splitmix_reference.c; its source
// is Sebastiano Vigna's public-domain implementation:
// https://prng.di.unimi.it/splitmix.c
// The separate attribute salt is an authored rendering choice.
func TestEmberFinalizerMatchesThePublishedSplitMixSequence(t *testing.T) {
	expected := []uint64{
		0x1c4415072f63b9, 0xdcf13cd54372c, 0xd88ba3100128, 0x1f1177150e4990,
		0x367312d4a350e, 0xa7973e18e8fd4, 0x5905357c3e8a6, 0x18b082675922d5,
	}
	for i, want := range expected {
		got := uint64(share(0, int64(i+1), 0) * (1 << 53))
		if got != want {
			t.Fatalf("birth %d high bits %#x, reference %#x", i+1, got, want)
		}
	}
}
