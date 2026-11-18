package assets

import (
	"encoding/json"
	"fmt"
	"strings"
)

type annotation struct {
	Description string                `json:"description"`
	Properties  map[string]annotation `json:"properties"`
}

// Guidance reads a field's prose from the bundled schema, not a second contract.
func Guidance(path ...string) (string, error) {
	raw, err := Read("job.schema.json")
	if err != nil {
		return "", err
	}
	var field annotation
	if err := json.Unmarshal(raw, &field); err != nil {
		return "", err
	}
	for _, key := range path {
		child, ok := field.Properties[key]
		if !ok {
			return "", fmt.Errorf("no schema guidance for %s", strings.Join(path, "."))
		}
		field = child
	}
	if field.Description == "" {
		return "", fmt.Errorf("field has no description")
	}
	return field.Description, nil
}
