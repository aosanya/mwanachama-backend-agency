package mcp_test

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func strings1(v any) []string {
	raw, _ := v.([]any)
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.(string))
	}
	return out
}

func TestDesignFromSpec_CreatesAWorkflowObjectiveRowPerObjectiveAWorkflowServes(t *testing.T) {
	d := newStepDesign(t)

	sc := d.mustCall(t, "agency_design_from_spec", map[string]any{
		"draft_id": d.draft.ID,
		"goals":    []any{map[string]any{"key": "g", "title": "Grow"}},
		"objectives": []any{
			map[string]any{"key": "win", "goal_key": "g", "title": "Win customers"},
			map[string]any{"key": "keep", "goal_key": "g", "title": "Keep customers"},
		},
		"workflows": []any{
			map[string]any{"key": "sell", "name": "Sell", "objective_keys": []any{"win", "keep"}},
			map[string]any{"key": "support", "name": "Support", "objective_keys": []any{"keep"}, "objective_ids": []any{d.objectiveA.ID}},
			map[string]any{"key": "idle", "name": "Idle"},
		},
		"work_items": []any{
			map[string]any{"key": "quote", "workflow_key": "sell", "title": "Quote"},
			map[string]any{"key": "answer", "workflow_key": "support", "title": "Answer"},
		},
	})
	if errs, _ := sc["errors"].([]any); len(errs) != 0 {
		t.Fatalf("agency_design_from_spec errors: %+v", errs)
	}
	if links := strings1(sc["workflow_objectives"]); len(links) != 4 {
		t.Fatalf("workflow_objectives = %+v, want 4 created link ids (sell:2, support:2, idle:0)", sc["workflow_objectives"])
	}

	workflows, _ := sc["workflows"].(map[string]any)
	objectives, _ := sc["objectives"].(map[string]any)
	stored, err := d.mgr.ListWorkflowObjectives(ctxT(t), d.draft.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives: %v", err)
	}
	var got []string
	for _, l := range stored {
		got = append(got, l.WorkflowID+">"+l.ObjectiveID)
	}
	want := []string{
		workflows["sell"].(string) + ">" + objectives["win"].(string),
		workflows["sell"].(string) + ">" + objectives["keep"].(string),
		workflows["support"].(string) + ">" + objectives["keep"].(string),
		workflows["support"].(string) + ">" + d.objectiveA.ID,
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("stored workflow objective links\n got %v\nwant %v", got, want)
	}
	for _, l := range stored {
		if l.DraftID != d.draft.ID {
			t.Errorf("link %+v is outside the spec's draft", l)
		}
		if l.WorkflowID == workflows["idle"] {
			t.Errorf("a workflow that named no objective was linked: %+v", l)
		}
	}
}

func TestDesignFromSpec_ANamedObjectiveThatIsNotInTheSpecIsReportedAndTheWorkflowStillExists(t *testing.T) {
	d := newStepDesign(t)

	sc := d.mustCall(t, "agency_design_from_spec", map[string]any{
		"draft_id": d.draft.ID,
		"workflows": []any{
			map[string]any{"key": "w", "name": "Lonely", "objective_keys": []any{"nowhere"}, "objective_ids": []any{"no-such-objective"}},
		},
	})
	errs, _ := sc["errors"].([]any)
	if len(errs) != 2 {
		t.Fatalf("errors = %+v, want one for the unknown objective_key and one for the unknown objective id", errs)
	}
	if !strings.Contains(errs[0].(string)+errs[1].(string), `"nowhere"`) {
		t.Errorf("no error names the unknown objective_key: %+v", errs)
	}
	if workflows, _ := sc["workflows"].(map[string]any); workflows["w"] == nil {
		t.Errorf("the workflow was not created: %+v", sc["workflows"])
	}
	if links, _ := d.mgr.ListWorkflowObjectives(ctxT(t), d.draft.ID, mwanachamaagency.WorkflowObjectiveFilter{}); len(links) != 0 {
		t.Errorf("links were stored for objectives that do not exist: %+v", links)
	}
}

