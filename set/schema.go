package set

import (
	_ "embed"

	"github.com/janearc/puzzlemouth/yamlfile"
	yaml "gopkg.in/yaml.v3"
)

// EcosystemSchema is the bundled schema's identity, never a network request.
const EcosystemSchema = "https://github.com/janearc/slayer/schemas/ecosystem-v1.json"

//go:embed ecosystem.schema.json
var ecosystemRaw []byte

var ecosystem, ecosystemError = yamlfile.NewSchema(EcosystemSchema, ecosystemRaw)

// checkedEcosystem applies schema validation to documents opting into v1.
// Legacy set documents retain their existing validation and diagnostics.
func checkedEcosystem(file string, raw []byte, root *yaml.Node) error {
	if !says(root, "$schema") {
		return nil
	}
	if ecosystemError != nil {
		return ecosystemError
	}
	_, err := ecosystem.Read(file, raw)
	return err
}
