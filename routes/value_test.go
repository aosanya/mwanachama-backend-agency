package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func TestCreateValue(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/values", strings.NewReader(`{"text":"we prioritize correctness over speed"}`))
	rec := httptest.NewRecorder()
	routes.CreateValue(mgr)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Value
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Code == "" {
		t.Fatalf("unexpected value: %+v", out)
	}
}

func TestCreateValue_Invalid(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/values", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	routes.CreateValue(mgr)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

// TestValueAttachment_HTTPLifecycle drives attach/list/detach through the
// HTTP layer end to end.
func TestValueAttachment_HTTPLifecycle(t *testing.T) {
	mgr := newTestManager(t)
	value, err := mgr.CreateValue(t.Context(), models.Value{Text: "never touch production data"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	role, err := mgr.CreateRole(t.Context(), models.Role{Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/values/"+value.ID+"/attachments",
		strings.NewReader(`{"target_type":"role","target_id":"`+role.ID+`"}`))
	req.SetPathValue("valueID", value.ID)
	rec := httptest.NewRecorder()
	routes.CreateValueAttachment(mgr)(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("attach status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var att models.ValueAttachment
	if err := json.Unmarshal(rec.Body.Bytes(), &att); err != nil {
		t.Fatalf("unmarshal attachment: %v", err)
	}
	if att.ID == "" || att.TargetType != models.TargetTypeRole {
		t.Fatalf("unexpected attachment: %+v", att)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/values/"+value.ID+"/attachments", nil)
	listReq.SetPathValue("valueID", value.ID)
	listRec := httptest.NewRecorder()
	routes.ListValueAttachments(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var atts []models.ValueAttachment
	if err := json.Unmarshal(listRec.Body.Bytes(), &atts); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(atts) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(atts))
	}

	filterReq := httptest.NewRequest(http.MethodGet, "/values?role_id="+role.ID, nil)
	filterRec := httptest.NewRecorder()
	routes.ListValues(mgr)(filterRec, filterReq)
	if filterRec.Code != http.StatusOK {
		t.Fatalf("filter status = %d, body = %s", filterRec.Code, filterRec.Body.String())
	}
	var filtered []models.Value
	if err := json.Unmarshal(filterRec.Body.Bytes(), &filtered); err != nil {
		t.Fatalf("unmarshal filtered: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != value.ID {
		t.Fatalf("expected the role_id filter to find the value, got %+v", filtered)
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/values/"+value.ID+"/attachments/"+att.ID, nil)
	delReq.SetPathValue("attachmentID", att.ID)
	delRec := httptest.NewRecorder()
	routes.DeleteValueAttachment(mgr)(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", delRec.Code, delRec.Body.String())
	}
}

func TestCreateValueAttachment_UnknownTargetIs404(t *testing.T) {
	mgr := newTestManager(t)
	value, err := mgr.CreateValue(t.Context(), models.Value{Text: "x"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/values/"+value.ID+"/attachments",
		strings.NewReader(`{"target_type":"role","target_id":"unknown"}`))
	req.SetPathValue("valueID", value.ID)
	rec := httptest.NewRecorder()
	routes.CreateValueAttachment(mgr)(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
