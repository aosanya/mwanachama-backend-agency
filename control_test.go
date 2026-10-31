package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func paymentHoldControl(draftID string) models.Control {
	return models.Control{
		DraftID:      draftID,
		Text:         "Payments over 5000 need a second approver",
		Action:       "payments.release",
		CondField:    "amount",
		CondOperator: models.OperatorGt,
		CondValue:    "5000",
		OnBreach:     models.OnBreachHoldForApproval,
		Severity:     models.SeverityHigh,
	}
}

func TestCreateControl_MintsCodeAndKeepsAnAdvisoryControlAdvisory(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	first, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "We prioritise correctness over speed"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	second, err := mgr.CreateControl(ctx, paymentHoldControl(o.DraftID))
	if err != nil {
		t.Fatalf("CreateControl enforced: %v", err)
	}
	if first.Code != "C-1" || second.Code != "C-2" || first.ID == "" || first.Enforced() || !second.Enforced() {
		t.Fatalf("first = %+v, second = %+v", first, second)
	}
	got, err := mgr.GetControl(ctx, second.ID)
	if err != nil || got != second {
		t.Fatalf("GetControl = %+v, %v, want %+v", got, err, second)
	}
}

func TestCreateControl_RefusesWhatTheDefinitionCannotHold(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, w, _ := seedLiveAgency(t, mgr, ctx)

	withWorkflow := models.Control{DraftID: o.DraftID, Text: "Escalate", Action: "payments.release", OnBreach: models.OnBreachRunWorkflow, WorkflowID: w.ID}
	if _, err := mgr.CreateControl(ctx, withWorkflow); err != nil {
		t.Fatalf("CreateControl run_workflow in the same draft: %v", err)
	}

	cases := map[string]models.Control{
		"no text":                     {DraftID: o.DraftID},
		"action without an outcome":   {DraftID: o.DraftID, Text: "x", Action: "payments.release"},
		"outcome without an action":   {DraftID: o.DraftID, Text: "x", OnBreach: models.OnBreachBlock},
		"unknown outcome":             {DraftID: o.DraftID, Text: "x", Action: "a.b", OnBreach: "explode"},
		"unknown severity":            {DraftID: o.DraftID, Text: "x", Severity: "dire"},
		"half a condition":            {DraftID: o.DraftID, Text: "x", Action: "a.b", OnBreach: models.OnBreachBlock, CondField: "amount"},
		"unknown operator":            {DraftID: o.DraftID, Text: "x", Action: "a.b", OnBreach: models.OnBreachBlock, CondField: "amount", CondOperator: "near", CondValue: "1"},
		"condition without an action": {DraftID: o.DraftID, Text: "x", CondField: "amount", CondOperator: models.OperatorGt, CondValue: "1"},
		"run_workflow without one":    {DraftID: o.DraftID, Text: "x", Action: "a.b", OnBreach: models.OnBreachRunWorkflow},
		"run_workflow naming nothing": {DraftID: o.DraftID, Text: "x", Action: "a.b", OnBreach: models.OnBreachRunWorkflow, WorkflowID: "nope"},
		"a workflow on a block":       {DraftID: o.DraftID, Text: "x", Action: "a.b", OnBreach: models.OnBreachBlock, WorkflowID: w.ID},
	}
	for name, c := range cases {
		if _, err := mgr.CreateControl(ctx, c); !errors.Is(err, mwanachamaagency.ErrInvalidControl) {
			t.Errorf("%s: CreateControl = %v, want ErrInvalidControl", name, err)
		}
	}
	if _, err := mgr.CreateControl(ctx, models.Control{Text: "x"}); !errors.Is(err, mwanachamaagency.ErrDraftRequired) {
		t.Fatalf("CreateControl without a draft = %v, want ErrDraftRequired", err)
	}
}

