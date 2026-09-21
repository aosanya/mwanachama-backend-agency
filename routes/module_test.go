package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

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

// TestListAndGetModule lists and gets the backfilled catalog rows —
// Module has no HTTP Create/Update/Delete any more (AGD-002 reversed to
// fully read-only), so this exercises the always-present catalog rows
// (e.g. "auth") rather than a route-created Module.
func TestListAndGetModule(t *testing.T) {
	mgr := newTestManager(t)
	draftID := bootstrapDraftID(t, mgr)

	listReq := httptest.NewRequest(http.MethodGet, "/modules?draft_id="+draftID+"&code=auth", nil)
	listRec := httptest.NewRecorder()
	routes.ListModules(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var list []models.Module
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list) != 1 || list[0].Code != "auth" {
		t.Fatalf("expected exactly the auth module, got %+v", list)
	}
	if !list[0].Enabled || !list[0].Locked {
		t.Fatalf("expected auth to be Enabled and Locked, got %+v", list[0])
	}

	getReq := httptest.NewRequest(http.MethodGet, "/modules/"+list[0].ID, nil)
	getReq.SetPathValue("moduleID", list[0].ID)
	getRec := httptest.NewRecorder()
	routes.GetModule(mgr)(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}
	var got models.Module
	if err := json.Unmarshal(getRec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if got.Code != "auth" {
		t.Fatalf("unexpected module: %+v", got)
	}
}

// TestListModules_DefaultCatalogState asserts the HTTP list reflects
// AGD-002's reversed defaults on a fresh instance: only "auth" enabled,
// every other known code disabled.
func TestListModules_DefaultCatalogState(t *testing.T) {
	mgr := newTestManager(t)
	draftID := bootstrapDraftID(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/modules?draft_id="+draftID, nil)
	rec := httptest.NewRecorder()
	routes.ListModules(mgr)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var list []models.Module
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != len(models.ModuleCatalog) {
		t.Fatalf("expected %d modules, got %d", len(models.ModuleCatalog), len(list))
	}
	for _, mod := range list {
		wantEnabled := mod.Code == "auth"
		if mod.Enabled != wantEnabled {
			t.Fatalf("expected module %q Enabled=%v, got %+v", mod.Code, wantEnabled, mod)
		}
		if mod.Locked != wantEnabled {
			t.Fatalf("expected module %q Locked=%v, got %+v", mod.Code, wantEnabled, mod)
		}
	}
}

// TestObjectiveModule_HTTPLifecycle drives the nested
// /objectives/{objectiveID}/modules sub-resource (attach/list/detach)
// through the HTTP layer end to end, mirroring
// TestValueAttachment_HTTPLifecycle's shape. Module linking is unaffected
// by AGD-002's reversal — only the Module catalog's own mutability changed
// — so this attaches an already-backfilled catalog row ("auth") rather
// than a route-created Module.
func TestObjectiveModule_HTTPLifecycle(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	draftID := bootstrapDraftID(t, mgr)
	goal, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objective, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: goal.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	mods, err := mgr.ListModules(ctx, draftID, mwanachamaagency.ModuleFilter{Code: "auth"})
	if err != nil || len(mods) != 1 {
		t.Fatalf("ListModules(code=auth): rows=%+v err=%v", mods, err)
	}
	mod := mods[0]

	req := httptest.NewRequest(http.MethodPost, "/objectives/"+objective.ID+"/modules",
		strings.NewReader(`{"rationale":"needed","module_id":"`+mod.ID+`","draft_id":"`+draftID+`"}`))
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
		strings.NewReader(`{"rationale":"needed","module_id":"`+mod.ID+`","draft_id":"`+draftID+`"}`))
	dupReq.SetPathValue("objectiveID", objective.ID)
	dupRec := httptest.NewRecorder()
	routes.CreateObjectiveModule(mgr)(dupRec, dupReq)
	if dupRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate link, got %d: %s", dupRec.Code, dupRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/objectives/"+objective.ID+"/modules?draft_id="+draftID, nil)
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

	afterReq := httptest.NewRequest(http.MethodGet, "/objectives/"+objective.ID+"/modules?draft_id="+draftID, nil)
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
	draftID := bootstrapDraftID(t, mgr)
	mods, err := mgr.ListModules(t.Context(), draftID, mwanachamaagency.ModuleFilter{Code: "auth"})
	if err != nil || len(mods) != 1 {
		t.Fatalf("ListModules(code=auth): rows=%+v err=%v", mods, err)
	}

	req := httptest.NewRequest(http.MethodPost, "/objectives/unknown/modules", strings.NewReader(`{"rationale":"needed","module_id":"`+mods[0].ID+`","draft_id":"`+draftID+`"}`))
	req.SetPathValue("objectiveID", "unknown")
	rec := httptest.NewRecorder()
	routes.CreateObjectiveModule(mgr)(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
