package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRecordRow_AnAmendmentAnswerAndTheApprovalStampAreKept(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	row := paymentRow(o, "order-a", "Wanjiru", "100")
	row.Answers["Changed: Enter the amount paid"] = "was 90, by amina at 2026-11-11"
	row.Answers[mwanachamaagency.ApprovedByPrompt] = "amina"
	row.Answers[mwanachamaagency.ApprovedAtPrompt] = "2026-11-11T10:00:00Z"
	saved, err := o.world.mgr.CreateRecordRow(ctx, row)
	if err != nil {
		t.Fatalf("a row with an amendment answer and an approval stamp was refused: %v", err)
	}
	if saved.Answers[mwanachamaagency.ApprovedByPrompt] != "amina" {
		t.Errorf("the approval stamp was not kept: %v", saved.Answers)
	}
}

func TestRecordRow_AnInventedPromptIsStillRefused(t *testing.T) {
	o := buildRoutedOrder(t)
	row := paymentRow(o, "order-a", "Wanjiru", "100")
	row.Answers["Approved by someone else"] = "amina"
	if _, err := o.world.mgr.CreateRecordRow(context.Background(), row); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
		t.Fatalf("an invented prompt = %v, want ErrInvalidRecordRow", err)
	}
}

func TestStepQuestion_ChecksApprovalLimitTravelsInTheJSON(t *testing.T) {
	var q models.StepQuestion
	if err := json.Unmarshal([]byte(`{"prompt":"Amount","answer_type":"currency","required":true,"checks_approval_limit":true}`), &q); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !q.ChecksApprovalLimit {
		t.Fatalf("checks_approval_limit was dropped: %+v", q)
	}
	out, err := json.Marshal(models.StepQuestion{Prompt: "Amount", AnswerType: models.AnswerCurrency})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("Unmarshal of %s: %v", out, err)
	}
	if _, present := back["checks_approval_limit"]; present {
		t.Errorf("an unset checks_approval_limit is written out: %s", out)
	}
}
