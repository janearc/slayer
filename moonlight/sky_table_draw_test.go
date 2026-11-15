//go:build draw

package moonlight

import (
	"bytes"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestASkyReadBackIsTheSameToTheBit(t *testing.T) {
	sky := tableSky(t)
	read, err := SkyFromBytes(sky.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if *read != *sky || !bytes.Equal(read.Bytes(), sky.Bytes()) {
		t.Fatal("the sky read back is not the sky written")
	}
	for _, way := range []maths.Vec{maths.V(0, 1, 0), maths.V(1, 0.2, 0),
		maths.V(-0.3, 0.05, 0.9)} {
		if read.Light(way) != sky.Light(way) {
			t.Fatalf("toward %v: %v read back, %v worked out", way,
				read.Light(way), sky.Light(way))
		}
	}
}
