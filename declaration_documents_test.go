package mwanachamaagency

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestEveryEmbeddedDeclarationIsExactlyOneJSONValue(t *testing.T) {
	for name, raw := range map[string][]byte{
		"agency.blueprint.json":  blueprintJSON,
		"agency.agency.json":     domainJSON,
		"agency.operations.json": operationsJSON,
	} {
		dec := json.NewDecoder(bytes.NewReader(raw))
		var first any
		if err := dec.Decode(&first); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		var trailing any
		if err := dec.Decode(&trailing); err != io.EOF {
			t.Errorf("%s carries data after its first JSON value, which a first-value loader drops silently (AG90): %v", name, err)
		}
	}
}
