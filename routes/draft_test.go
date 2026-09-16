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

func TestCreateDraftAndLifecycle(t *testing.T) {
	mgr := newTestManager(t)
	ag, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "SWELab"})
	if err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	body := `{"description":"restructure","forked_from_id":"` + ag.ID + `","forked_from_type":"live"}`
	req := httptest.NewRequest(http.MethodPost, "/drafts", strings.NewReader(body))
	rec := httptest.NewRecorder()
	routes.CreateDraft(mgr)(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var draft models.Draft
	if err := json.Unmarshal(rec.Body.Bytes(), &draft); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/drafts/"+draft.ID, nil)
	getReq.SetPathValue("draftID", draft.ID)
	getRec := httptest.NewRecorder()
	routes.GetDraft(mgr)(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	listRec := httptest.NewRecorder()
	routes.ListDrafts(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}

	descReq := httptest.NewRequest(http.MethodPatch, "/drafts/"+draft.ID, strings.NewReader(`{"description":"restructure v2"}`))
	descReq.SetPathValue("draftID", draft.ID)
	descRec := httptest.NewRecorder()
	routes.UpdateDraftDescription(mgr)(descRec, descReq)
	if descRec.Code != http.StatusOK {
		t.Fatalf("update description status = %d, body = %s", descRec.Code, descRec.Body.String())
	}

	promoteReq := httptest.NewRequest(http.MethodPost, "/drafts/"+draft.ID+"/promote", nil)
	promoteReq.SetPathValue("draftID", draft.ID)
	promoteRec := httptest.NewRecorder()
	routes.PromoteDraft(mgr)(promoteRec, promoteReq)
	if promoteRec.Code != http.StatusOK {
		t.Fatalf("promote status = %d, body = %s", promoteRec.Code, promoteRec.Body.String())
	}

	// Already promoted -> re-promoting is refused.
	rePromoteRec := httptest.NewRecorder()
	routes.PromoteDraft(mgr)(rePromoteRec, promoteReq)
	if rePromoteRec.Code != http.StatusConflict {
		t.Fatalf("re-promote status = %d, body = %s", rePromoteRec.Code, rePromoteRec.Body.String())
	}
}

func TestArchiveDraft(t *testing.T) {
	mgr := newTestManager(t)
	ag, _ := mgr.CreateAgency(t.Context(), models.Agency{Name: "SWELab"})
	draft, err := mgr.CreateDraft(t.Context(), "abandoned", ag.ID, "live")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/drafts/"+draft.ID+"/archive", nil)
	req.SetPathValue("draftID", draft.ID)
	rec := httptest.NewRecorder()
	routes.ArchiveDraft(mgr)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var archived models.Draft
	if err := json.Unmarshal(rec.Body.Bytes(), &archived); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if archived.Status != models.DraftStatusArchived {
		t.Fatalf("expected archived status, got %q", archived.Status)
	}
}
