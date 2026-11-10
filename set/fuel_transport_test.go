package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

const heatVelocity = "      heat-transport:\n" +
	"        east: {value: 0.5, unit: m/s}\n" +
	"        south: {value: -0.25, unit: m/s}\n"

func TestHeatTransportKeepsSignedComponentsAndRefusesNonfiniteInputAtReadTime(test *testing.T) {
	base := bedFixture(test)
	raw := strings.Replace(base, "      fuel: 1", heatVelocity+"      fuel: 1", 1)
	if raw == base {
		test.Fatal("heat transport insertion anchor moved")
	}
	read, err := set.Read("transport.yaml", []byte(raw))
	if err != nil {
		test.Fatal(err)
	}
	east, south := read.FuelBeds()[0].HeatTransport()
	if east != .5 || south != -.25 {
		test.Fatal("signed heat velocity changed", east, south)
	}
	for _, change := range [][2]string{
		{"value: 0.5, unit: m/s", "value: .inf, unit: m/s"},
		{"value: -0.25, unit: m/s", "value: -.inf, unit: m/s"},
		{"value: 0.5, unit: m/s", "value: .nan, unit: m/s"},
		{"value: 0.5, unit: m/s", "value: 0.5, unit: km/s"},
		{"        south: {value: -0.25, unit: m/s}\n", ""},
	} {
		bad := strings.Replace(raw, change[0], change[1], 1)
		if bad == raw {
			test.Fatal("ineffective heat transport refusal", change)
		}
		if _, err := set.Read("bad-transport.yaml", []byte(bad)); err == nil {
			test.Fatal("invalid heat velocity reached preparation", change)
		}
	}
}
