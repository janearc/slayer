package reptiles

import (
	"bytes"
	"testing"
)

func TestMalformedReptileProfileErrors(t *testing.T) {
	_, raw := Embedded()
	for _, tc := range []struct{ old, replacement, want string }{
		{"fin-roots: [0.65, -2.10]", "fin-roots: [.nan, -2.10]", "invalid pliosaur landmarks"},
		{"[0.65, 0.45, 0.14, 0]", "[0.65, .inf, 0.14, 0]", "invalid pliosaur fin section 0"},
		{"[-5.00, 0, 0.015, 0.015]", "[.nan, 0, 0.015, 0.015]", "invalid pliosaur section 0"},
		{"[-0.24, 0, 0, 0.79, 0.26, 0.34]", "[-0.24, 0, 0, 0.79, .inf, 0.34]", "invalid pliosaur mass 0"},
		{"limb-roots: [0.43, -0.40]", "limb-roots: [.nan, -0.40]", "invalid crocodylian limb roots"},
		{"[0, -0.075, 0.26]", "[.inf, -0.075, 0.26]", "invalid crocodylian limb point 0"},
		{"[-1.70, -1.28, 0.018, 0.095, 0.012, 0.055]", "[.nan, -1.28, 0.018, 0.095, 0.012, 0.055]", "invalid crocodylian station 0"},
		{"[-1.25, 2.55, 0, 0.04, 0.05]", "[.inf, 2.55, 0, 0.04, 0.05]", "invalid pterosaur body station 0"},
		{"[-1.30, 2.54, 0.04, 0.04]", "[.nan, 2.54, 0.04, 0.04]", "invalid pterosaur cage station 0"},
		{"[-0.80, 0, 0, 0.35, 0.20, 0.16, 0.95]", "[-0.80, 0, 0, .inf, 0.20, 0.16, 0.95]", "invalid pterosaur density mass 0"},
		{"[0.32, 2.70, 0.36]", "[.inf, 2.70, 0.36]", "invalid pterosaur wing rig"},
		{"[-0.35, 0.12, 2.18]", "[.nan, 0.12, 2.18]", "invalid pterosaur cage rig"},
		{"    fin-roots: [0.65, -2.10]\n", "", "invalid pliosaur controls"},
		{"    scale: 3\n", "    scale: 3\n    yaw: 0.1\n", "invalid pliosaur controls"},
	} {
		t.Run(tc.want+"/"+tc.old, func(t *testing.T) {
			bad := bytes.Replace(raw, []byte(tc.old), []byte(tc.replacement), 1)
			if bytes.Equal(bad, raw) {
				t.Fatalf("missing fixture %q", tc.old)
			}
			_, err := Read("damaged.yaml", bad)
			if err == nil || err.Error() != "damaged.yaml: "+tc.want {
				t.Fatalf("got %v; want %s", err, tc.want)
			}
		})
	}
}
