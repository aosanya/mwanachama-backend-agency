package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func declaredTools(t *testing.T) []toolShape {
	t.Helper()
	shapes, err := toolsFor(mwanachamaagency.ManagerShape(), nil)
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

func TestPublishedToolNamesSurvive(t *testing.T) {
	got := map[string]bool{}
	for _, tool := range declaredTools(t) {
		got[tool.Name] = true
	}
	for _, name := range []string{
		"agency_create_goal", "agency_list_goals", "agency_update_goal", "agency_delete_goal",
		"agency_link_workflow_objective", "agency_list_workflow_objectives",
		"agency_get_workflow_objective", "agency_delete_workflow_objective",
		"agency_set_module_enabled", "agency_set_module_mounts",
		"agency_publish_agency", "agency_set_publication_channel",
		"agency_lookup_event_flow_step", "agency_list_issues",
		"agency_attach_value", "agency_detach_rule",
		"agency_set_measure", "agency_list_measures",
		"agency_start_review", "agency_get_review", "agency_list_reviews",
		"agency_set_finding_status", "agency_comment_finding",
		"agency_review_input", "agency_record_review_area",
	} {
		if !got[name] {
			t.Errorf("%s is no longer published", name)
		}
	}
}

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

func writes(name string) bool {
	for _, prefix := range []string{"agency_create_", "agency_update_", "agency_set_", "agency_attach_", "agency_link_"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

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

func TestSetMeasureOffersNoCurrentReading(t *testing.T) {
	for _, tool := range declaredTools(t) {
		if tool.Name != "agency_set_measure" {
			continue
		}
		for _, owned := range []string{"current", "draft_id"} {
			if _, offered := tool.Properties[owned]; offered {
				t.Errorf("agency_set_measure offers %s, which the manager does not set through it", owned)
			}
		}
		for _, want := range []string{"objective_id", "name", "target", "unit", "status"} {
			if _, ok := tool.Properties[want]; !ok {
				t.Errorf("agency_set_measure offers no %s", want)
			}
		}
		return
	}
	t.Fatal("agency_set_measure is not declared")
}

func toolNamed(t *testing.T, name string) toolShape {
	t.Helper()
	for _, tool := range declaredTools(t) {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("%s is not declared", name)
	return toolShape{}
}

func requires(tool toolShape, arg string) bool {
	for _, r := range tool.Required {
		if r == arg {
			return true
		}
	}
	return false
}

func TestTheObjectiveListOfAStepIsGone(t *testing.T) {
	for _, tool := range declaredTools(t) {
		if tool.Name == "agency_list_role_assignments_for_objective" {
			t.Errorf("%s is still published, but a seat is on a step and no longer on an objective", tool.Name)
		}
	}
}

func TestTheRoleAssignmentToolsAreGone(t *testing.T) {
	gone := map[string]bool{
		"agency_set_raci": true, "agency_list_role_assignments": true, "agency_get_role_assignment": true,
		"agency_update_role_assignment": true, "agency_delete_role_assignment": true,
		"agency_list_role_assignments_for_work_item": true,
	}
	for _, tool := range declaredTools(t) {
		if gone[tool.Name] {
			t.Errorf("%s is still published", tool.Name)
		}
	}
}

func TestStepToolsOfferNoObjective(t *testing.T) {
	for _, name := range []string{"agency_create_work_item", "agency_update_work_item"} {
		if _, offered := toolNamed(t, name).Properties["objective_id"]; offered {
			t.Errorf("%s still offers objective_id", name)
		}
	}
}

func TestStepToolsOfferTheAccountableRole(t *testing.T) {
	for _, name := range []string{"agency_create_work_item", "agency_update_work_item"} {
		if _, offered := toolNamed(t, name).Properties["accountable_role_id"]; !offered {
			t.Errorf("%s offers no accountable_role_id", name)
		}
	}
}

func TestWorkflowObjectiveToolsTakeBothEnds(t *testing.T) {
	link := toolNamed(t, "agency_link_workflow_objective")
	for _, want := range []string{"workflow_id", "objective_id", "draft_id"} {
		if _, ok := link.Properties[want]; !ok {
			t.Errorf("agency_link_workflow_objective offers no %s", want)
		}
	}
	for _, want := range []string{"workflow_id", "objective_id"} {
		if !requires(link, want) {
			t.Errorf("agency_link_workflow_objective does not require %s", want)
		}
	}
	list := toolNamed(t, "agency_list_workflow_objectives")
	for _, want := range []string{"workflow_id", "objective_id", "draft_id"} {
		if _, ok := list.Properties[want]; !ok {
			t.Errorf("agency_list_workflow_objectives offers no %s", want)
		}
	}
	for _, name := range []string{"agency_get_workflow_objective", "agency_delete_workflow_objective"} {
		if !requires(toolNamed(t, name), "workflow_objective_id") {
			t.Errorf("%s does not require workflow_objective_id", name)
		}
	}
}
