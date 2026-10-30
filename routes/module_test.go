package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestGetModule_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/modules/unknown", nil)
	req.SetPathValue("moduleID", "unknown")
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListAndGetModule(t *testing.T) {
	mgr := newTestManager(t)
	draftID := bootstrapDraftID(t, mgr)

	listReq := httptest.NewRequest(http.MethodGet, "/modules?draft_id="+draftID+"&code=auth", nil)
	listRec := httptest.NewRecorder()
	serve(mgr, listRec, listReq)
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
	serve(mgr, getRec, getReq)
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

func TestListModules_DefaultCatalogState(t *testing.T) {
	mgr := newTestManager(t)
	draftID := bootstrapDraftID(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/modules?draft_id="+draftID, nil)
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
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
		locked := mod.Code == "auth" || mod.Code == "actor"
		wantEnabled := locked || mod.Code == "comm"
		if mod.Enabled != wantEnabled || mod.Explicit != wantEnabled {
			t.Fatalf("expected module %q Enabled=%v, got %+v", mod.Code, wantEnabled, mod)
		}
		if mod.Locked != locked {
			t.Fatalf("expected module %q Locked=%v, got %+v", mod.Code, locked, mod)
		}
		if mod.NeededBySteps {
			t.Fatalf("expected module %q needed by no step in an empty draft, got %+v", mod.Code, mod)
		}
	}
}

func putEnabled(mgr mwanachamaagency.AgencyManager, moduleID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/modules/"+moduleID+"/enabled", strings.NewReader(body))
	req.SetPathValue("moduleID", moduleID)
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
	return rec
}

func TestSetModuleEnabled_HTTPSwitchesOnAndOff(t *testing.T) {
	mgr := newTestManager(t)
	draftID := bootstrapDraftID(t, mgr)
	git := moduleWithCode(t, mgr, draftID, "git")

	rec := putEnabled(mgr, git.ID, `{"draft_id":"`+draftID+`","enabled":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("switch on status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var on models.Module
	if err := json.Unmarshal(rec.Body.Bytes(), &on); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !on.Enabled || !on.Explicit || on.NeededBySteps {
		t.Fatalf("after switching on: %+v", on)
	}

	rec = putEnabled(mgr, git.ID, `{"draft_id":"`+draftID+`","enabled":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("switch off status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var off models.Module
	if err := json.Unmarshal(rec.Body.Bytes(), &off); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if off.Enabled || off.Explicit {
		t.Fatalf("after switching off: %+v", off)
	}
}

func TestSetModuleEnabled_HTTPRefusesALockedModuleSwitchedOff(t *testing.T) {
	mgr := newTestManager(t)
	draftID := bootstrapDraftID(t, mgr)
	auth := moduleWithCode(t, mgr, draftID, "auth")

	rec := putEnabled(mgr, auth.ID, `{"draft_id":"`+draftID+`","enabled":false}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !moduleWithCode(t, mgr, draftID, "auth").Enabled {
		t.Fatal("a refused switch-off still turned auth off")
	}
}

func TestSetModuleEnabled_HTTPIgnoresAModuleOfAnotherDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := t.Context()
	draftA := bootstrapDraftID(t, mgr)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draftB, err := mgr.CreateDraft(ctx, "b", draftA, "draft")
	if err != nil {
		t.Fatalf("CreateDraft fork: %v", err)
	}
	gitA := moduleWithCode(t, mgr, draftA, "git")

	rec := putEnabled(mgr, gitA.ID, `{"draft_id":"`+draftB.ID+`","enabled":true}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if moduleWithCode(t, mgr, draftA, "git").Explicit || moduleWithCode(t, mgr, draftB.ID, "git").Explicit {
		t.Fatal("a mismatched draft id still switched a module")
	}
}

func TestSetModuleEnabled_HTTPMissingModuleIs404(t *testing.T) {
	mgr := newTestManager(t)
	draftID := bootstrapDraftID(t, mgr)
	rec := putEnabled(mgr, "unknown", `{"draft_id":"`+draftID+`","enabled":true}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
