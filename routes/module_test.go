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

func TestCreateModule(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/modules", strings.NewReader(`{"code":"widgets","name":"Widgets","enabled":true}`))
	rec := httptest.NewRecorder()
	routes.CreateModule(mgr)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Module
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Code != "widgets" || !out.Enabled {
		t.Fatalf("unexpected module: %+v", out)
	}
}

func TestCreateModule_EmptyCodeIsBadRequest(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/modules", strings.NewReader(`{"name":"no code"}`))
	rec := httptest.NewRecorder()
	routes.CreateModule(mgr)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateModule_DuplicateCodeIsConflict(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/modules", strings.NewReader(`{"code":"git","name":"Git again"}`))
	rec := httptest.NewRecorder()
	routes.CreateModule(mgr)(rec, req)

	// "git" is already live via gormstore.BackfillModules.
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetModule_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/modules/unknown", nil)
	req.SetPathValue("moduleID", "unknown")
	rec := httptest.NewRecorder()
	routes.GetModule(mgr)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListAndUpdateModule(t *testing.T) {
	mgr := newTestManager(t)
	mod, err := mgr.CreateModule(t.Context(), models.Module{Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/modules?code=widgets", nil)
	listRec := httptest.NewRecorder()
	routes.ListModules(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var list []models.Module
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 module matching the code filter, got %d", len(list))
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/modules/"+mod.ID, strings.NewReader(`{"name":"Widgets Renamed","enabled":true}`))
	updateReq.SetPathValue("moduleID", mod.ID)
	updateRec := httptest.NewRecorder()
	routes.UpdateModule(mgr)(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateRec.Code, updateRec.Body.String())
	}
	var updated models.Module
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal updated: %v", err)
	}
	if updated.Name != "Widgets Renamed" || !updated.Enabled || updated.Code != "widgets" {
		t.Fatalf("unexpected updated module: %+v", updated)
	}
}

func TestDeleteModule(t *testing.T) {
	mgr := newTestManager(t)
	mod, err := mgr.CreateModule(t.Context(), models.Module{Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/modules/"+mod.ID, nil)
	req.SetPathValue("moduleID", mod.ID)
	rec := httptest.NewRecorder()
	routes.DeleteModule(mgr)(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/modules/"+mod.ID, nil)
	getReq.SetPathValue("moduleID", mod.ID)
	getRec := httptest.NewRecorder()
	routes.GetModule(mgr)(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getRec.Code)
	}
}

// TestObjectiveModule_HTTPLifecycle drives the nested
// /objectives/{objectiveID}/modules sub-resource (attach/list/detach)
// through the HTTP layer end to end, mirroring
// TestValueAttachment_HTTPLifecycle's shape.
func TestObjectiveModule_HTTPLifecycle(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	goal, err := mgr.CreateGoal(ctx, models.Goal{Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objective, err := mgr.CreateObjective(ctx, models.Objective{GoalID: goal.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	mod, err := mgr.CreateModule(ctx, models.Module{Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/objectives/"+objective.ID+"/modules",
		strings.NewReader(`{"module_id":"`+mod.ID+`"}`))
	req.SetPathValue("objectiveID", objective.ID)
	rec := httptest.NewRecorder()
	routes.CreateObjectiveModule(mgr)(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("attach status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var link models.ObjectiveModule
	if err := json.Unmarshal(rec.Body.Bytes(), &link); err != nil {
		t.Fatalf("unmarshal link: %v", err)
	}
	if link.ID == "" || link.ModuleID != mod.ID || link.ObjectiveID != objective.ID {
		t.Fatalf("unexpected link: %+v", link)
	}

	// Duplicate attach is a conflict.
	dupReq := httptest.NewRequest(http.MethodPost, "/objectives/"+objective.ID+"/modules",
		strings.NewReader(`{"module_id":"`+mod.ID+`"}`))
	dupReq.SetPathValue("objectiveID", objective.ID)
	dupRec := httptest.NewRecorder()
	routes.CreateObjectiveModule(mgr)(dupRec, dupReq)
	if dupRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate link, got %d: %s", dupRec.Code, dupRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/objectives/"+objective.ID+"/modules", nil)
	listReq.SetPathValue("objectiveID", objective.ID)
	listRec := httptest.NewRecorder()
	routes.ListObjectiveModulesForObjective(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var links []models.ObjectiveModule
	if err := json.Unmarshal(listRec.Body.Bytes(), &links); err != nil {
		t.Fatalf("unmarshal links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/objectives/"+objective.ID+"/modules/"+link.ID, nil)
	delReq.SetPathValue("objectiveModuleID", link.ID)
	delRec := httptest.NewRecorder()
	routes.DeleteObjectiveModule(mgr)(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", delRec.Code, delRec.Body.String())
	}

	afterReq := httptest.NewRequest(http.MethodGet, "/objectives/"+objective.ID+"/modules", nil)
	afterReq.SetPathValue("objectiveID", objective.ID)
	afterRec := httptest.NewRecorder()
	routes.ListObjectiveModulesForObjective(mgr)(afterRec, afterReq)
	var afterLinks []models.ObjectiveModule
	if err := json.Unmarshal(afterRec.Body.Bytes(), &afterLinks); err != nil {
		t.Fatalf("unmarshal after-delete links: %v", err)
	}
	if len(afterLinks) != 0 {
		t.Fatalf("expected 0 links after detach, got %d", len(afterLinks))
	}
}

// TestCreateObjectiveModule_ObjectiveNotFoundIs404 confirms a nonexistent
// objectiveID (as named by the URL path, not the body) is refused cleanly.
func TestCreateObjectiveModule_ObjectiveNotFoundIs404(t *testing.T) {
	mgr := newTestManager(t)
	mod, err := mgr.CreateModule(t.Context(), models.Module{Code: "widgets", Name: "Widgets"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/objectives/unknown/modules", strings.NewReader(`{"module_id":"`+mod.ID+`"}`))
	req.SetPathValue("objectiveID", "unknown")
	rec := httptest.NewRecorder()
	routes.CreateObjectiveModule(mgr)(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
