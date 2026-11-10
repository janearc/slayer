package embers

import (
	"math"
	"math/big"
	"testing"
	"time"
)

// At long clocks, rounding a birth through floating nanoseconds can move an
// expired candidate into the apparent lifetime. The independent rational
// calculation below holds this actual boundary without copying the window.
func TestDistantBirthWindowDoesNotAdmitAnAlreadyExpiredCandidate(t *testing.T) {
	const id int64 = 4934013240334081
	const now time.Duration = 4934013240340724605
	const lifetime time.Duration = 6641796
	const phaseBits int64 = 8962399066576436
	rate := math.Nextafter(1e6, 0)
	phase := new(big.Rat).SetFrac64(phaseBits, 1<<53)
	birth := new(big.Rat).Add(new(big.Rat).SetInt64(id), phase)
	birth.Quo(birth, new(big.Rat).SetFloat64(rate))
	birth.Mul(birth, big.NewRat(1_000_000_000, 1))
	exactNanoseconds := new(big.Int).Quo(birth.Num(), birth.Denom()).Int64()
	exactAge := now - time.Duration(exactNanoseconds)
	if exactAge != 6642036 || exactAge < lifetime {
		t.Fatalf("independent expired-birth control age %d", exactAge)
	}
	// The phase is from the retained C finalizer producer, not share().
	if got := share(0, id, 0) * (1 << 53); got != float64(phaseBits) {
		t.Fatalf("large birth phase %.0f, independent C result %d", got, phaseBits)
	}
	spec := fixture()
	spec.Seed, spec.Rate, spec.Life = 0, rate, lifetime
	source, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan []Particle, 1)
	go func() { done <- source.At(now) }()
	var particles []Particle
	select {
	case particles = <-done:
	case <-time.After(time.Second):
		t.Fatal("distant snapshot did not finish within one second")
	}
	laterFound := false
	for _, particle := range particles {
		if particle.ID() == id {
			t.Fatal("window admitted a birth already expired in exact arithmetic")
		}
		if particle.ID() == id+4 {
			laterFound = true
		}
	}
	if !laterFound {
		t.Fatal("positive control: later living birth was omitted")
	}
}
