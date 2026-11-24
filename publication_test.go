package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func onlyStep(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) models.WorkItem {
	t.Helper()
	items, err := mgr.ListWorkItems(ctx, draftID, mwanachamaagency.WorkItemFilter{})
	if err != nil || len(items) != 1 {
		t.Fatalf("ListWorkItems(%s): %v, %+v", draftID, err, items)
	}
	return items[0]
}

func seedPublishableAgency(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) (models.Agency, models.Objective, models.Role) {
	t.Helper()
	ag, _, o, _, wi := seedLiveAgency(t, mgr, ctx)
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	wi.AccountableRoleID = role.ID
	if _, err := mgr.UpdateWorkItem(ctx, wi); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	return ag, o, role
}

func TestPublishAgency_NoDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	p1, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if p1.Version != 1 || p1.Tag != "v1" || p1.Channel != models.DefaultChannel || p1.ContentHash == "" {
		t.Fatalf("unexpected first publication: %+v", p1)
	}

	if _, err := mgr.PublishAgency(ctx, draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("expected ErrNoChangesDetected republishing unchanged content, got %v", err)
	}

	if _, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: "correctness over speed"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	p2, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency after change: %v", err)
	}
	if p2.Version != 2 || p2.Tag != "v2" {
		t.Fatalf("expected version 2, got %+v", p2)
	}

	gotP1, err := mgr.GetPublication(ctx, 1)
	if err != nil {
		t.Fatalf("GetPublication(1): %v", err)
	}
	if gotP1.Channel != models.ChannelArchived {
		t.Fatalf("expected publishing v2 to archive v1, got channel %q", gotP1.Channel)
	}
}

func TestPublishAgency_AccountableRoleOnlyChangeIsAChange(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	p1, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	successor, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	step := onlyStep(t, mgr, ctx, draftID)
	step.AccountableRoleID = successor.ID
	if _, err := mgr.UpdateWorkItem(ctx, step); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}

	p2, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("expected an accountable-role-only change to require republishing, got %v", err)
	}
	if p2.Version != p1.Version+1 {
		t.Fatalf("expected version %d, got %d", p1.Version+1, p2.Version)
	}
}

func TestPublishAgency_ModuleOnlyChangeIsAChange(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	p1, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	git := moduleByCode(t, mgr, ctx, draftID, "git")
	if _, err := mgr.SetModuleEnabled(ctx, draftID, git.ID, true); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}

	p2, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("expected a Module-only change to require republishing, got %v", err)
	}
	if p2.Version != p1.Version+1 {
		t.Fatalf("expected version %d, got %d", p1.Version+1, p2.Version)
	}
}

func TestPublishAgency_ModuleSpecOnlyChangeIsAChange(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	mod, err := mgr.CreateModule(ctx, models.Module{DraftID: draftID, Code: "widgets", Name: "Widgets", Enabled: true})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	p1, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	activated := `{"module":"widgets","domain":"agency","objects":[{"role":"widget","name":"widget","table":"widgets"}]}`
	if err := db.Exec("update "+tableOf(t, s, "module")+" set spec = ? where id = ?", activated, mod.ID).Error; err != nil {
		t.Fatalf("activating the module's spec: %v", err)
	}

	p2, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("expected a module-spec-only change to require republishing, got %v", err)
	}
	if p2.Version != p1.Version+1 {
		t.Fatalf("expected version %d, got %d", p1.Version+1, p2.Version)
	}

	var published []models.DocumentModule
	for _, dm := range p2.Document.Modules {
		if dm.Code == "widgets" {
			published = append(published, dm)
		}
	}
	if len(published) != 1 {
		t.Fatalf("expected one widgets module in the published document, got %d", len(published))
	}
	if string(published[0].Spec) != activated {
		t.Fatalf("expected the published document to carry the activated spec, got %q", string(published[0].Spec))
	}
}

func TestPublishAgency_UnknownDraftRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedPublishableAgency(t, mgr, ctx)

	if _, err := mgr.PublishAgency(ctx, "no-such-draft", "", ""); !errors.Is(err, mwanachamaagency.ErrDraftNotFound) {
		t.Fatalf("expected ErrDraftNotFound for an unknown draftID, got %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, "", "", ""); !errors.Is(err, mwanachamaagency.ErrDraftNotFound) {
		t.Fatalf("expected ErrDraftNotFound for an empty draftID, got %v", err)
	}
}

