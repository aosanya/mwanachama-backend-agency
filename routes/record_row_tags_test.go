package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRecordRowTagRoutes(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	wf, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: draftID, Name: "Group order"})
	coordinator, _ := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Group coordinator"})
	record, err := mgr.CreateRecord(ctx, models.Record{DraftID: draftID, WorkflowID: wf.ID, Name: "Payments", RoleID: coordinator.ID, FormName: "Payment",
		Questions: []models.StepQuestion{{Prompt: "Enter the amount paid", AnswerType: "currency", Required: true}}})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	send := func(method, path, payload string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(method, path, strings.NewReader(payload)))
		return rec
	}
	add := func(amount, tags string) models.RecordRow {
		rec := send(http.MethodPost, "/record-rows", `{"draft_id":"`+draftID+`","record_id":"`+record.ID+`","role_id":"`+coordinator.ID+
			`","tags":`+tags+`,"answers":{"Enter the amount paid":"`+amount+`"}}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("add row: status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var row models.RecordRow
		if err := json.Unmarshal(rec.Body.Bytes(), &row); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return row
	}
	listed := func(query string) []models.RecordRow {
		rec := send(http.MethodGet, "/record-rows?draft_id="+draftID+"&record_id="+record.ID+query, "")
		var rows []models.RecordRow
		if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("list rows %q: status = %d, body = %s", query, rec.Code, rec.Body.String())
		}
		return rows
	}

	first := add("100", `["a"]`)
	add("200", `["b"]`)
	add("300", `[]`)

	if got := listed("&tags=a"); len(got) != 1 || got[0].ID != first.ID {
		t.Errorf("tags=a listed %+v", got)
	}
	if got := listed("&tags=a&tags=b"); len(got) != 2 {
		t.Errorf("tags=a&tags=b listed %d rows, want 2", len(got))
	}
	if got := listed("&tags=all"); len(got) != 3 {
		t.Errorf("tags=all listed %d rows, want 3", len(got))
	}
	if got := listed(""); len(got) != 3 {
		t.Errorf("no tags listed %d rows, want 3", len(got))
	}
	if got := listed("&tags="); len(got) != 0 {
		t.Errorf("a blank tag listed %d rows, want none", len(got))
	}

	tagged := send(http.MethodPost, "/record-rows/"+first.ID+"/tags", `{"tag":"c"}`)
	var row models.RecordRow
	if err := json.Unmarshal(tagged.Body.Bytes(), &row); err != nil || tagged.Code != http.StatusOK || !slices.Equal(row.Tags, []string{"a", "c"}) {
		t.Fatalf("add tag: status = %d, body = %s", tagged.Code, tagged.Body.String())
	}
	if rec := send(http.MethodPost, "/record-rows/"+first.ID+"/tags", `{"tag":" "}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a blank tag: status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
	if rec := send(http.MethodPost, "/record-rows/no-such-row/tags", `{"tag":"c"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a row that does not exist: status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
	for _, method := range []string{http.MethodDelete, http.MethodPut, http.MethodPatch} {
		for _, path := range []string{"/record-rows/" + first.ID + "/tags", "/record-rows/" + first.ID + "/tags/c"} {
			if rec := send(method, path, `{"tag":"c"}`); rec.Code < 400 {
				t.Errorf("%s %s: status = %d, want a refusal", method, path, rec.Code)
			}
		}
	}
	if got := listed("&tags=c"); len(got) != 1 || !slices.Equal(got[0].Tags, []string{"a", "c"}) {
		t.Errorf("after the add and the refused removals, tags=c listed %+v", got)
	}
}
