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

func enabledCodes(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID string) map[string]bool {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/modules?draft_id="+draftID+"&enabled=true", nil)
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /modules?draft_id=%s&enabled=true = %d %s", draftID, rec.Code, rec.Body.String())
	}
	var mods []models.Module
	if err := json.Unmarshal(rec.Body.Bytes(), &mods); err != nil {
		t.Fatalf("decode: %v", err)
	}
	out := map[string]bool{}
	for _, m := range mods {
		out[m.Code] = true
	}
	return out
}

func moduleWithCode(t *testing.T, mgr mwanachamaagency.AgencyManager, draftID, code string) models.Module {
	t.Helper()
	mods, err := mgr.ListModules(t.Context(), draftID, mwanachamaagency.ModuleFilter{Code: code})
	if err != nil {
		t.Fatalf("ListModules(%s, %s): %v", draftID, code, err)
	}
	for _, m := range mods {
		if m.Code == code {
			return m
		}
	}
	t.Fatalf("no %q module in draft %s", code, draftID)
	return models.Module{}
}

func TestACapabilityIsSwitchedOnOnlyByALinkInItsOwnDraft(t *testing.T) {
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
	goal, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftA, Title: "g"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objective, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftA, GoalID: goal.ID, Title: "o"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	gitA := moduleWithCode(t, mgr, draftA, "git")
	gitB := moduleWithCode(t, mgr, draftB.ID, "git")
	if gitA.Locked || gitA.Enabled || gitB.Enabled || gitA.ID == gitB.ID {
		t.Fatalf("precondition: an unlocked, switched-off git module in each draft, got a=%+v b=%+v", gitA, gitB)
	}

	link := func(moduleID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/objectives/"+objective.ID+"/modules",
			strings.NewReader(`{"rationale":"needs it","module_id":"`+moduleID+`","draft_id":"`+draftA+`"}`))
		rec := httptest.NewRecorder()
		serve(mgr, rec, req)
		return rec
	}

	if rec := link(gitB.ID); rec.Code != http.StatusNotFound {
		t.Fatalf("linking draft A's objective to draft B's module answered %d %s, want 404", rec.Code, rec.Body.String())
	}
	if enabledCodes(t, mgr, draftB.ID)["git"] || enabledCodes(t, mgr, draftA)["git"] {
		t.Fatalf("a refused cross-draft link switched git on")
	}

	if rec := link(gitA.ID); rec.Code != http.StatusCreated {
		t.Fatalf("same-draft link answered %d %s, want 201", rec.Code, rec.Body.String())
	}
	if !enabledCodes(t, mgr, draftA)["git"] {
		t.Fatalf("draft A's own link left git switched off in draft A")
	}
	if enabledCodes(t, mgr, draftB.ID)["git"] {
		t.Fatalf("draft A's link switched git on in draft B")
	}
	if got, err := mgr.GetModule(ctx, gitB.ID); err != nil || got.Enabled {
		t.Fatalf("GetModule(draft B git) = %+v, %v; want switched off", got, err)
	}

	for _, door := range []struct{ method, path, body string }{
		{http.MethodPost, "/modules", `{"code":"git","enabled":true,"draft_id":"` + draftB.ID + `"}`},
		{http.MethodPatch, "/modules/" + gitB.ID, `{"enabled":true}`},
		{http.MethodDelete, "/modules/" + gitB.ID, ""},
	} {
		req := httptest.NewRequest(door.method, door.path, strings.NewReader(door.body))
		rec := httptest.NewRecorder()
		serve(mgr, rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s answered %d %s, want 405: the catalog has no write door", door.method, door.path, rec.Code, rec.Body.String())
		}
	}
	if enabledCodes(t, mgr, draftB.ID)["git"] {
		t.Fatalf("a write to the catalog switched git on in draft B")
	}
}
