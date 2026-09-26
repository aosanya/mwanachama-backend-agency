package routes_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func ag46Harness(t *testing.T) (mwanachamaagency.AgencyManager, *httptest.Server) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	spec, err := mwanachamaagency.SpecFor("ag46test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, spec); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, spec)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	if _, err := mgr.CreateAgency(context.Background(), models.Agency{Name: "AG46 Probe Co"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	mux := http.NewServeMux()
	for _, rt := range routes.Routes(mgr) {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return mgr, srv
}

func TestAG46_PinsKnownDefect_ArchivedDraftRoleStillEditableOverHTTP(t *testing.T) {
	mgr, srv := ag46Harness(t)
	ctx := context.Background()

	draft, err := mgr.CreateDraft(ctx, "probe draft", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draft.ID, Name: "Engineer"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	archiveReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/drafts/"+draft.ID+"/archive", nil)
	archiveResp, err := http.DefaultClient.Do(archiveReq)
	if err != nil {
		t.Fatalf("archive POST: %v", err)
	}
	archiveResp.Body.Close()
	if archiveResp.StatusCode != http.StatusOK {
		t.Fatalf("archive status = %d, want 200", archiveResp.StatusCode)
	}

	patchReq, _ := http.NewRequest(http.MethodPatch, srv.URL+"/roles/"+role.ID, strings.NewReader(`{"name":"renamed after archive"}`))
	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatalf("PATCH: %v", err)
	}
	patchBody, _ := io.ReadAll(patchResp.Body)
	patchResp.Body.Close()

	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("AG46 regression: PATCH on an archived-draft role = %d, want the current (broken) 200 — "+
			"if this now fails with 409/ErrDraftNotOpen, AG46 is fixed and this test should be inverted", patchResp.StatusCode)
	}

	var patched models.Role
	if err := json.Unmarshal(patchBody, &patched); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if patched.Name != "renamed after archive" {
		t.Fatalf("AG46 regression: expected the archived-draft row to still have been renamed (current broken behaviour), got %q", patched.Name)
	}

	deleteReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/roles/"+role.ID, nil)
	deleteResp, err := http.DefaultClient.Do(deleteReq)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	deleteResp.Body.Close()
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("AG46 regression: DELETE on an archived-draft role = %d, want the current (broken) 204", deleteResp.StatusCode)
	}
}
