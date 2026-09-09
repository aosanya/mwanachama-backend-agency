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

func TestCreateListAndRenameLevel(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/levels", strings.NewReader(`{"name":"Division","depth":0}`))
	rec := httptest.NewRecorder()
	routes.CreateLevel(mgr)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Level
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/levels", nil)
	listRec := httptest.NewRecorder()
	routes.ListLevels(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}

	renameReq := httptest.NewRequest(http.MethodPatch, "/levels/"+out.ID, strings.NewReader(`{"name":"Team"}`))
	renameReq.SetPathValue("levelID", out.ID)
	renameRec := httptest.NewRecorder()
	routes.RenameLevel(mgr)(renameRec, renameReq)
	if renameRec.Code != http.StatusOK {
		t.Fatalf("rename status = %d, body = %s", renameRec.Code, renameRec.Body.String())
	}
}

func TestDeleteLevel_WornRefused(t *testing.T) {
	mgr := newTestManager(t)
	level, _ := mgr.CreateLevel(t.Context(), models.Level{Name: "Division"})
	if _, err := mgr.CreateDepartment(t.Context(), models.Department{Name: "Software", LevelID: level.ID}); err != nil {
		t.Fatalf("CreateDepartment: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/levels/"+level.ID, nil)
	req.SetPathValue("levelID", level.ID)
	rec := httptest.NewRecorder()
	routes.DeleteLevel(mgr)(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteLevel_Succeeds(t *testing.T) {
	mgr := newTestManager(t)
	level, _ := mgr.CreateLevel(t.Context(), models.Level{Name: "Unused"})

	req := httptest.NewRequest(http.MethodDelete, "/levels/"+level.ID, nil)
	req.SetPathValue("levelID", level.ID)
	rec := httptest.NewRecorder()
	routes.DeleteLevel(mgr)(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
