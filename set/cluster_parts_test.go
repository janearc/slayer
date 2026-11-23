package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// clusterSet is a set in space of one cluster, saying what it is given
// under the thing.
func clusterSet(says string) string {
	return `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: cluster
  setting: space
  light: none
  film: {width: 1, height: 1, rays: 1}
  cameras:
    away:
      title: away
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -100, 0]}
  set:
    things:
      cluster:
        kind: scavenged-cluster
        seed: 1
        east: 0
        north: 0
` + says
}

func TestAClusterSaysItsPartsItsCountAndItsBandSpacing(t *testing.T) {
	source := clusterSet(
		"        parts: {sls-core-tank: 6, alfa-hero: 2, winnebago: 2}\n" +
			"        count: 9\n" +
			"        band_spacing: {value: 4, unit: m}\n")
	read, err := set.Read("cluster.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	cluster := read.Things()[0]
	if got := cluster.Parts(); len(got) != 3 || got["sls-core-tank"] != 6 ||
		got["winnebago"] != 2 {
		t.Errorf("the parts are %v", got)
	}
	if cluster.PartCount() != 9 || cluster.BandSpacing() != 4 {
		t.Errorf("%d parts at a spacing of %v", cluster.PartCount(),
			cluster.BandSpacing())
	}
}

func TestAClusterThatSaysNothingLeavesItAllToTheData(t *testing.T) {
	source := clusterSet("")
	read, err := set.Read("cluster.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	cluster := read.Things()[0]
	if cluster.Parts() != nil || cluster.PartCount() != 0 ||
		cluster.BandSpacing() != 0 {
		t.Errorf("a bare cluster says %v, %d, %v", cluster.Parts(),
			cluster.PartCount(), cluster.BandSpacing())
	}
}

func TestAClusterThatNamesAPartWrongIsRefused(t *testing.T) {
	for name, says := range map[string]string{
		"a kind nobody has":      "        parts: {no-such-kind: 1}\n",
		"a cluster as a part":    "        parts: {scavenged-cluster: 1}\n",
		"a weight of nothing":    "        parts: {yurt: 0}\n",
		"a negative weight":      "        parts: {yurt: -1}\n",
		"no parts at all":        "        parts: {}\n",
		"a count of nothing":     "        count: 0\n",
		"a spacing in feet":      "        band_spacing: {value: 4, unit: ft}\n",
		"a spacing of nothing":   "        band_spacing: {value: 0, unit: m}\n",
		"a count that is partly": "        count: 2.5\n",
	} {
		source := clusterSet(says)
		_, err := set.Read("cluster.yaml", []byte(source))
		if err == nil || !strings.Contains(err.Error(), "cluster.yaml") {
			t.Errorf("%s was read: %v", name, err)
		}
	}
}
