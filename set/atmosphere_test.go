package set_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/set"
)

//go:embed testdata/atmosphere-contract.yaml
var atmosphereContractRaw []byte

type atmosphereContract struct {
	Block   string  `json:"block"`
	Weather string  `json:"weather"`
	Albedo  float64 `json:"albedo"`
	Invalid []struct {
		Before string `json:"before"`
		After  string `json:"after"`
	} `json:"invalid"`
}

func readAtmosphereContract(test *testing.T) atmosphereContract {
	test.Helper()
	var contract atmosphereContract
	if err := yamlfile.Read("testdata/atmosphere-contract.yaml", atmosphereContractRaw, &contract); err != nil {
		test.Fatal(err)
	}
	return contract
}

func TestAtmosphereSchemaAndDomainKeepRegionalWeatherExplicit(test *testing.T) {
	contract := readAtmosphereContract(test)
	for _, raw := range laserSources() {
		source := strings.ReplaceAll(raw, "  weather:\n", contract.Block+"  weather:\n")
		made, err := set.Read("atmosphere.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		name, albedo := made.Atmosphere()
		if name != contract.Weather || albedo != contract.Albedo {
			test.Fatal("lost regional appearance", name, albedo)
		}
		for _, invalid := range contract.Invalid {
			broken := strings.ReplaceAll(source, invalid.Before, invalid.After)
			if broken == source {
				test.Fatal("ineffective atmosphere fault", invalid)
			}
			if _, err := set.Read("bad-atmosphere.yaml", []byte(broken)); err == nil {
				test.Fatal("accepted invalid atmosphere", invalid)
			}
		}
	}
}
