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

// ecosystemV2Schema is the version 2 schema's identity, never a network
// request.
const ecosystemV2Schema = "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"

//go:embed ecosystem-v2.schema.json
var ecosystemV2Raw []byte

var ecosystemV2, ecosystemV2Error = yamlfile.NewSchema(ecosystemV2Schema,
	ecosystemV2Raw)

// saysVersionTwo is whether a set file names the version 2 schema.
func saysVersionTwo(root *yaml.Node) bool {
	return valueOf(root, "$schema") == ecosystemV2Schema
}

// checkedEcosystem applies schema validation to documents opting into a
// schema: version 2 for a file that names it, version 1 for any other
// name. Legacy set documents, which name none, retain their existing
// validation and diagnostics.
func checkedEcosystem(file string, raw []byte, root *yaml.Node) error {
	if !says(root, "$schema") {
		return nil
	}
	if saysVersionTwo(root) {
		if ecosystemV2Error != nil {
			return ecosystemV2Error
		}
		_, err := ecosystemV2.Read(file, raw)
		return err
	}
	if ecosystemError != nil {
		return ecosystemError
	}
	_, err := ecosystem.Read(file, raw)
	return err
}
