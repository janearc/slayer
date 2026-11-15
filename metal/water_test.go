//go:build draw && darwin

package metal

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

// waterProbe asks the library at points: x, y, z pick each function's
// arguments, and the Quan-Fry coefficients ride in the params after a
// selector.
const waterProbe = WaterLibrary + `
struct water_params {
	int ask;
	float pad;
	float qf[10];
};

kernel void water_points(device const packed_float3 *points [[buffer(0)]],
                         device float *out [[buffer(1)]],
                         constant water_params &p [[buffer(2)]],
                         uint id [[thread_position_in_grid]]) {
	float3 q = float3(points[id]);
	switch (p.ask) {
	case 0:
		out[id] = water_index(p.qf, q.x, q.y, q.z);
		return;
	case 1:
		out[id] = water_fresnel(q.x, q.y);
		return;
	case 2:
		out[id] = slope_chance(q.x, q.y, q.z, q.z * 2);
		return;
	}
	out[id] = smith_hidden(q.x, q.y);
}
`

// quanFry reads the ten coefficients from oceantex's ledger, so the
// numbers stay in sources.yaml and nowhere else.
func quanFry(t *testing.T) []float64 {
	t.Helper()
	for _, source := range oceantex.Sources() {
		if source.Name == "quan-fry.coefficients" {
			if len(source.Values) != 10 {
				t.Fatalf("quan-fry has %d values", len(source.Values))
			}
			return source.Values
		}
	}
	t.Fatal("sources.yaml has no quan-fry.coefficients")
	return nil
}

// asked packs the selector and the coefficients for the probe.
func asked(which int, qf []float64) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(int32(which)))
	out = binary.LittleEndian.AppendUint32(out, 0)
	for _, value := range qf {
		out = binary.LittleEndian.AppendUint32(out,
			math.Float32bits(float32(value)))
	}
	return out
}

// close enough: the library is 32-bit where oceantex is 64, and erfc
// is a 1.5e-7 fit, so a few millionths relative, or absolute near
// nought.
func apart(got float32, want float64) float64 {
	return math.Abs(float64(got)-want) / math.Max(math.Abs(want), 1e-6)
}

func TestTheWaterLibraryAgreesWithOceantex(t *testing.T) {
	kernel, err := Open(waterProbe, "water_points")
	if errors.Is(err, ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	qf := quanFry(t)

	// the index over oceans and estuaries, cold to warm, violet to red
	var indexAt [][3]float32
	for _, salinity := range []float32{0, 10, 35} {
		for _, celsius := range []float32{0, 12, 30} {
			for _, nm := range []float32{400, 550, 700} {
				indexAt = append(indexAt, [3]float32{salinity, celsius, nm})
			}
		}
	}
	got, err := kernel.Points(indexAt, asked(0, qf))
	if err != nil {
		t.Fatal(err)
	}
	for i, point := range indexAt {
		want := oceantex.Index(float64(point[0]), float64(point[1]),
			float64(point[2]))
		if apart(got[i], want) > 2e-6 {
			t.Errorf("index%v is %v, oceantex says %v", point, got[i], want)
		}
	}

	// fresnel from grazing to square, in water and glass
	var fresnelAt [][3]float32
	for _, cosine := range []float32{0.01, 0.2, 0.5, 0.9, 1} {
		for _, index := range []float32{1.33, 1.5} {
			fresnelAt = append(fresnelAt, [3]float32{cosine, index, 0})
		}
	}
	if got, err = kernel.Points(fresnelAt, asked(1, qf)); err != nil {
		t.Fatal(err)
	}
	for i, point := range fresnelAt {
		want := oceantex.Fresnel(float64(point[0]), float64(point[1]))
		if apart(got[i], want) > 2e-5 {
			t.Errorf("fresnel%v is %v, oceantex says %v", point, got[i], want)
		}
	}

	// the slope chance over the measured range of variances
	chanceAt := [][3]float32{{0, 0, 0.003}, {0.05, -0.02, 0.01},
		{-0.1, 0.1, 0.03}, {0.2, 0.05, 0.05}}
	if got, err = kernel.Points(chanceAt, asked(2, qf)); err != nil {
		t.Fatal(err)
	}
	for i, point := range chanceAt {
		want := oceantex.SlopeChance(float64(point[0]), float64(point[1]),
			float64(point[2]), 2*float64(point[2]))
		if apart(got[i], want) > 2e-5 {
			t.Errorf("chance%v is %v, oceantex says %v", point, got[i], want)
		}
	}

	// smith's hiding from near grazing to overhead
	hiddenAt := [][3]float32{{0.05, 0.03, 0}, {0.2, 0.03, 0},
		{0.5, 0.01, 0}, {0.95, 0.05, 0}}
	if got, err = kernel.Points(hiddenAt, asked(3, qf)); err != nil {
		t.Fatal(err)
	}
	for i, point := range hiddenAt {
		want := oceantex.Hidden(float64(point[0]), float64(point[1]))
		if apart(got[i], want) > 5e-5 {
			t.Errorf("hidden%v is %v, oceantex says %v", point, got[i], want)
		}
	}
}