func TestDesignFromSpec_RaciNamesAStepByKeyOrId(t *testing.T) {
	d := newStepDesign(t)

	sc := d.mustCall(t, "agency_design_from_spec", map[string]any{
		"draft_id": d.draft.ID,
		"workflows": []any{
			map[string]any{"key": "w", "name": "Handle", "objective_ids": []any{d.objectiveA.ID}},
		},
		"work_items": []any{map[string]any{"key": "s1", "workflow_key": "w", "title": "Triage"}},
		"raci": []any{
			map[string]any{"role_id": d.owner.ID, "work_item_key": "s1", "label": "accountable"},
			map[string]any{"role_id": d.helper.ID, "work_item_key": "s1", "label": "accountable"},
			map[string]any{"role_id": d.owner.ID, "work_item_key": "s1", "label": "responsible"},
			map[string]any{"role_id": d.helper.ID, "work_item_id": d.step.ID, "label": "informed"},
		},
	})
	if errs, _ := sc["errors"].([]any); len(errs) != 0 {
		t.Fatalf("a step may have any number of accountable seats, got errors %+v", errs)
	}
	if ids := strings1(sc["role_assignments"]); len(ids) != 4 {
		t.Fatalf("role_assignments = %+v, want 4", sc["role_assignments"])
	}

	created, _ := sc["work_items"].(map[string]any)
	triage, _ := created["s1"].(string)
	seats, err := d.mgr.ListRoleAssignmentsForWorkItem(ctxT(t), triage, d.draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil || len(seats) != 3 {
		t.Fatalf("the spec's step holds %+v err=%v, want 3 seats", seats, err)
	}
	existing, err := d.mgr.ListRoleAssignmentsForWorkItem(ctxT(t), d.step.ID, d.draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil || len(existing) != 1 || existing[0].Label != models.RACILabelInformed {
		t.Errorf("the step that already existed holds %+v err=%v, want the one informed seat", existing, err)
	}
}

func TestDesignFromSpec_RaciRefusesOnlyTheSameSeatAndLabelOnAStepTwice(t *testing.T) {
	d := newStepDesign(t)

	sc := d.mustCall(t, "agency_design_from_spec", map[string]any{
		"draft_id": d.draft.ID,
		"raci": []any{
			map[string]any{"role_id": d.owner.ID, "work_item_id": d.step.ID, "label": "accountable"},
			map[string]any{"role_id": d.owner.ID, "work_item_id": d.step.ID, "label": "accountable"},
			map[string]any{"role_id": d.owner.ID, "work_item_id": d.step.ID, "label": "consulted"},
		},
	})
	if errs, _ := sc["errors"].([]any); len(errs) != 1 {
		t.Errorf("errors = %+v, want exactly the repeated seat refused", sc["errors"])
	}
	if ids := strings1(sc["role_assignments"]); len(ids) != 2 {
		t.Errorf("role_assignments = %+v, want the first accountable and the consulted", sc["role_assignments"])
	}
	if seats, _ := d.mgr.ListRoleAssignmentsForWorkItem(ctxT(t), d.step.ID, d.draft.ID, mwanachamaagency.RoleAssignmentFilter{}); len(seats) != 2 {
		t.Errorf("the step holds %+v, want 2 seats", seats)
	}
}

func TestDesignFromSpec_RaciNeedsAStepAndASeat(t *testing.T) {
	d := newStepDesign(t)

	sc := d.mustCall(t, "agency_design_from_spec", map[string]any{
		"draft_id": d.draft.ID,
		"raci": []any{
			map[string]any{"role_id": d.owner.ID, "label": "accountable"},
			map[string]any{"role_id": d.owner.ID, "work_item_key": "never-made", "label": "accountable"},
			map[string]any{"work_item_id": d.step.ID, "label": "accountable"},
			map[string]any{"role_key": "never-made", "work_item_id": d.step.ID, "label": "accountable"},
		},
	})
	errs, _ := sc["errors"].([]any)
	if len(errs) != 4 {
		t.Fatalf("errors = %+v, want each of the four incomplete raci entries refused", errs)
	}
	joined := ""
	for _, e := range errs {
		joined += e.(string) + "\n"
	}
	for _, want := range []string{"work_item_key or work_item_id", `unknown work_item_key "never-made"`, "role_key or role_id", `unknown role_key "never-made"`} {
		if !strings.Contains(joined, want) {
			t.Errorf("errors do not say %q:\n%s", want, joined)
		}
	}
	if seats, _ := d.mgr.ListRoleAssignments(ctxT(t), d.draft.ID, mwanachamaagency.RoleAssignmentFilter{}); len(seats) != 0 {
		t.Errorf("incomplete raci entries were stored: %+v", seats)
	}
}

func TestDesignFromSpec_TheSchemaPutsObjectivesOnAWorkflowAndRaciOnAStep(t *testing.T) {
	d := newStepDesign(t)
	tools, err := d.cs.ListTools(ctxT(t), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	var schema any
	for _, tl := range tools.Tools {
		if tl.Name == "agency_design_from_spec" {
			schema = tl.InputSchema
		}
	}
	if schema == nil {
		t.Fatal("agency_design_from_spec is not listed")
	}
	raw, _ := json.Marshal(schema)
	var doc struct {
		Properties map[string]struct {
			Items struct {
				Properties map[string]any `json:"properties"`
			} `json:"items"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("schema: %v", err)
	}
	has := func(section, field string) bool {
		_, ok := doc.Properties[section].Items.Properties[field]
		return ok
	}
	for section, fields := range map[string][]string{
		"workflows": {"objective_keys", "objective_ids"},
		"raci":      {"work_item_key", "work_item_id", "role_key", "role_id", "label"},
	} {
		for _, f := range fields {
			if !has(section, f) {
				t.Errorf("%s entries offer no %s", section, f)
			}
		}
	}
	if has("work_items", "objective_key") || has("work_items", "objective_id") {
		t.Errorf("work_items entries still offer an objective: %+v", doc.Properties["work_items"].Items.Properties)
	}
	if has("raci", "objective_id") || has("raci", "objective_key") {
		t.Errorf("raci entries still name an objective: %+v", doc.Properties["raci"].Items.Properties)
	}
}
