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

func TestCreateRole(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	req := httptest.NewRequest(http.MethodPost, "/roles", strings.NewReader(`{"name":"bug-hunter","draft_id":"`+draftID+`"}`))
	rec := httptest.NewRecorder()
	routes.CreateRole(mgr)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Role
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Name != "bug-hunter" {
		t.Fatalf("unexpected role: %+v", out)
	}
}

func TestCreateRole_Invalid(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/roles", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	routes.CreateRole(mgr)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateRole_WithinDraft(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	role, err := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/roles/"+role.ID, strings.NewReader(`{"name":"renamed"}`))
	req.SetPathValue("roleID", role.ID)
	rec := httptest.NewRecorder()
	routes.UpdateRole(mgr)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteRole_WithinDraft(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	role, err := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/roles/"+role.ID, nil)
	req.SetPathValue("roleID", role.ID)
	rec := httptest.NewRecorder()
	routes.DeleteRole(mgr)(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListRoles(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	if _, err := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "bug-hunter"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/roles?draft_id="+draftID, nil)
	rec := httptest.NewRecorder()
	routes.ListRoles(mgr)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out []models.Role
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 role, got %d", len(out))
	}
}