func TestPublishAgency_ArchivedDraftAllowed(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	if _, err := mgr.ArchiveDraft(ctx, draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}
	p, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("expected publishing an archived draft to succeed, got %v", err)
	}
	if p.DraftID != draftID {
		t.Fatalf("expected publication to record DraftID %q, got %q", draftID, p.DraftID)
	}

	gotDraft, err := mgr.GetDraft(ctx, draftID)
	if err != nil {
		t.Fatalf("GetDraft: %v", err)
	}
	if gotDraft.Status != models.DraftStatusArchived {
		t.Fatalf("expected publish to leave the draft's own status unchanged, got %q", gotDraft.Status)
	}
}

func TestPublishAgency_RefusedOnValidationError(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := o.DraftID
	w := addUnlinkedWorkflow(t, mgr, ctx, draftID)

	_, err := mgr.PublishAgency(ctx, draftID, "", "")
	if !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("expected ErrPublishValidationFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "no Objective") {
		t.Errorf("expected the error to name what is wrong, got %q", err)
	}

	pubs, err := mgr.ListPublications(ctx, mwanachamaagency.PublicationFilter{})
	if err != nil {
		t.Fatalf("ListPublications: %v", err)
	}
	if len(pubs) != 0 {
		t.Fatalf("a refused publish must not write a publication, got %+v", pubs)
	}

	issues, err := mgr.ListIssues(ctx, draftID, mwanachamaagency.IssueFilter{Severity: "error"})
	if err != nil {
		t.Fatalf("ListIssues(draft, error): %v", err)
	}
	if len(issues) != 1 || issues[0].EntityID != w.ID {
		t.Fatalf("expected the same single finding for workflow %s, got %+v", w.ID, issues)
	}

	if _, err := mgr.CreateWorkflowObjective(ctx, models.WorkflowObjective{DraftID: draftID, WorkflowID: w.ID, ObjectiveID: o.ID}); err != nil {
		t.Fatalf("CreateWorkflowObjective: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "", ""); err != nil {
		t.Fatalf("expected the publish to succeed once fixed, got %v", err)
	}
}

func TestPublishAgency_NoAccountableDoesNotBlock(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	pub, err := mgr.PublishAgency(ctx, o.DraftID, "", "")
	if err != nil {
		t.Fatalf("a step with no RACI at all must publish, got %v", err)
	}
	if len(pub.Document.RoleAssignments) != 0 {
		t.Fatalf("expected no role assignments in the document, got %+v", pub.Document.RoleAssignments)
	}
}

func TestPublishAgency_TheAccountableRoleIsInTheDocument(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	step := onlyStep(t, mgr, ctx, o.DraftID)

	pub, err := mgr.PublishAgency(ctx, o.DraftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.WorkItems) != 1 || pub.Document.WorkItems[0].ID != step.ID || pub.Document.WorkItems[0].AccountableRoleID != role.ID {
		t.Fatalf("expected the step's accountable role in the document, got %+v", pub.Document.WorkItems)
	}
	if len(pub.Document.RoleAssignments) != 0 {
		t.Fatalf("a new document carries no role assignments, got %+v", pub.Document.RoleAssignments)
	}
}

func TestPublishAgency_DanglingJoinsAreRefused(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	links, err := mgr.ListWorkflowObjectives(ctx, o.DraftID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil || len(links) != 1 {
		t.Fatalf("ListWorkflowObjectives: %v, %+v", err, links)
	}
	if err := db.Exec("update "+tableOf(t, s, "workflow_objective")+" set objective_id = ? where id = ?", "ghost-objective", links[0].ID).Error; err != nil {
		t.Fatalf("dangling the join: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("expected a dangling workflow objective to block the publish, got %v", err)
	}
	if err := db.Exec("update "+tableOf(t, s, "workflow_objective")+" set objective_id = ? where id = ?", o.ID, links[0].ID).Error; err != nil {
		t.Fatalf("restoring the join: %v", err)
	}

	step := onlyStep(t, mgr, ctx, o.DraftID)
	if err := db.Exec("update "+tableOf(t, s, "work_item")+" set accountable_role_id = ? where id = ?", "ghost-role", step.ID).Error; err != nil {
		t.Fatalf("dangling the accountable role: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "", ""); !errors.Is(err, mwanachamaagency.ErrPublishValidationFailed) {
		t.Fatalf("expected a dangling accountable role to block the publish, got %v", err)
	}
}

func TestPublishAgency_WarningsDoNotBlock(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	issues, err := mgr.ListIssues(ctx, draftID, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues(draft): %v", err)
	}
	for _, i := range issues {
		if i.Severity == models.IssueSeverityError {
			t.Fatalf("seed should be clean of errors, got %+v", i)
		}
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
}

func TestPublishAgency_Tag(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	p1, err := mgr.PublishAgency(ctx, draftID, "2026-Q4 restructure", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if p1.Tag != "2026-Q4 restructure" || p1.Version != 1 {
		t.Fatalf("expected the operator's tag on version 1, got %+v", p1)
	}

	if _, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: "correctness over speed"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "2026-Q4 restructure", ""); !errors.Is(err, mwanachamaagency.ErrDuplicateTag) {
		t.Fatalf("expected ErrDuplicateTag, got %v", err)
	}

	p2, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency (fallback tag): %v", err)
	}
	if p2.Tag != "v2" || p2.Version != 2 {
		t.Fatalf("expected an empty tag to fall back to v2, got %+v", p2)
	}
}

func TestPublishAgency_TagFallbackCollision(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	if _, err := mgr.PublishAgency(ctx, draftID, "v2", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: "correctness over speed"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "", ""); !errors.Is(err, mwanachamaagency.ErrDuplicateTag) {
		t.Fatalf("expected ErrDuplicateTag from the generated tag, got %v", err)
	}
}

func TestSetPublicationChannel(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)

	p, err := mgr.PublishAgency(ctx, o.DraftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	archived, err := mgr.SetPublicationChannel(ctx, p.Version, models.ChannelArchived)
	if err != nil {
		t.Fatalf("SetPublicationChannel: %v", err)
	}
	if archived.Channel != models.ChannelArchived {
		t.Fatalf("expected archived, got channel %q", archived.Channel)
	}

	back, err := mgr.SetPublicationChannel(ctx, p.Version, models.DefaultChannel)
	if err != nil {
		t.Fatalf("expected re-living an archived publication to be allowed, got %v", err)
	}
	if back.Channel != models.DefaultChannel {
		t.Fatalf("expected the publication back on production, got %q", back.Channel)
	}
}

func TestPublishAgency_Channels(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	prod, err := mgr.PublishAgency(ctx, draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency(production): %v", err)
	}
	if _, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: "correctness over speed"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	staging, err := mgr.PublishAgency(ctx, draftID, "release-2", "staging")
	if err != nil {
		t.Fatalf("PublishAgency(staging): %v", err)
	}

	gotProd, err := mgr.GetPublicationForChannel(ctx, models.DefaultChannel)
	if err != nil {
		t.Fatalf("GetPublicationForChannel(production): %v", err)
	}
	if gotProd.Version != prod.Version {
		t.Fatalf("expected production still on version %d, got %d", prod.Version, gotProd.Version)
	}
	gotStaging, err := mgr.GetPublicationForChannel(ctx, "staging")
	if err != nil {
		t.Fatalf("GetPublicationForChannel(staging): %v", err)
	}
	if gotStaging.Version != staging.Version {
		t.Fatalf("expected staging on version %d, got %d", staging.Version, gotStaging.Version)
	}

	if _, err := mgr.SetPublicationChannel(ctx, staging.Version, models.DefaultChannel); err != nil {
		t.Fatalf("SetPublicationChannel: %v", err)
	}
	live, err := mgr.ListPublications(ctx, mwanachamaagency.PublicationFilter{Channel: models.DefaultChannel})
	if err != nil {
		t.Fatalf("ListPublications: %v", err)
	}
	if len(live) != 1 || live[0].Version != staging.Version {
		t.Fatalf("expected exactly version %d on production, got %+v", staging.Version, live)
	}
}

func TestGetPublicationForChannel_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetPublicationForChannel(context.Background(), "staging"); !errors.Is(err, mwanachamaagency.ErrPublicationNotFound) {
		t.Fatalf("expected ErrPublicationNotFound, got %v", err)
	}
}

func TestGetPublication_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetPublication(context.Background(), 99); !errors.Is(err, mwanachamaagency.ErrPublicationNotFound) {
		t.Fatalf("expected ErrPublicationNotFound, got %v", err)
	}
}

func TestPublishAgency_CarriesDocument(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, o, w, wi := seedLiveAgency(t, mgr, ctx)
	draftID := g.DraftID
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	wi.AccountableRoleID = role.ID
	if _, err := mgr.UpdateWorkItem(ctx, wi); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	link, err := mgr.ListWorkflowObjectives(ctx, draftID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil || len(link) != 1 {
		t.Fatalf("ListWorkflowObjectives: %v, %+v", err, link)
	}

	pub, err := mgr.PublishAgency(ctx, draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	doc := pub.Document
	if doc.Agency.Name == "" {
		t.Error("expected the Agency's own identity in the document")
	}
	if len(doc.Goals) != 1 || doc.Goals[0].ID != g.ID || doc.Goals[0].Code != g.Code {
		t.Errorf("expected the Goal carried by id and Code, got %+v", doc.Goals)
	}
	if len(doc.Objectives) != 1 || doc.Objectives[0].GoalID != g.ID {
		t.Errorf("expected the Objective still pointing at its Goal, got %+v", doc.Objectives)
	}
	if len(doc.Workflows) != 1 || doc.Workflows[0].ID != w.ID {
		t.Errorf("expected the Workflow, got %+v", doc.Workflows)
	}
	if len(doc.WorkItems) != 1 || doc.WorkItems[0].ID != wi.ID || doc.WorkItems[0].WorkflowID != w.ID {
		t.Errorf("expected the WorkItem under its Workflow, got %+v", doc.WorkItems)
	}
	if len(doc.WorkflowObjectives) != 1 || doc.WorkflowObjectives[0].WorkflowID != w.ID ||
		doc.WorkflowObjectives[0].ObjectiveID != o.ID || doc.WorkflowObjectives[0].ID != link[0].ID {
		t.Errorf("expected the workflow objective carried by workflow and objective id, got %+v", doc.WorkflowObjectives)
	}
	if doc.WorkItems[0].AccountableRoleID != role.ID {
		t.Errorf("expected the step's accountable role, got %+v", doc.WorkItems[0])
	}
	if len(doc.RoleAssignments) != 0 || doc.WorkItems[0].LegacyObjectiveID != "" {
		t.Errorf("a new document must not carry role assignments or the old objective keys: %+v %+v", doc.RoleAssignments, doc.WorkItems[0])
	}
	if len(doc.Roles) != 1 || doc.Roles[0].ID != role.ID {
		t.Errorf("expected the Role, got %+v", doc.Roles)
	}

	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
	reread, err := mgr.GetPublicationForChannel(ctx, "")
	if err != nil {
		t.Fatalf("GetPublicationForChannel: %v", err)
	}
	if len(reread.Document.Goals) != 1 || reread.Document.Goals[0].ID != g.ID {
		t.Fatalf("the published document moved with the draft: %+v", reread.Document.Goals)
	}
	if len(reread.Document.WorkItems) != 1 {
		t.Fatalf("expected the published WorkItem to survive the draft-side delete, got %+v", reread.Document.WorkItems)
	}
	if len(reread.Document.WorkflowObjectives) != 1 || len(reread.Document.WorkItems) != 1 || reread.Document.WorkItems[0].AccountableRoleID != role.ID {
		t.Fatalf("expected the join and the accountable role to survive the draft-side delete, got %+v / %+v",
			reread.Document.WorkflowObjectives, reread.Document.WorkItems)
	}
}

func TestCreateDraft_ForkFromPublication(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, w, wi := seedLiveAgency(t, mgr, ctx)
	draftID := g.DraftID
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	wi.AccountableRoleID = role.ID
	if _, err := mgr.UpdateWorkItem(ctx, wi); err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "release-1", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	if err := mgr.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
	remainingGoals, err := mgr.ListGoals(ctx, draftID, mwanachamaagency.GoalFilter{})
	if err != nil || len(remainingGoals) != 0 {
		t.Fatalf("expected the source draft to have no Goals, got %v / %+v", err, remainingGoals)
	}

	d, err := mgr.CreateDraft(ctx, "revisit release-1", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft(publication): %v", err)
	}

	draftGoals, err := mgr.ListGoals(ctx, d.ID, mwanachamaagency.GoalFilter{})
	if err != nil {
		t.Fatalf("ListGoals(draft): %v", err)
	}
	if len(draftGoals) != 1 {
		t.Fatalf("expected the published Goal materialized into the draft, got %+v", draftGoals)
	}
	if draftGoals[0].ID == g.ID {
		t.Error("expected the draft copy to be a new row, not the published id")
	}
	if draftGoals[0].Code != g.Code {
		t.Errorf("expected Code %q carried across, got %q", g.Code, draftGoals[0].Code)
	}

	draftObjectives, err := mgr.ListObjectives(ctx, d.ID, mwanachamaagency.ObjectiveFilter{})
	if err != nil || len(draftObjectives) != 1 {
		t.Fatalf("ListObjectives(draft): %v, %+v", err, draftObjectives)
	}
	if draftObjectives[0].GoalID != draftGoals[0].ID {
		t.Errorf("expected the Objective remapped onto the draft Goal %q, got %q",
			draftGoals[0].ID, draftObjectives[0].GoalID)
	}

	draftItems, err := mgr.ListWorkItems(ctx, d.ID, mwanachamaagency.WorkItemFilter{})
	if err != nil || len(draftItems) != 1 {
		t.Fatalf("ListWorkItems(draft): %v, %+v", err, draftItems)
	}
	draftWorkflows, err := mgr.ListWorkflows(ctx, d.ID, mwanachamaagency.WorkflowFilter{})
	if err != nil || len(draftWorkflows) != 1 {
		t.Fatalf("ListWorkflows(draft): %v, %+v", err, draftWorkflows)
	}
	if draftItems[0].WorkflowID != draftWorkflows[0].ID || draftItems[0].ID == wi.ID || draftItems[0].Code != wi.Code {
		t.Errorf("expected the step remapped onto the draft Workflow with a new id and the same Code, got %+v", draftItems[0])
	}
	if draftWorkflows[0].ID == w.ID || draftWorkflows[0].Code != w.Code {
		t.Errorf("expected a new Workflow id with the same Code, got %+v", draftWorkflows[0])
	}

	draftLinks, err := mgr.ListWorkflowObjectives(ctx, d.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil || len(draftLinks) != 1 {
		t.Fatalf("ListWorkflowObjectives(draft): %v, %+v", err, draftLinks)
	}
	if draftLinks[0].WorkflowID != draftWorkflows[0].ID || draftLinks[0].ObjectiveID != draftObjectives[0].ID {
		t.Errorf("expected the workflow objective remapped onto both draft rows, got %+v", draftLinks[0])
	}

	draftRoles, err := mgr.ListRoles(ctx, d.ID, mwanachamaagency.RoleFilter{})
	if err != nil || len(draftRoles) != 1 {
		t.Fatalf("ListRoles(draft): %v, %+v", err, draftRoles)
	}
	if draftItems[0].AccountableRoleID != draftRoles[0].ID {
		t.Errorf("expected the accountable role remapped onto the draft Role %q, got %+v", draftRoles[0].ID, draftItems[0])
	}
}

func TestCreateDraft_ForkFromUnknownPublication(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	seedPublishableAgency(t, mgr, ctx)

	_, err := mgr.CreateDraft(ctx, "revisit", "no-such-tag", "publication")
	if !errors.Is(err, mwanachamaagency.ErrPublicationNotFound) {
		t.Fatalf("expected ErrPublicationNotFound, got %v", err)
	}
}

func TestPublishedDocument_CarriesTheEffectiveEnabledValue(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	addStep(t, mgr, ctx, draftID, models.StepKindFillForm)
	taskmanager := moduleByCode(t, mgr, ctx, draftID, "taskmanager")
	if _, err := mgr.SetModuleEnabled(ctx, draftID, taskmanager.ID, true); err != nil {
		t.Fatalf("SetModuleEnabled: %v", err)
	}

	published, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	enabled := map[string]bool{}
	for _, mod := range published.Document.Modules {
		enabled[mod.Code] = mod.Enabled
	}
	want := map[string]bool{
		"auth": true, "actor": true, "comm": true,
		"taskmanager": true,
		"forms":       true, "catalog": true, "git": true,
		"agency": false, "custody": false, "digitaltwin": false, "accounting": false,
		"voting": false, "engage": false, "market": false,
	}
	for code, on := range want {
		if enabled[code] != on {
			t.Errorf("published %s enabled = %v, want %v (%v)", code, enabled[code], on, enabled)
		}
	}
	if len(enabled) != len(models.ModuleCatalog) {
		t.Errorf("published %d modules, want the catalog's %d", len(enabled), len(models.ModuleCatalog))
	}
}
