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

const wakalaDocument = "testdata/mwanachama-wakala.agency.json"

func distinctWorkflowObjectivePairs(doc models.AgencyDocument) map[[2]string]bool {
	pairs := map[[2]string]bool{}
	for _, wo := range doc.WorkflowObjectives {
		pairs[[2]string{wo.WorkflowID, wo.ObjectiveID}] = true
	}
	for _, wi := range doc.WorkItems {
		if wi.LegacyObjectiveID != "" {
			pairs[[2]string{wi.WorkflowID, wi.LegacyObjectiveID}] = true
		}
	}
	return pairs
}

func stepScopedAssignments(doc models.AgencyDocument) int {
	n := 0
	for _, ra := range doc.RoleAssignments {
		if ra.WorkItemID != "" {
			n++
		}
	}
	return n
}

func readDocument(t *testing.T, path string) models.AgencyDocument {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the document: %v", err)
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
		{"workflow objectives", count(t, func() (int, error) {
			v, err := mgr.ListWorkflowObjectives(ctx, draftID, mwanachamaagency.WorkflowObjectiveFilter{})
			return len(v), err
		}), len(distinctWorkflowObjectivePairs(doc))},
		{"RACI links", count(t, func() (int, error) {
			v, err := mgr.ListRoleAssignments(ctx, draftID, mwanachamaagency.RoleAssignmentFilter{})
			return len(v), err
		}), stepScopedAssignments(doc)},
		{"attachments", out.Attachments, len(doc.ValueAttachments) + len(doc.RuleAttachments) + len(doc.ControlAttachments)},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.what, tc.got, tc.want)
		}
	}
}

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
	workflows := map[string]bool{}
	workItems := map[string]bool{}
	for _, w := range listWorkflows(t, mgr, ctx, draftID) {
		workflows[w.ID] = true
	}
	items, err := mgr.ListWorkItems(ctx, draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil {
		t.Fatalf("ListWorkItems: %v", err)
	}
	for _, wi := range items {
		workItems[wi.ID] = true
	}

	assignments, err := mgr.ListRoleAssignments(ctx, draftID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignments: %v", err)
	}
	for _, a := range assignments {
		if !roles[a.RoleID] {
			t.Errorf("RACI link %s names a seat that is not in the draft", a.ID)
		}
		if !workItems[a.WorkItemID] {
			t.Errorf("RACI link %s names a step that is not in the draft", a.ID)
		}
	}

	links, err := mgr.ListWorkflowObjectives(ctx, draftID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives: %v", err)
	}
	for _, l := range links {
		if !workflows[l.WorkflowID] || !objectives[l.ObjectiveID] {
			t.Errorf("workflow objective %s names a workflow or objective that is not in the draft: %+v", l.ID, l)
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
	if second.Attachments != first.Attachments {
		t.Errorf("second import attachments = %d, want %d", second.Attachments, first.Attachments)
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

func listWorkflows(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) []models.Workflow {
	t.Helper()
	out, err := mgr.ListWorkflows(ctx, draftID, mwanachamaagency.WorkflowFilter{})
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
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

const olderShapeDocument = `{
	"agency": {"name": "Ward office"},
	"goals": [{"id": "g1", "code": "G-1", "title": "Hear residents"}],
	"objectives": [
		{"id": "o1", "code": "O-1", "goal_id": "g1", "title": "Ask residents"},
		{"id": "o2", "code": "O-2", "goal_id": "g1", "title": "Report back"}
	],
	"roles": [{"id": "r1", "code": "R-1", "name": "Clerk"}],
	"workflows": [
		{"id": "w1", "code": "W-1", "name": "Survey"},
		{"id": "w2", "code": "W-2", "name": "Newsletter"}
	],
	"work_items": [
		{"id": "s1", "code": "WI-1", "workflow_id": "w1", "objective_id": "o1", "title": "Ask", "ordinality": 1},
		{"id": "s2", "code": "WI-2", "workflow_id": "w1", "objective_id": "o1", "title": "Collect", "ordinality": 2},
		{"id": "s3", "code": "WI-3", "workflow_id": "w1", "objective_id": "o2", "title": "Summarise", "ordinality": 3},
		{"id": "s4", "code": "WI-4", "workflow_id": "w2", "objective_id": "o2", "title": "Send", "ordinality": 1}
	],
	"role_assignments": [
		{"id": "a1", "role_id": "r1", "objective_id": "o1", "label": "accountable"},
		{"id": "a2", "role_id": "r1", "objective_id": "o2", "label": "responsible"}
	],
	"some_future_section": [{"anything": true}]
}`

func TestImportDocument_OlderShapeSeedsWorkflowObjectivesFromItsSteps(t *testing.T) {
	var doc models.AgencyDocument
	if err := json.Unmarshal([]byte(olderShapeDocument), &doc); err != nil {
		t.Fatalf("an older document must still decode: %v", err)
	}
	mgr := newTestManager(t)
	ctx := context.Background()

	out, err := mgr.ImportDocument(ctx, "ward", doc)
	if err != nil {
		t.Fatalf("ImportDocument of an older document: %v", err)
	}

	links, err := mgr.ListWorkflowObjectives(ctx, out.Draft.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListWorkflowObjectives: %v", err)
	}
	if len(links) != 3 {
		t.Fatalf("expected one link per distinct (workflow, objective) pair, w1/o1 w1/o2 w2/o2, got %d: %+v", len(links), links)
	}

	workflowCode := map[string]string{}
	for _, w := range listWorkflows(t, mgr, ctx, out.Draft.ID) {
		workflowCode[w.ID] = w.Code
	}
	objectiveCode := map[string]string{}
	for _, o := range listObjectives(t, mgr, ctx, out.Draft.ID) {
		objectiveCode[o.ID] = o.Code
	}
	got := map[string]bool{}
	for _, l := range links {
		got[workflowCode[l.WorkflowID]+"/"+objectiveCode[l.ObjectiveID]] = true
	}
	for _, want := range []string{"W-1/O-1", "W-1/O-2", "W-2/O-2"} {
		if !got[want] {
			t.Errorf("missing link %s in %v", want, got)
		}
	}
}

func TestImportDocument_OlderShapeIgnoresObjectiveLevelAssignments(t *testing.T) {
	var doc models.AgencyDocument
	if err := json.Unmarshal([]byte(olderShapeDocument), &doc); err != nil {
		t.Fatalf("an older document must still decode: %v", err)
	}
	mgr := newTestManager(t)
	ctx := context.Background()

	out, err := mgr.ImportDocument(ctx, "ward", doc)
	if err != nil {
		t.Fatalf("the old objective-level assignments must be ignored without an error, got %v", err)
	}
	assignments, err := mgr.ListRoleAssignments(ctx, out.Draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignments: %v", err)
	}
	if len(assignments) != 0 {
		t.Fatalf("expected the objective-level assignments dropped, got %+v", assignments)
	}
	if len(listWorkflows(t, mgr, ctx, out.Draft.ID)) != 2 || len(listObjectives(t, mgr, ctx, out.Draft.ID)) != 2 {
		t.Fatal("the rest of the older document must still import")
	}
}

func TestImportDocument_OlderShapeScansClean(t *testing.T) {
	var doc models.AgencyDocument
	if err := json.Unmarshal([]byte(olderShapeDocument), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	mgr := newTestManager(t)
	ctx := context.Background()
	out, err := mgr.ImportDocument(ctx, "ward", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	issues, err := mgr.ListIssues(ctx, out.Draft.ID, mwanachamaagency.IssueFilter{Severity: string(models.IssueSeverityError)})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	for _, i := range issues {
		t.Errorf("an older document with objectives on its steps should scan clean once seeded: %s", i.Message)
	}
}

func TestImportDocument_NewShapeCarriesBothJoins(t *testing.T) {
	raw := `{
		"agency": {"name": "Ward office"},
		"goals": [{"id": "g1", "code": "G-1", "title": "Hear residents"}],
		"objectives": [{"id": "o1", "code": "O-1", "goal_id": "g1", "title": "Ask residents"}],
		"roles": [{"id": "r1", "code": "R-1", "name": "Clerk"}, {"id": "r2", "code": "R-2", "name": "Mayor"}],
		"workflows": [{"id": "w1", "code": "W-1", "name": "Survey"}],
		"workflow_objectives": [{"id": "wo1", "workflow_id": "w1", "objective_id": "o1"}],
		"work_items": [
			{"id": "s1", "code": "WI-1", "workflow_id": "w1", "title": "Ask", "ordinality": 1},
			{"id": "s2", "code": "WI-2", "workflow_id": "w1", "title": "Collect", "ordinality": 2}
		],
		"role_assignments": [
			{"id": "a1", "work_item_id": "s1", "role_id": "r1", "label": "responsible"},
			{"id": "a2", "work_item_id": "s1", "role_id": "r2", "label": "accountable"},
			{"id": "a3", "work_item_id": "s2", "role_id": "r2", "label": "accountable"},
			{"id": "a4", "work_item_id": "s2", "role_id": "r1", "label": "accountable"}
		]
	}`
	var doc models.AgencyDocument
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	mgr := newTestManager(t)
	ctx := context.Background()
	out, err := mgr.ImportDocument(ctx, "ward", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	links, err := mgr.ListWorkflowObjectives(ctx, out.Draft.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil || len(links) != 1 {
		t.Fatalf("ListWorkflowObjectives: %v, %+v", err, links)
	}
	assignments, err := mgr.ListRoleAssignments(ctx, out.Draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil || len(assignments) != 4 {
		t.Fatalf("expected all four step-scoped assignments, two Accountables on one step included, got %v, %+v", err, assignments)
	}
	issues, err := mgr.ListIssues(ctx, out.Draft.ID, mwanachamaagency.IssueFilter{Severity: string(models.IssueSeverityError)})
	if err != nil || len(issues) != 0 {
		t.Fatalf("expected a clean scan, got %v, %+v", err, issues)
	}
}

func TestImportDocument_PublishedDocumentRoundTrips(t *testing.T) {
	ctx := context.Background()
	source := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, source, ctx)
	pub, err := source.PublishAgency(ctx, o.DraftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	target := newTestManager(t)
	out, err := target.ImportDocument(ctx, "copy", pub.Document)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	links, err := target.ListWorkflowObjectives(ctx, out.Draft.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil || len(links) != len(pub.Document.WorkflowObjectives) || len(links) != 1 {
		t.Fatalf("ListWorkflowObjectives: %v, %+v", err, links)
	}
	assignments, err := target.ListRoleAssignments(ctx, out.Draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil || len(assignments) != len(pub.Document.RoleAssignments) || len(assignments) != 1 {
		t.Fatalf("ListRoleAssignments: %v, %+v", err, assignments)
	}
	if _, err := target.PublishAgency(ctx, out.Draft.ID, "", ""); err != nil {
		t.Fatalf("the imported copy should publish: %v", err)
	}
}
