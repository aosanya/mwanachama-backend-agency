package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func seedRiskWithBothArms(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context, draftID string) (models.Risk, models.RiskPathway, models.RiskPathway) {
	t.Helper()
	risk, err := mgr.CreateRisk(ctx, models.Risk{DraftID: draftID, Title: "Unsold harvest", TopEvent: "A harvest goes unsold", Hazard: "Perishable produce", Owner: "owner"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	threat, err := mgr.CreateRiskPathway(ctx, models.RiskPathway{DraftID: draftID, RiskID: risk.ID, Side: models.RiskSideThreat, Title: "No buyer is listed", Scenario: "Farmers post produce nobody asks for"})
	if err != nil {
		t.Fatalf("CreateRiskPathway(threat): %v", err)
	}
	consequence, err := mgr.CreateRiskPathway(ctx, models.RiskPathway{DraftID: draftID, RiskID: risk.ID, Side: models.RiskSideConsequence, Title: "Produce spoils"})
	if err != nil {
		t.Fatalf("CreateRiskPathway(consequence): %v", err)
	}
	return risk, threat, consequence
}

func TestCreateRisk_MintsCodeAndRefusesAnUnnamedRisk(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	first, err := mgr.CreateRisk(ctx, models.Risk{DraftID: o.DraftID, Title: "One", TopEvent: "Something happens"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	second, err := mgr.CreateRisk(ctx, models.Risk{DraftID: o.DraftID, Title: "Two", TopEvent: "Something else happens"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	if first.Code != "RK-1" || second.Code != "RK-2" || first.ID == "" {
		t.Fatalf("codes = %q, %q", first.Code, second.Code)
	}
	for _, bad := range []models.Risk{{DraftID: o.DraftID, TopEvent: "x"}, {DraftID: o.DraftID, Title: "x"}} {
		if _, err := mgr.CreateRisk(ctx, bad); !errors.Is(err, mwanachamaagency.ErrInvalidRisk) {
			t.Fatalf("CreateRisk(%+v) = %v, want ErrInvalidRisk", bad, err)
		}
	}
	if _, err := mgr.CreateRisk(ctx, models.Risk{Title: "x", TopEvent: "y"}); !errors.Is(err, mwanachamaagency.ErrDraftRequired) {
		t.Fatalf("CreateRisk without a draft = %v, want ErrDraftRequired", err)
	}
}

func TestListRisks_FiltersByCodeAndTitleWithinOneDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	seedRiskWithBothArms(t, mgr, ctx, o.DraftID)
	if _, err := mgr.CreateRisk(ctx, models.Risk{DraftID: o.DraftID, Title: "Late inputs", TopEvent: "Seed arrives after planting"}); err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}

	all, err := mgr.ListRisks(ctx, o.DraftID, mwanachamaagency.RiskFilter{})
	if err != nil || len(all) != 2 {
		t.Fatalf("ListRisks = %+v, %v", all, err)
	}
	byTitle, _ := mgr.ListRisks(ctx, o.DraftID, mwanachamaagency.RiskFilter{Title: "harvest"})
	byCode, _ := mgr.ListRisks(ctx, o.DraftID, mwanachamaagency.RiskFilter{Code: "RK-2"})
	if len(byTitle) != 1 || byTitle[0].Title != "Unsold harvest" || len(byCode) != 1 || byCode[0].Title != "Late inputs" {
		t.Fatalf("byTitle = %+v, byCode = %+v", byTitle, byCode)
	}
}

func TestUpdateRisk_KeepsCodeAndDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	risk, _, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)

	updated, err := mgr.UpdateRisk(ctx, models.Risk{ID: risk.ID, Code: "RK-99", DraftID: "elsewhere", Title: "Unsold produce", TopEvent: risk.TopEvent, Owner: "supplier agent"})
	if err != nil {
		t.Fatalf("UpdateRisk: %v", err)
	}
	if updated.Code != risk.Code || updated.DraftID != risk.DraftID || updated.Title != "Unsold produce" || updated.Owner != "supplier agent" {
		t.Fatalf("updated = %+v", updated)
	}
	if _, err := mgr.UpdateRisk(ctx, models.Risk{ID: risk.ID, Title: "", TopEvent: "x"}); !errors.Is(err, mwanachamaagency.ErrInvalidRisk) {
		t.Fatalf("UpdateRisk blank title = %v", err)
	}
	if _, err := mgr.UpdateRisk(ctx, models.Risk{ID: "nope", Title: "x", TopEvent: "y"}); !errors.Is(err, mwanachamaagency.ErrRiskNotFound) {
		t.Fatalf("UpdateRisk unknown = %v", err)
	}
}

func TestDeleteRisk_RemovesItsPathwaysAndLeavesOtherRisks(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	doomed, _, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)
	kept, err := mgr.CreateRisk(ctx, models.Risk{DraftID: o.DraftID, Title: "Late inputs", TopEvent: "Seed arrives late"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	keptArm, err := mgr.CreateRiskPathway(ctx, models.RiskPathway{DraftID: o.DraftID, RiskID: kept.ID, Side: models.RiskSideThreat, Title: "Supplier is slow"})
	if err != nil {
		t.Fatalf("CreateRiskPathway: %v", err)
	}

	if err := mgr.DeleteRisk(ctx, doomed.ID); err != nil {
		t.Fatalf("DeleteRisk: %v", err)
	}
	if err := mgr.DeleteRisk(ctx, doomed.ID); !errors.Is(err, mwanachamaagency.ErrRiskNotFound) {
		t.Fatalf("second DeleteRisk = %v", err)
	}
	left, err := mgr.ListRiskPathways(ctx, o.DraftID, mwanachamaagency.RiskPathwayFilter{})
	if err != nil || len(left) != 1 || left[0].ID != keptArm.ID {
		t.Fatalf("pathways left = %+v, %v", left, err)
	}
}

func TestDeleteRole_LeavesRisksAlone(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, role := seedPublishableAgency(t, mgr, ctx)
	risk, _, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)

	if err := mgr.DeleteRole(ctx, role.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if _, err := mgr.GetRisk(ctx, risk.ID); err != nil {
		t.Fatalf("GetRisk after DeleteRole: %v", err)
	}
	arms, _ := mgr.ListRiskPathways(ctx, o.DraftID, mwanachamaagency.RiskPathwayFilter{RiskID: risk.ID})
	if len(arms) != 2 {
		t.Fatalf("pathways after DeleteRole = %+v", arms)
	}
}

func TestCreateRiskPathway_NeedsASideATitleAndARiskInTheSameDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	risk, _, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)
	other, err := mgr.CreateDraft(ctx, "another", o.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	cases := []struct {
		name string
		in   models.RiskPathway
		want error
	}{
		{"unknown side", models.RiskPathway{DraftID: o.DraftID, RiskID: risk.ID, Side: "cause", Title: "x"}, mwanachamaagency.ErrInvalidRiskPathway},
		{"no title", models.RiskPathway{DraftID: o.DraftID, RiskID: risk.ID, Side: models.RiskSideThreat}, mwanachamaagency.ErrInvalidRiskPathway},
		{"unknown risk", models.RiskPathway{DraftID: o.DraftID, RiskID: "nope", Side: models.RiskSideThreat, Title: "x"}, mwanachamaagency.ErrRiskNotFound},
		{"risk of another draft", models.RiskPathway{DraftID: other.ID, RiskID: risk.ID, Side: models.RiskSideThreat, Title: "x"}, mwanachamaagency.ErrRiskNotFound},
	}
	for _, c := range cases {
		if _, err := mgr.CreateRiskPathway(ctx, c.in); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

func TestListRiskPathways_FiltersBySideAndRisk(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	risk, threat, consequence := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)

	threats, _ := mgr.ListRiskPathways(ctx, o.DraftID, mwanachamaagency.RiskPathwayFilter{RiskID: risk.ID, Side: "threat"})
	consequences, _ := mgr.ListRiskPathways(ctx, o.DraftID, mwanachamaagency.RiskPathwayFilter{Side: "consequence"})
	none, _ := mgr.ListRiskPathways(ctx, o.DraftID, mwanachamaagency.RiskPathwayFilter{RiskID: "nope"})
	if len(threats) != 1 || threats[0].ID != threat.ID || threat.Code != "RP-1" {
		t.Fatalf("threats = %+v", threats)
	}
	if len(consequences) != 1 || consequences[0].ID != consequence.ID || len(none) != 0 {
		t.Fatalf("consequences = %+v, none = %+v", consequences, none)
	}
}

func TestUpdateRiskPathway_KeepsRiskAndCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	_, threat, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)

	updated, err := mgr.UpdateRiskPathway(ctx, models.RiskPathway{ID: threat.ID, RiskID: "elsewhere", Code: "RP-9", Side: models.RiskSideConsequence, Title: "Buyers go elsewhere"})
	if err != nil {
		t.Fatalf("UpdateRiskPathway: %v", err)
	}
	if updated.RiskID != threat.RiskID || updated.Code != threat.Code || updated.Side != models.RiskSideConsequence || updated.Title != "Buyers go elsewhere" {
		t.Fatalf("updated = %+v", updated)
	}
	if err := mgr.DeleteRiskPathway(ctx, threat.ID); err != nil {
		t.Fatalf("DeleteRiskPathway: %v", err)
	}
	if _, err := mgr.GetRiskPathway(ctx, threat.ID); !errors.Is(err, mwanachamaagency.ErrRiskPathwayNotFound) {
		t.Fatalf("GetRiskPathway after delete = %v", err)
	}
}

func TestCreateDraft_ForkCopiesRisksWithRemappedPathwaysAndKeptCodes(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	risk, threat, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)

	fork, err := mgr.CreateDraft(ctx, "copy", o.DraftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	risks, err := mgr.ListRisks(ctx, fork.ID, mwanachamaagency.RiskFilter{})
	if err != nil || len(risks) != 1 {
		t.Fatalf("fork risks = %+v, %v", risks, err)
	}
	if risks[0].ID == risk.ID || risks[0].Code != risk.Code || risks[0].Title != risk.Title || risks[0].Owner != risk.Owner {
		t.Fatalf("fork risk = %+v, source = %+v", risks[0], risk)
	}
	arms, err := mgr.ListRiskPathways(ctx, fork.ID, mwanachamaagency.RiskPathwayFilter{})
	if err != nil || len(arms) != 2 {
		t.Fatalf("fork pathways = %+v, %v", arms, err)
	}
	for _, arm := range arms {
		if arm.RiskID != risks[0].ID || arm.DraftID != fork.ID {
			t.Fatalf("fork pathway = %+v, want it on the copied risk", arm)
		}
	}
	if arms[0].Code != threat.Code {
		t.Fatalf("fork pathway code = %q, want %q kept", arms[0].Code, threat.Code)
	}
	source, _ := mgr.ListRiskPathways(ctx, o.DraftID, mwanachamaagency.RiskPathwayFilter{})
	if len(source) != 2 || source[0].RiskID != risk.ID {
		t.Fatalf("source pathways moved: %+v", source)
	}
}

func TestPublishAgency_CarriesRisksIntoTheDocumentAndBackOutOnAFork(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	seedRiskWithBothArms(t, mgr, ctx, o.DraftID)

	pub, err := mgr.PublishAgency(ctx, o.DraftID, "with-risks", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Risks) != 1 || len(pub.Document.RiskPathways) != 2 {
		t.Fatalf("document risks = %+v, pathways = %+v", pub.Document.Risks, pub.Document.RiskPathways)
	}
	doc := pub.Document.Risks[0]
	if doc.Code != "RK-1" || doc.TopEvent != "A harvest goes unsold" || doc.Hazard != "Perishable produce" || doc.Owner != "owner" {
		t.Fatalf("document risk = %+v", doc)
	}

	fork, err := mgr.CreateDraft(ctx, "back to with-risks", "with-risks", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	risks, _ := mgr.ListRisks(ctx, fork.ID, mwanachamaagency.RiskFilter{})
	arms, _ := mgr.ListRiskPathways(ctx, fork.ID, mwanachamaagency.RiskPathwayFilter{})
	if len(risks) != 1 || risks[0].Code != "RK-1" || len(arms) != 2 {
		t.Fatalf("fork risks = %+v, pathways = %+v", risks, arms)
	}
	for _, arm := range arms {
		if arm.RiskID != risks[0].ID {
			t.Fatalf("pathway %+v is not on the restored risk %s", arm, risks[0].ID)
		}
	}
	next, err := mgr.CreateRisk(ctx, models.Risk{DraftID: fork.ID, Title: "Late inputs", TopEvent: "Seed arrives late"})
	if err != nil || next.Code != "RK-2" {
		t.Fatalf("next risk = %+v, %v; want the code sequence past the restored one", next, err)
	}
}

func TestPublishAgency_AddingARiskIsAChange(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "first", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "second", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("unchanged publish = %v, want ErrNoChangesDetected", err)
	}
	risk, _, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "third", ""); err != nil {
		t.Fatalf("publish after adding a risk: %v", err)
	}
	if _, err := mgr.UpdateRisk(ctx, models.Risk{ID: risk.ID, Title: risk.Title, TopEvent: "A different event"}); err != nil {
		t.Fatalf("UpdateRisk: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, o.DraftID, "fourth", ""); err != nil {
		t.Fatalf("publish after editing a risk: %v", err)
	}
}

func TestScan_RiskWithoutAnArmOnlyWarnsButADanglingPathwayIsAnError(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	bare, err := mgr.CreateRisk(ctx, models.Risk{DraftID: o.DraftID, Title: "Bare", TopEvent: "Nothing says how"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	got := issuesFor(t, mgr, ctx, o.DraftID, "risk")
	if len(got) != 4 {
		t.Fatalf("risk issues = %+v, want four warnings for a bare bowtie", got)
	}
	for _, issue := range got {
		if issue.Severity != models.IssueSeverityWarning {
			t.Fatalf("risk issue %+v is not a warning", issue)
		}
	}
	if got := issuesFor(t, mgr, ctx, o.DraftID, "risk_pathway"); len(got) != 0 {
		t.Fatalf("pathway issues = %+v", got)
	}

	arm, err := mgr.CreateRiskPathway(ctx, models.RiskPathway{DraftID: o.DraftID, RiskID: bare.ID, Side: models.RiskSideThreat, Title: "Orphan soon"})
	if err != nil {
		t.Fatalf("CreateRiskPathway: %v", err)
	}
	risks, _ := s.ByRole("risk")
	if err := db.Exec("delete from "+s.TableFor(risks)+" where id = ?", bare.ID).Error; err != nil {
		t.Fatalf("plant a dangling pathway: %v", err)
	}
	got = issuesFor(t, mgr, ctx, o.DraftID, "risk_pathway")
	if len(got) != 1 || got[0].Severity != models.IssueSeverityError || got[0].EntityID != arm.ID {
		t.Fatalf("pathway issues = %+v, want one error on %s", got, arm.ID)
	}
}
