//go:build draw

package manufacture_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

//go:embed testdata/summit-instruments.yaml
var summitContractRaw []byte

func TestSummitFamiliesKeepTheirSourceAperturesAndClosedGeometry(test *testing.T) {
	var data summitContract
	if err := yamlfile.Read("summit-instruments.yaml", summitContractRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	for _, profile := range data.Profiles {
		probe := summitProbe{contract: data, profile: profile}
		test.Run(profile.Kind, probe.check)
	}
}
