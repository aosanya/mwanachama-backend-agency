package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRecordRoutes_ABodyThatCarriesAGoneRoleFieldIsRefusedWithTheFieldNamed(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	wf, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Tenders"})
	clerk, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Clerk"})

	send := func(method, path, payload string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(payload)))
		return rec
	}
	const form = `"form_name":"Tender","questions":[{"prompt":"Closing date","answer_type":"date","required":true}]`
	plain := `{"draft_id":"` + draftID + `","workflow_id":"` + wf.ID + `","name":"Tender",` + form + `}`
	created := send(http.MethodPost, "/records", plain)
	if created.Code != http.StatusCreated {
		t.Fatalf("create a record with no role field: status = %d, body = %s", created.Code, created.Body.String())
	}

	for _, field := range []string{"role_id", "edit_role_id", "viewer_role_id", "delete_role_id"} {
		carried := `{"draft_id":"` + draftID + `","workflow_id":"` + wf.ID + `","name":"Other",` + form + `,"` + field + `":"` + clerk.ID + `"}`
		rec := send(http.MethodPost, "/records", carried)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), field) {
			t.Errorf("create carrying %s: status = %d, body = %s; want 400 naming the field", field, rec.Code, rec.Body.String())
		}
	}
}
