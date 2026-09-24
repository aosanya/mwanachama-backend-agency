package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateDraftAndLifecycle(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	body := `{"description":"restructure","forked_from_type":""}`
	req := httptest.NewRequest(http.MethodPost, "/drafts", strings.NewReader(body))
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
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
	serve(mgr, getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	listRec := httptest.NewRecorder()
	serve(mgr, listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}

	descReq := httptest.NewRequest(http.MethodPatch, "/drafts/"+draft.ID, strings.NewReader(`{"description":"restructure v2"}`))
	descReq.SetPathValue("draftID", draft.ID)
	descRec := httptest.NewRecorder()
	serve(mgr, descRec, descReq)
	if descRec.Code != http.StatusOK {
		t.Fatalf("update description status = %d, body = %s", descRec.Code, descRec.Body.String())
	}

	archiveReq := httptest.NewRequest(http.MethodPost, "/drafts/"+draft.ID+"/archive", nil)
	archiveReq.SetPathValue("draftID", draft.ID)
	archiveRec := httptest.NewRecorder()
	serve(mgr, archiveRec, archiveReq)
	if archiveRec.Code != http.StatusOK {
		t.Fatalf("archive status = %d, body = %s", archiveRec.Code, archiveRec.Body.String())
	}

	// Already archived -> re-archiving is refused.
	reArchiveRec := httptest.NewRecorder()
	serve(mgr, reArchiveRec, archiveReq)
	if reArchiveRec.Code != http.StatusConflict {
		t.Fatalf("re-archive status = %d, body = %s", reArchiveRec.Code, reArchiveRec.Body.String())
	}
}

func TestArchiveDraft(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(t.Context(), "abandoned", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/drafts/"+draft.ID+"/archive", nil)
	req.SetPathValue("draftID", draft.ID)
	rec := httptest.NewRecorder()
	serve(mgr, rec, req)
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
