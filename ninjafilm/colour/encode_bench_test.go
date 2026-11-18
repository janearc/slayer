package colour

import "testing"

var sinkB uint8

func BenchmarkEncode(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkB += Encode(float64(i%1000) / 999)
	}
}

// BenchmarkEncodeByCurve is the old Encode, for the same machine and the same
// minute as BenchmarkEncode.
func BenchmarkEncodeByCurve(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkB += oldEncode(float64(i%1000) / 999)
	}
}
