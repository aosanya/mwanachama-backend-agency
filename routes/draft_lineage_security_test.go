package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG49_OpenHole_DraftStartedFromNothingRecordsACallerChosenForkSource(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)

	publish := httptest.NewRecorder()
	serve(mgr, publish, httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`","tag":"v1","channel":"production"}`)))
	if publish.Code != http.StatusCreated {
		t.Fatalf("precondition: publishing tag v1 answered %d %s", publish.Code, publish.Body.String())
	}

	rec := httptest.NewRecorder()
	serve(mgr, rec, httptest.NewRequest(http.MethodPost, "/drafts", strings.NewReader(`{"description":"blank","forked_from_type":"","forked_from_id":"v1"}`)))
	if rec.Code >= 400 && rec.Code < 500 {
		return
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /drafts answered %d %s", rec.Code, rec.Body.String())
	}
	var created models.Draft
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created draft: %v", err)
	}

	get := httptest.NewRecorder()
	serve(mgr, get, httptest.NewRequest(http.MethodGet, "/drafts/"+created.ID, nil))
	var stored models.Draft
	if err := json.Unmarshal(get.Body.Bytes(), &stored); err != nil {
		t.Fatalf("GET /drafts/%s answered %d %s", created.ID, get.Code, get.Body.String())
	}
	if stored.ForkedFromType == "" && stored.ForkedFromID != "" {
		t.Errorf("AG49 (P2) open hole: a draft started from nothing reads forked_from_id=%q because the request body said so — got {forked_from_type:%q, forked_from_id:%q}, want forked_from_id \"\" (or the request refused 400)", stored.ForkedFromID, stored.ForkedFromType, stored.ForkedFromID)
	}
}
