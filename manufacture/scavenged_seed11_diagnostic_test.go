package manufacture

import (
	"github.com/janearc/slayer/maths"
	"runtime"
	"testing"
)

func TestScavengedClusterSeed11ProducesSceneMeshes(t *testing.T) {
	for _, detail := range []string{"hull", "quick", "demo"} {
		t.Run(detail, func(t *testing.T) {
			made, err := (scavengedRow{}).madeAt(sow(11), detail)
			if err != nil {
				t.Fatal(err)
			}
			if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
				// Unmodified f8ddbbbb and b74ee1ba both failed this case. These
				// are corrected references independently captured from f8 with
				// the same touching-parts fix, not historical successes.
				want := map[string]string{
					"hull":  "75b8dc7394451047ab719931ff34660edc28c935568407a4fc5f8f7c5474f97c",
					"quick": "b37efef6fef94c7a68d0b72b715ec7cf2083bb2dd2ab85fa7457febf2acfe236",
					"demo":  "9ef33373e2c9b481e61e62194c0f4eaf21ed1261bedb1fb805d9608c0006ab8f",
				}
				if got := referenceDigest(t, made); got != want[detail] {
					t.Fatalf("got %s, want corrected f8 %s", got, want[detail])
				}
			} else if _, err := made.Things(maths.Subspace{}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
