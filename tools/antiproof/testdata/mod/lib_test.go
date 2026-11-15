package fixture

import (
	"testing"
	"time"
)

func TestDouble(t *testing.T) {
	if Double(3) != 6 {
		t.Fatal("Double(3) is not 6")
	}
}

func TestSlow(t *testing.T) {
	time.Sleep(60 * time.Second)
}
