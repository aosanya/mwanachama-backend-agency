package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func declaredTools(t *testing.T) []toolShape {
	t.Helper()
	shapes, err := toolsFor(mwanachamaagency.ManagerShape())
	if err != nil {
		t.Fatalf("tools: %v", err)
	}
	out := make([]toolShape, 0, len(shapes))
	for _, s := range shapes {
		schema, err := withInstance(s.InputSchema)
		if err != nil {
			t.Fatalf("%s: %v", s.Name, err)
		}
		var doc toolShape
		if err := json.Unmarshal(schema, &doc); err != nil {
			t.Fatalf("%s: %v", s.Name, err)
		}
		doc.Name, doc.Title, doc.Description = s.Name, s.Title, s.Description
		out = append(out, doc)
	}
	return out
}

type toolShape struct {
	Name        string
	Title       string
	Description string
	Properties  map[string]any `json:"properties"`
	Required    []string       `json:"required"`
}

// The names in here are published: the Studio's chat wiring and
// wakala-api's own tests call them by name, so a rename is a breaking
// change and is declared rather than derived.
func TestPublishedToolNamesSurvive(t *testing.T) {
	got := map[string]bool{}
	for _, tool := range declaredTools(t) {
		got[tool.Name] = true
	}
	for _, name := range []string{
		"agency_create_goal", "agency_list_goals", "agency_update_goal", "agency_delete_goal",
		"agency_set_raci", "agency_list_role_assignments_for_objective",
		"agency_attach_module", "agency_update_module_rationale", "agency_detach_module",
		"agency_publish_agency", "agency_set_publication_channel",
		"agency_lookup_event_flow_step", "agency_list_issues",
		"agency_attach_value", "agency_detach_rule",
	} {
		if !got[name] {
			t.Errorf("%s is no longer published", name)
		}
	}
}

// Every tool takes the instance it operates on, because one mount serves
// many Agencies and the manager is resolved per call.
func TestEveryToolTakesAnInstance(t *testing.T) {
	for _, tool := range declaredTools(t) {
		if _, ok := tool.Properties[instanceArg]; !ok {
			t.Errorf("%s takes no %s", tool.Name, instanceArg)
		}
		required := false
		for _, r := range tool.Required {
			if r == instanceArg {
				required = true
			}
		}
		if !required {
			t.Errorf("%s does not require %s", tool.Name, instanceArg)
		}
	}
}

// A value the store owns is not a value a model may set: offering it is an
// invitation to write something the manager will overwrite.
func TestStoreOwnedValuesAreNotArguments(t *testing.T) {
	for _, tool := range declaredTools(t) {
		for _, owned := range []string{"id", "code", "created_at", "last_updated", "deleted"} {
			if !writes(tool.Name) {
				continue
			}
			if _, offered := tool.Properties[owned]; offered {
				t.Errorf("%s offers %q, which the store owns", tool.Name, owned)
			}
		}
	}
}

// writes reports whether a tool asks the store to change something. A read
// offers a store-owned value as a filter, which is not the same as offering
// to write it.
func writes(name string) bool {
	for _, prefix := range []string{"agency_create_", "agency_update_", "agency_set_", "agency_attach_"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// Prose reaches a model as the whole of what it knows about a call.
func TestEveryToolIsDescribed(t *testing.T) {
	for _, tool := range declaredTools(t) {
		if len(strings.Fields(tool.Description)) < 5 {
			t.Errorf("%s is not described: %q", tool.Name, tool.Description)
		}
		if tool.Title == "" {
			t.Errorf("%s has no title", tool.Name)
		}
	}
}
