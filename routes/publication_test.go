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

func TestPublishAgencyAndGetPublication(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "NeoLabHQ"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":""}`))
	rec := httptest.NewRecorder()
	routes.PublishAgency(mgr)(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var pub models.AgencyPublication
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pub.Version != 1 {
		t.Fatalf("expected version 1, got %d", pub.Version)
	}

	// Republishing unchanged content is refused.
	req2 := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":""}`))
	rec2 := httptest.NewRecorder()
	routes.PublishAgency(mgr)(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("re-publish status = %d, body = %s", rec2.Code, rec2.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/publications/1", nil)
	getReq.SetPathValue("version", "1")
	getRec := httptest.NewRecorder()
	routes.GetPublication(mgr)(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/publications", nil)
	listRec := httptest.NewRecorder()
	routes.ListPublications(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodPatch, "/publications/1", strings.NewReader(`{"status":"archived"}`))
	statusReq.SetPathValue("version", "1")
	statusRec := httptest.NewRecorder()
	routes.UpdatePublicationStatus(mgr)(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", statusRec.Code, statusRec.Body.String())
	}
}

func TestGetPublication_BadVersion(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/publications/not-a-number", nil)
	req.SetPathValue("version", "not-a-number")
	rec := httptest.NewRecorder()
	routes.GetPublication(mgr)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