func TestCreateControl_RefusesAWorkflowFromAnotherDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	other, err := mgr.CreateDraft(ctx, "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	foreign, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: other.ID, Name: "Elsewhere"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	c := models.Control{DraftID: o.DraftID, Text: "x", Action: "a.b", OnBreach: models.OnBreachRunWorkflow, WorkflowID: foreign.ID}
	if _, err := mgr.CreateControl(ctx, c); !errors.Is(err, mwanachamaagency.ErrInvalidControl) {
		t.Fatalf("CreateControl = %v, want ErrInvalidControl", err)
	}
}

func TestListControls_FiltersByCodeTextAndAction(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	if _, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Never touch production data"}); err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := mgr.CreateControl(ctx, paymentHoldControl(o.DraftID)); err != nil {
		t.Fatalf("CreateControl: %v", err)
	}

	all, err := mgr.ListControls(ctx, o.DraftID, mwanachamaagency.ControlFilter{})
	if err != nil || len(all) != 2 {
		t.Fatalf("ListControls = %+v, %v", all, err)
	}
	byText, _ := mgr.ListControls(ctx, o.DraftID, mwanachamaagency.ControlFilter{Text: "production"})
	byCode, _ := mgr.ListControls(ctx, o.DraftID, mwanachamaagency.ControlFilter{Code: "C-2"})
	byAction, _ := mgr.ListControls(ctx, o.DraftID, mwanachamaagency.ControlFilter{Action: "payments.release"})
	if len(byText) != 1 || byText[0].Code != "C-1" || len(byCode) != 1 || byCode[0].Action != "payments.release" || len(byAction) != 1 || byAction[0].Code != "C-2" {
		t.Fatalf("byText = %+v, byCode = %+v, byAction = %+v", byText, byCode, byAction)
	}
}

func TestUpdateControl_MovesBetweenAdvisoryAndEnforcedAndKeepsCode(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	c, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Two approvers on payments"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}

	enforced := paymentHoldControl("elsewhere")
	enforced.ID = c.ID
	enforced.Code = "C-99"
	got, err := mgr.UpdateControl(ctx, enforced)
	if err != nil {
		t.Fatalf("UpdateControl: %v", err)
	}
	if got.Code != c.Code || got.DraftID != c.DraftID || !got.Enforced() || got.OnBreach != models.OnBreachHoldForApproval {
		t.Fatalf("updated = %+v", got)
	}

	back, err := mgr.UpdateControl(ctx, models.Control{ID: c.ID, Text: "Two approvers on payments"})
	if err != nil || back.Enforced() || back.OnBreach != "" || back.CondField != "" {
		t.Fatalf("back to advisory = %+v, %v", back, err)
	}
	if _, err := mgr.UpdateControl(ctx, models.Control{ID: c.ID}); !errors.Is(err, mwanachamaagency.ErrInvalidControl) {
		t.Fatalf("UpdateControl blank text = %v, want ErrInvalidControl", err)
	}
	if _, err := mgr.UpdateControl(ctx, models.Control{ID: "missing", Text: "x"}); !errors.Is(err, mwanachamaagency.ErrControlNotFound) {
		t.Fatalf("UpdateControl missing = %v, want ErrControlNotFound", err)
	}
}

func TestDeleteControl_TakesItsAttachmentsAndRefusesAnArchivedDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, _, o, w, _ := seedLiveAgency(t, mgr, ctx)
	c, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Review before release"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, c.ID, models.TargetTypeWorkflow, w.ID); err != nil {
		t.Fatalf("CreateControlAttachment: %v", err)
	}

	if err := mgr.DeleteControl(ctx, c.ID); err != nil {
		t.Fatalf("DeleteControl: %v", err)
	}
	if _, err := mgr.GetControl(ctx, c.ID); !errors.Is(err, mwanachamaagency.ErrControlNotFound) {
		t.Fatalf("GetControl after delete = %v, want ErrControlNotFound", err)
	}
	if err := mgr.DeleteControl(ctx, c.ID); !errors.Is(err, mwanachamaagency.ErrControlNotFound) {
		t.Fatalf("second DeleteControl = %v, want ErrControlNotFound", err)
	}
}
