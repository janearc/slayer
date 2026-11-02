package manufacture

import "testing"

// a seed's draws are ninja's unit and across, which stay the one place
// the arithmetic is written: the kernel mirrors that function, so a
// second copy here would be a second thing to keep in step.
func TestASeedDrawsWhatUnitDraws(t *testing.T) {
	number := uint64(20261006)
	from := sow(number)
	for place := range 60 {
		part, index, which := place/12, place/3%4, place%3
		got := from.draw(part, index, which)
		want := unit(number, part, index, which)
		if got != want {
			t.Fatalf("draw(%d, %d, %d) is %v, unit says %v", part,
				index, which, got, want)
		}
		swing := from.swing(part, index, which)
		if swing != across(number, part, index, which) {
			t.Fatalf("swing(%d, %d, %d) is %v, not across's", part,
				index, which, swing)
		}
	}
}

func TestADrawIsAShareAndASwingIsEitherSideOfNought(t *testing.T) {
	from := sow(3)
	var low, high int
	for index := range 4000 {
		share := from.draw(index, 1, 2)
		if share < 0 || share >= 1 {
			t.Fatalf("draw %d is %v, not a share", index, share)
		}
		swing := from.swing(index, 1, 2)
		if swing < -1 || swing >= 1 {
			t.Fatalf("swing %d is %v, past one either way", index,
				swing)
		}
		if swing != 2*share-1 {
			t.Fatalf("swing %d is %v, not its draw %v stretched",
				index, swing, share)
		}
		if swing < 0 {
			low++
		} else {
			high++
		}
	}
	// a fair coin lands within a tenth of even in 4000 throws far more
	// often than a test can be run
	if low < 1600 || high < 1600 {
		t.Errorf("%d swings below nought and %d above", low, high)
	}
}

func TestTheSameSeedIsTheSameAndAnotherIsNot(t *testing.T) {
	one, same, other := sow(11), sow(11), sow(12)
	alike := 0
	for index := range 64 {
		if one.draw(index, 0, 0) != same.draw(index, 0, 0) {
			t.Fatalf("draw %d of one seed changed", index)
		}
		if one.draw(index, 0, 0) == other.draw(index, 0, 0) {
			alike++
		}
	}
	if alike > 0 {
		t.Errorf("%d of 64 draws alike from two seeds", alike)
	}
}

func TestASeedNeverSownIsNotMade(t *testing.T) {
	if (seed{}).made() {
		t.Error("an empty seed says it was sown")
	}
	if !sow(0).made() {
		t.Error("a seed sown from nought says it was not")
	}
}
