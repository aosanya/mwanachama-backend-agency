package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const wakalaDocument = "../developer/documentation/4. qa/mwanachama-wakala/agency.json"

func readDocument(t *testing.T, path string) models.AgencyDocument {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("the document is not here: %v", err)
	}
	var doc models.AgencyDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("read %s: %v", filepath.Base(path), err)
	}
	return doc
}

func TestImportDocument_MwanachamaWakala(t *testing.T) {
	doc := readDocument(t, wakalaDocument)
	mgr := newTestManager(t)
	ctx := context.Background()

	out, err := mgr.ImportDocument(ctx, "Mwanachama-Wakala", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if out.Draft.ID == "" {
		t.Fatal("the import produced no draft")
	}

	agency, err := mgr.GetAgency(ctx)
	if err != nil {
		t.Fatalf("GetAgency: %v", err)
	}
	if agency.Name != doc.Agency.Name {
		t.Errorf("agency = %q, want %q", agency.Name, doc.Agency.Name)
	}

	draftID := out.Draft.ID
	for _, tc := range []struct {
		what string
		got  int
		want int
	}{
		{"roles", count(t, func() (int, error) {
			v, err := mgr.ListRoles(ctx, draftID, mwanachamaagency.RoleFilter{})
			return len(v), err
		}), len(doc.Roles)},
		{"goals", count(t, func() (int, error) {
			v, err := mgr.ListGoals(ctx, draftID, mwanachamaagency.GoalFilter{})
			return len(v), err
		}), len(doc.Goals)},
		{"objectives", count(t, func() (int, error) {
			v, err := mgr.ListObjectives(ctx, draftID, mwanachamaagency.ObjectiveFilter{})
			return len(v), err
		}), len(doc.Objectives)},
		{"workflows", count(t, func() (int, error) {
			v, err := mgr.ListWorkflows(ctx, draftID, mwanachamaagency.WorkflowFilter{})
			return len(v), err
		}), len(doc.Workflows)},
		{"work items", count(t, func() (int, error) {
			v, err := mgr.ListWorkItems(ctx, draftID, mwanachamaagency.WorkItemFilter{})
			return len(v), err
		}), len(doc.WorkItems)},
		{"RACI links", count(t, func() (int, error) {
			v, err := mgr.ListRoleAssignments(ctx, draftID, mwanachamaagency.RoleAssignmentFilter{})
			return len(v), err
		}), len(doc.RoleAssignments)},
		{"attachments", out.Attachments, len(doc.ValueAttachments) + len(doc.RuleAttachments)},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.what, tc.got, tc.want)
		}
	}
}

// Every RACI link has to point at a seat and an objective that exist in the
// draft it landed in — the ids in the document are the exporter's, and the
// import mints its own.
func TestImportDocument_ReferencesResolve(t *testing.T) {
	doc := readDocument(t, wakalaDocument)
	mgr := newTestManager(t)
	ctx := context.Background()

	out, err := mgr.ImportDocument(ctx, "Mwanachama-Wakala", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	draftID := out.Draft.ID

	roles := map[string]bool{}
	for _, r := range listRoles(t, mgr, ctx, draftID) {
		roles[r.ID] = true
	}
	objectives := map[string]bool{}
	for _, o := range listObjectives(t, mgr, ctx, draftID) {
		objectives[o.ID] = true
	}

	assignments, err := mgr.ListRoleAssignments(ctx, draftID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignments: %v", err)
	}
	for _, a := range assignments {
		if !roles[a.RoleID] {
			t.Errorf("RACI link %s names a seat that is not in the draft", a.ID)
		}
		if !objectives[a.ObjectiveID] {
			t.Errorf("RACI link %s names an objective that is not in the draft", a.ID)
		}
	}

	issues, err := mgr.ListIssues(ctx, draftID, mwanachamaagency.IssueFilter{Severity: string(models.IssueSeverityError)})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, issue := range issues {
		t.Errorf("the imported draft does not validate: %s", issue.Message)
	}
}

// Importing the same document twice leaves the first draft alone and does
// not attach anything a second time.
func TestImportDocument_Twice(t *testing.T) {
	doc := readDocument(t, wakalaDocument)
	mgr := newTestManager(t)
	ctx := context.Background()

	first, err := mgr.ImportDocument(ctx, "first", doc)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	second, err := mgr.ImportDocument(ctx, "second", doc)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}

	if first.Draft.ID == second.Draft.ID {
		t.Fatal("the second import reused the first draft")
	}
	if second.Attachments != 0 {
		t.Errorf("attachments written again = %d, want none", second.Attachments)
	}
}

func count(t *testing.T, read func() (int, error)) int {
	t.Helper()
	n, err := read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return n
}

func listRoles(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) []models.Role {
	t.Helper()
	out, err := mgr.ListRoles(ctx, draftID, mwanachamaagency.RoleFilter{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	return out
}

func listObjectives(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) []models.Objective {
	t.Helper()
	out, err := mgr.ListObjectives(ctx, draftID, mwanachamaagency.ObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListObjectives: %v", err)
	}
	return out
}

// An imported row keeps the code it was exported with, so the counter has
// to move past it: the next goal created in that draft must not be handed a
// code the import already used.
func TestImportDocument_NextCodeDoesNotCollide(t *testing.T) {
	doc := readDocument(t, wakalaDocument)
	mgr := newTestManager(t)
	ctx := context.Background()

	out, err := mgr.ImportDocument(ctx, "Mwanachama-Wakala", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	held := map[string]bool{}
	for _, g := range listGoals(t, mgr, ctx, out.Draft.ID) {
		held[g.Code] = true
	}

	next, err := mgr.CreateGoal(ctx, models.Goal{DraftID: out.Draft.ID, Title: "One more"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if held[next.Code] {
		t.Errorf("the new goal was handed %q, which the import already used", next.Code)
	}
}

func listGoals(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) []models.Goal {
	t.Helper()
	out, err := mgr.ListGoals(ctx, draftID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	return out
}
