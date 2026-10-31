package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func controlsByCode(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID string) map[string]models.Control {
	t.Helper()
	all, err := mgr.ListControls(context.Background(), draftID, mwanachamaagency.ControlFilter{})
	if err != nil {
		t.Fatalf("ListControls: %v", err)
	}
	out := make(map[string]models.Control, len(all))
	for _, c := range all {
		out[c.Code] = c
	}
	return out
}

func TestImportDocument_RulesAndValuesBecomeAdvisoryControls(t *testing.T) {
	doc := readDocument(t, wakalaDocument)
	if len(doc.Rules)+len(doc.Values) == 0 {
		t.Skip("the fixture carries no rules or values")
	}
	mgr := newTestManager(t)
	ctx := context.Background()

	out, err := mgr.ImportDocument(ctx, "legacy", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	controls := controlsByCode(t, mgr, out.Draft.ID)
	for _, st := range append(append([]models.DocumentStatement{}, doc.Rules...), doc.Values...) {
		c, ok := controls[st.Code]
		if !ok {
			t.Errorf("no control keeps the code %s", st.Code)
			continue
		}
		if c.Text != st.Text || c.Action != "" || c.OnBreach != "" {
			t.Errorf("control %s = %+v, want advisory with text %q", st.Code, c, st.Text)
		}
	}

	rules, err := mgr.ListRules(ctx, out.Draft.ID, mwanachamaagency.RuleFilter{})
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	values, err := mgr.ListValues(ctx, out.Draft.ID, mwanachamaagency.ValueFilter{})
	if err != nil {
		t.Fatalf("ListValues: %v", err)
	}
	if len(rules)+len(values) != 0 {
		t.Errorf("rules=%d values=%d were written beside the controls", len(rules), len(values))
	}
}

func TestImportDocument_ConvertedAttachmentsStillResolve(t *testing.T) {
	doc := readDocument(t, wakalaDocument)
	if len(doc.RuleAttachments)+len(doc.ValueAttachments) == 0 {
		t.Skip("the fixture carries no attachments")
	}
	mgr := newTestManager(t)
	ctx := context.Background()

	out, err := mgr.ImportDocument(ctx, "legacy", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	controls := controlsByCode(t, mgr, out.Draft.ID)
	written := 0
	for _, c := range controls {
		attached, err := mgr.ListControlAttachments(ctx, c.ID)
		if err != nil {
			t.Fatalf("ListControlAttachments: %v", err)
		}
		written += len(attached)
	}
	if written != out.Attachments {
		t.Errorf("attachments listed = %d, import reported %d", written, out.Attachments)
	}
}

func TestImportDocument_ControlsInADocumentImportDirectly(t *testing.T) {
	doc := models.AgencyDocument{
		Agency: models.DocumentAgency{Name: "Direct"},
		Controls: []models.DocumentControl{
			{ID: "c1", Code: "C-1", Text: "Two approvers over 5000", Action: "payments.release", OnBreach: models.OnBreachHoldForApproval, Severity: models.SeverityHigh},
		},
		Rules: []models.DocumentStatement{{ID: "r1", Code: "C-1", Text: "same code, ignored"}, {ID: "r2", Code: "RU-1", Text: "Never touch production"}},
	}
	mgr := newTestManager(t)

	out, err := mgr.ImportDocument(context.Background(), "direct", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	controls := controlsByCode(t, mgr, out.Draft.ID)
	if len(controls) != 2 {
		t.Fatalf("controls = %d, want 2: %+v", len(controls), controls)
	}
	if got := controls["C-1"]; got.Text != "Two approvers over 5000" || got.Action != "payments.release" || got.OnBreach != models.OnBreachHoldForApproval {
		t.Errorf("C-1 = %+v", got)
	}
	if got := controls["RU-1"]; got.Text != "Never touch production" || got.Action != "" {
		t.Errorf("RU-1 = %+v", got)
	}
}
