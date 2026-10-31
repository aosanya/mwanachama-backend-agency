package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func documentWithLabels(t *testing.T, labels ...string) models.AgencyDocument {
	t.Helper()
	assignments := make([]map[string]string, 0, len(labels))
	for _, l := range labels {
		assignments = append(assignments, map[string]string{"work_item_id": "s1", "role_id": "r1", "label": l})
	}
	raw, err := json.Marshal(map[string]any{
		"roles":               []map[string]string{{"id": "r1", "code": "R-1", "name": "A"}},
		"goals":               []map[string]string{{"id": "g1", "code": "G-1", "title": "G"}},
		"objectives":          []map[string]string{{"id": "o1", "code": "O-1", "goal_id": "g1", "title": "O"}},
		"workflows":           []map[string]string{{"id": "w1", "code": "W-1", "name": "W"}},
		"workflow_objectives": []map[string]string{{"id": "x", "workflow_id": "w1", "objective_id": "o1"}},
		"work_items":          []map[string]string{{"id": "s1", "code": "WI-1", "workflow_id": "w1", "title": "S"}},
		"role_assignments":    assignments,
	})
	if err != nil {
		t.Fatal(err)
	}
	var doc models.AgencyDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc.Agency.Name = "Imp"
	return doc
}

func TestImportDocument_SkipsARoleAssignmentWhoseLabelIsNotARACILabel(t *testing.T) {
	for _, label := range []string{"bogus", "", "RESPONSIBLE", " responsible"} {
		mgr := newTestManager(t)
		ctx := context.Background()
		res, err := mgr.ImportDocument(ctx, "imp", documentWithLabels(t, label))
		if err != nil {
			t.Fatalf("label %q: ImportDocument: %v", label, err)
		}
		got, err := mgr.ListRoleAssignments(ctx, res.Draft.ID, mwanachamaagency.RoleAssignmentFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("label %q imported and stored as %+v", label, got)
		}
	}
}

func TestImportDocument_AWrongCaseLabelDoesNotBeatTheDuplicateRefusal(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	res, err := mgr.ImportDocument(ctx, "imp", documentWithLabels(t, "responsible", "RESPONSIBLE"))
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	got, err := mgr.ListRoleAssignments(ctx, res.Draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Label != models.RACILabelResponsible {
		t.Errorf("expected the one valid responsible assignment, got %+v", got)
	}
}

func TestListIssues_RoleAssignmentWithAnInvalidLabelBlocks(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	d, _, _ := seedDraftForIssues(t, mgr, ctx)
	items, err := mgr.ListWorkItems(ctx, d.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil || len(items) != 1 {
		t.Fatalf("ListWorkItems: %v, %+v", err, items)
	}
	role, err := mgr.CreateRole(ctx, models.Role{Name: "owner", DraftID: d.ID})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	a, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		RoleID: role.ID, WorkItemID: items[0].ID, Label: models.RACILabelResponsible, DraftID: d.ID,
	})
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	if found := issuesFor(t, mgr, ctx, d.ID, "role_assignment"); len(found) != 0 {
		t.Fatalf("a valid label raised %+v", found)
	}

	table := tableOf(t, s, "role_assignment")
	if err := db.Exec("update "+table+" set label = ? where id = ?", "RESPONSIBLE", a.ID).Error; err != nil {
		t.Fatalf("storing a bad label: %v", err)
	}
	found := issuesFor(t, mgr, ctx, d.ID, "role_assignment")
	if len(found) != 1 || found[0].EntityID != a.ID || found[0].Severity != models.IssueSeverityError {
		t.Fatalf("expected one error for the invalid label, got %+v", found)
	}
	if !strings.Contains(found[0].Message, "RESPONSIBLE") {
		t.Errorf("expected the message to name the label, got %q", found[0].Message)
	}
}
