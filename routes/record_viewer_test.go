package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRecordRoutes_ViewerRoleIdIsWrittenAndReadBack(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	wf, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Group order"})
	coordinator, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Group coordinator"})
	farmer, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Farmer"})

	send := func(method, path, payload string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(payload)))
		return rec
	}
	body := func(viewer string) string {
		return `{"draft_id":"` + draftID + `","workflow_id":"` + wf.ID + `","name":"Payments","role_id":"` + coordinator.ID + `",` +
			`"viewer_role_id":"` + viewer + `","form_name":"Payment","questions":[{"prompt":"Enter the amount paid","answer_type":"currency","required":true}]}`
	}

	created := send(http.MethodPost, "/records", body(farmer.ID))
	if created.Code != http.StatusCreated {
		t.Fatalf("create record: status = %d, body = %s", created.Code, created.Body.String())
	}
	var record models.Record
	if err := json.Unmarshal(created.Body.Bytes(), &record); err != nil || record.ViewerRoleID != farmer.ID {
		t.Fatalf("created record = %+v, %v; want the Farmer as viewer", record, err)
	}

	read := send(http.MethodGet, "/records/"+record.ID, "")
	var got models.Record
	if err := json.Unmarshal(read.Body.Bytes(), &got); err != nil || got.ViewerRoleID != farmer.ID || got.RoleID != coordinator.ID {
		t.Fatalf("read record: status = %d, body = %s", read.Code, read.Body.String())
	}

	if rec := send(http.MethodPost, "/records", strings.Replace(body("no-such-role"), `"name":"Payments"`, `"name":"Other"`, 1)); rec.Code != http.StatusNotFound {
		t.Errorf("a viewer role that does not exist: status = %d, want 404, body = %s", rec.Code, rec.Body.String())
	}

	updated := send(http.MethodPatch, "/records/"+record.ID, body(""))
	var after models.Record
	if updated.Code != http.StatusOK || json.Unmarshal(updated.Body.Bytes(), &after) != nil || after.ViewerRoleID != "" {
		t.Fatalf("update clearing the viewer: status = %d, body = %s", updated.Code, updated.Body.String())
	}
}

func TestRecordRoutes_AnUpdateThatLeavesViewerRoleIdOutClearsIt(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	wf, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Group order"})
	coordinator, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Group coordinator"})
	farmer, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Farmer"})

	send := func(method, path, payload string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(payload)))
		return rec
	}
	const form = `"form_name":"Payment","questions":[{"prompt":"Enter the amount paid","answer_type":"currency","required":true}]`
	named := `{"draft_id":"` + draftID + `","workflow_id":"` + wf.ID + `","name":"Payments","role_id":"` + coordinator.ID + `",`

	created := send(http.MethodPost, "/records", named+`"viewer_role_id":"`+farmer.ID+`",`+form+`}`)
	var record models.Record
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &record) != nil || record.ViewerRoleID != farmer.ID {
		t.Fatalf("create record with a viewer: status = %d, body = %s", created.Code, created.Body.String())
	}

	updated := send(http.MethodPatch, "/records/"+record.ID, named+form+`}`)
	var after models.Record
	if updated.Code != http.StatusOK || json.Unmarshal(updated.Body.Bytes(), &after) != nil || after.ViewerRoleID != "" {
		t.Fatalf("update that leaves viewer_role_id out: status = %d, body = %s", updated.Code, updated.Body.String())
	}

	read := send(http.MethodGet, "/records/"+record.ID, "")
	var got models.Record
	if json.Unmarshal(read.Body.Bytes(), &got) != nil || got.ViewerRoleID != "" || got.RoleID != coordinator.ID {
		t.Fatalf("read after the update: status = %d, body = %s; want no viewer and the Coordinator still adding", read.Code, read.Body.String())
	}
}
