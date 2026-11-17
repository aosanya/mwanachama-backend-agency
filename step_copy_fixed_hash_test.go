package mwanachamaagency

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const fixedDraft = "draft-fixed"

func fixedHashManager(t *testing.T, steps []models.WorkItem) *agencyManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := NewAgencyManager(db, s)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Lakeside"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	m := mgr.(*agencyManager)
	rows := []struct {
		role string
		row  any
	}{
		{roleGoal, models.Goal{ID: "g1", DraftID: fixedDraft, Title: "Lend well"}},
		{roleObjective, models.Objective{ID: "o1", DraftID: fixedDraft, GoalID: "g1", Title: "Lend well"}},
		{roleWorkflow, models.Workflow{ID: "w1", DraftID: fixedDraft, Name: "Loan payment"}},
		{roleWorkflowObjective, models.WorkflowObjective{ID: "wo1", DraftID: fixedDraft, WorkflowID: "w1", ObjectiveID: "o1"}},
	}
	for _, r := range rows {
		if err := m.copyRow(ctx, m.db, r.role, r.row); err != nil {
			t.Fatalf("copyRow %s: %v", r.role, err)
		}
	}
	for _, wi := range steps {
		wi.DraftID, wi.WorkflowID = fixedDraft, "w1"
		if err := m.copyRow(ctx, m.db, roleWorkItem, wi); err != nil {
			t.Fatalf("copyRow step %s: %v", wi.ID, err)
		}
	}
	return m
}

func fixedHash(t *testing.T, steps []models.WorkItem) string {
	t.Helper()
	m := fixedHashManager(t, steps)
	hash, err := m.contentHashFor(context.Background(), m.db, fixedDraft)
	if err != nil {
		t.Fatalf("contentHashFor: %v", err)
	}
	return hash
}

func olderFixedSteps() []models.WorkItem {
	return []models.WorkItem{
		{ID: "s1", Title: "Write the charter", Ordinality: 1},
		{ID: "s2", Title: "Collect the form", Ordinality: 2, Kind: models.StepKindFillForm, FormName: "Intake"},
		{ID: "s3", Title: "Empty calculating step", Ordinality: 3, Kind: models.StepKindCompute},
		{ID: "s4", Title: "Empty posting step", Ordinality: 4, Kind: models.StepKindPost},
	}
}

func withComputeFields(steps []models.WorkItem, change func(wi *models.WorkItem)) []models.WorkItem {
	out := append([]models.WorkItem(nil), steps...)
	change(&out[2])
	return out
}

func TestStepCopyFixedHash_AFixedAgencyOfOlderKindsHashesTheSameWithAndWithoutTheRules(t *testing.T) {
	ClearStepKindRegistry()
	t.Cleanup(ClearStepKindRegistry)
	before := fixedHash(t, olderFixedSteps())
	if again := fixedHash(t, olderFixedSteps()); again != before {
		t.Fatalf("a fixed document must hash the same twice: %s and %s", before, again)
	}
	registerStepCopyRules()
	if after := fixedHash(t, olderFixedSteps()); after != before {
		t.Fatalf("registering the copy rules moved the hash of an agency with none of the fields: %s then %s", before, after)
	}
}

func TestStepCopyFixedHash_ACalculatingStepHashesByItsFieldsAndNotByTheOrderOfItsBindings(t *testing.T) {
	ClearStepKindRegistry()
	t.Cleanup(ClearStepKindRegistry)
	registerStepCopyRules()
	older := fixedHash(t, olderFixedSteps())
	bindings := []models.StepBinding{
		{Input: "principal", SourceType: models.SourceAnswer, StepID: "s2", Question: "Amount requested"},
		{Input: "months", SourceType: models.SourceFixed, Value: "24"},
		{Input: "balance", SourceType: models.SourceLedgerBalance},
	}
	reversed := []models.StepBinding{bindings[2], bindings[1], bindings[0]}
	hashOf := func(change func(wi *models.WorkItem)) string {
		return fixedHash(t, withComputeFields(olderFixedSteps(), change))
	}
	july := hashOf(func(wi *models.WorkItem) {
		wi.Function, wi.Bindings, wi.ResultName = "fin.pmt@1", bindings, "Payment"
	})
	if july == older {
		t.Fatal("adding the fields of a calculating step must change the hash")
	}
	if again := hashOf(func(wi *models.WorkItem) {
		wi.Function, wi.Bindings, wi.ResultName = "fin.pmt@1", bindings, "Payment"
	}); again != july {
		t.Fatal("the same fields must hash the same")
	}
	if shuffled := hashOf(func(wi *models.WorkItem) {
		wi.Function, wi.Bindings, wi.ResultName = "fin.pmt@1", reversed, "Payment"
	}); shuffled != july {
		t.Fatal("reordering the bindings alone must not change the hash")
	}
	for label, change := range map[string]func(wi *models.WorkItem){
		"a new function version": func(wi *models.WorkItem) {
			wi.Function, wi.Bindings, wi.ResultName = "fin.pmt@2", bindings, "Payment"
		},
		"another result name": func(wi *models.WorkItem) {
			wi.Function, wi.Bindings, wi.ResultName = "fin.pmt@1", bindings, "Monthly payment"
		},
		"another binding": func(wi *models.WorkItem) {
			other := append([]models.StepBinding(nil), bindings...)
			other[1].Value = "36"
			wi.Function, wi.Bindings, wi.ResultName = "fin.pmt@1", other, "Payment"
		},
		"a failure choice": func(wi *models.WorkItem) {
			wi.Function, wi.Bindings, wi.ResultName = "fin.pmt@1", bindings, "Payment"
			wi.OnFailure = &models.StepFailure{Choice: models.FailureStop}
		},
		"a hold": func(wi *models.WorkItem) {
			wi.Function, wi.Bindings, wi.ResultName = "fin.pmt@1", bindings, "Payment"
			wi.Hold = &models.StepHold{ExpiresAfterDays: 14}
		},
	} {
		if hashOf(change) == july {
			t.Errorf("%s must change the hash", label)
		}
	}
}
