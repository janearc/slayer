package gpu

import "testing"

func TestNativeCountsDecodeEachSampleAndIgnoreTextureRecords(t *testing.T) {
	for _, stride := range []int{12, 23, 34} {
		samples := make([]float32, stride*2)
		samples[11] = float32(2 + 1<<4 + 1<<8)
		samples[stride+11] = float32(2 + 2<<4 + 1<<8 + 1<<12)
		got := nativeCounts(samples, stride)
		want := NativeCounts{Queries: 4, Candidates: 3, Refined: 2, Fallbacks: 1}
		if got != want {
			t.Fatalf("stride %d: %+v, want %+v", stride, got, want)
		}
	}
	if got := nativeCounts(nil, 12); got != (NativeCounts{}) {
		t.Fatal("empty dispatch reports native queries", got)
	}
}
