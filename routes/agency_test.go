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

func TestCreateAndGetAgency(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/agency", strings.NewReader(`{"name":"NeoLabHQ"}`))
	rec := httptest.NewRecorder()
	routes.CreateAgency(mgr)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Agency
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID == "" || out.Name != "NeoLabHQ" {
		t.Fatalf("unexpected agency: %+v", out)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/agency", nil)
	getRec := httptest.NewRecorder()
	routes.GetAgency(mgr)(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}
}

func TestGetAgency_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/agency", nil)
	rec := httptest.NewRecorder()
	routes.GetAgency(mgr)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateAgency(t *testing.T) {
	mgr := newTestManager(t)
	createReq := httptest.NewRequest(http.MethodPost, "/agency", strings.NewReader(`{"name":"NeoLabHQ"}`))
	createRec := httptest.NewRecorder()
	routes.CreateAgency(mgr)(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/agency", strings.NewReader(`{"name":"Renamed","mission":"New mission.","vision":"New vision."}`))
	patchRec := httptest.NewRecorder()
	routes.UpdateAgency(mgr)(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", patchRec.Code, patchRec.Body.String())
	}
	var out models.Agency
	if err := json.Unmarshal(patchRec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Name != "Renamed" || out.Mission != "New mission." || out.Vision != "New vision." {
		t.Fatalf("unexpected updated agency: %+v", out)
	}
}

func TestUpdateAgency_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPatch, "/agency", strings.NewReader(`{"name":"Nobody"}`))
	rec := httptest.NewRecorder()
	routes.UpdateAgency(mgr)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
