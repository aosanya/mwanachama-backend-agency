package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestPutModuleMounts_NamesTheMarketsAndRefusesADuplicate(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	mods, err := mgr.ListModules(t.Context(), draftID, mwanachamaagency.ModuleFilter{Code: "market"})
	if err != nil || len(mods) != 1 {
		t.Fatalf("ListModules: %v %+v", err, mods)
	}
	put := func(body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		serve(mgr, rec, httptest.NewRequest(http.MethodPut, "/modules/"+mods[0].ID+"/mounts", strings.NewReader(body)))
		return rec
	}

	ok := put(`{"mounts":[{"name":"Inputs"},{"name":"Harvest","icon":"wheat"}]}`)
	if ok.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", ok.Code, ok.Body.String())
	}
	mounts, err := models.ParseModuleMounts(mustModule(t, mgr, mods[0].ID).Mounts)
	if err != nil || len(mounts) != 2 || mounts[0].Key != "market" || mounts[1].Key != "harvest" {
		t.Fatalf("mounts = %+v %v", mounts, err)
	}

	dup := put(`{"mounts":[{"name":"Inputs"},{"name":"INPUTS"}]}`)
	if dup.Code != http.StatusBadRequest {
		t.Fatalf("duplicate status = %d, body = %s", dup.Code, dup.Body.String())
	}
}

func mustModule(t *testing.T, mgr mwanachamaagency.AgencyManager, id string) models.Module {
	t.Helper()
	mod, err := mgr.GetModule(t.Context(), id)
	if err != nil {
		t.Fatalf("GetModule: %v", err)
	}
	return mod
}
