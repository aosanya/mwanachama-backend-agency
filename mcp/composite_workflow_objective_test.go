package mcp_test

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
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

func TestDesignFromSpec_AStepNamesItsAccountableRoleByKeyOrId(t *testing.T) {
	d := newStepDesign(t)

	sc := d.mustCall(t, "agency_design_from_spec", map[string]any{
		"draft_id": d.draft.ID,
		"roles":    []any{map[string]any{"key": "lead", "name": "Lead"}},
		"workflows": []any{
			map[string]any{"key": "w", "name": "Handle", "objective_ids": []any{d.objectiveA.ID}},
		},
		"work_items": []any{
			map[string]any{"key": "s1", "workflow_key": "w", "title": "Triage", "accountable_role_key": "lead"},
			map[string]any{"key": "s2", "workflow_key": "w", "title": "Close", "accountable_role_id": d.owner.ID},
			map[string]any{"key": "s3", "workflow_key": "w", "title": "Archive"},
		},
	})
	if errs, _ := sc["errors"].([]any); len(errs) != 0 {
		t.Fatalf("got errors %+v", errs)
	}
	roles, _ := sc["roles"].(map[string]any)
	created, _ := sc["work_items"].(map[string]any)
	want := map[string]string{"s1": roles["lead"].(string), "s2": d.owner.ID, "s3": ""}
	for key, role := range want {
		id, _ := created[key].(string)
		got, err := d.mgr.GetWorkItem(ctxT(t), id)
		if err != nil || got.AccountableRoleID != role {
			t.Errorf("step %s accountable role = %q err=%v, want %q", key, got.AccountableRoleID, err, role)
		}
	}
}

func TestDesignFromSpec_AnAccountableRoleNeedsOneKnownRole(t *testing.T) {
	d := newStepDesign(t)

	sc := d.mustCall(t, "agency_design_from_spec", map[string]any{
		"draft_id": d.draft.ID,
		"workflows": []any{
			map[string]any{"key": "w", "name": "Handle", "objective_ids": []any{d.objectiveA.ID}},
		},
		"work_items": []any{
			map[string]any{"key": "both", "workflow_key": "w", "title": "Both", "accountable_role_key": "x", "accountable_role_id": d.owner.ID},
			map[string]any{"key": "unknown", "workflow_key": "w", "title": "Unknown", "accountable_role_key": "never-made"},
			map[string]any{"key": "ghost", "workflow_key": "w", "title": "Ghost", "accountable_role_id": "no-such-role"},
		},
	})
	errs, _ := sc["errors"].([]any)
	if len(errs) != 3 {
		t.Fatalf("errors = %+v, want each of the three refused", errs)
	}
	joined := ""
	for _, e := range errs {
		joined += e.(string) + "\n"
	}
	for _, want := range []string{"accountable_role_key or accountable_role_id", `unknown accountable_role_key "never-made"`} {
		if !strings.Contains(joined, want) {
			t.Errorf("errors do not say %q:\n%s", want, joined)
		}
	}
	if created, _ := sc["work_items"].(map[string]any); len(created) != 0 {
		t.Errorf("refused steps were created: %+v", created)
	}
}

func TestDesignFromSpec_TheSchemaPutsObjectivesOnAWorkflowAndTheAccountableRoleOnAStep(t *testing.T) {
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
		"workflows":  {"objective_keys", "objective_ids"},
		"work_items": {"accountable_role_key", "accountable_role_id"},
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
	if _, ok := doc.Properties["raci"]; ok {
		t.Errorf("the spec still offers a raci section")
	}
}
