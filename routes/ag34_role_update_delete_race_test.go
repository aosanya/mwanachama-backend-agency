package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func raceHarness(t *testing.T) (mwanachamaagency.AgencyManager, *httptest.Server) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	spec, err := mwanachamaagency.SpecFor("ag34racetest")
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
	if _, err := mgr.CreateAgency(context.Background(), models.Agency{Name: "AG34 Probe Co"}); err != nil {
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

func TestAG34_UpdatingARoleThatIsAlreadyGoneIsNotFound(t *testing.T) {
	mgr, srv := raceHarness(t)
	ctx := context.Background()

	d, err := mgr.CreateDraft(ctx, "deterministic", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if err := mgr.DeleteRole(ctx, role.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/roles/"+role.ID, strings.NewReader(`{"name":"renamed"}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("PATCH over a deleted role = %d, want 404 — a write that matched no row is not a success",
			resp.StatusCode)
	}
}

func TestAG34_ConcurrentUpdateAndDeleteAnswerOnlyOKOrNotFound(t *testing.T) {
	mgr, srv := raceHarness(t)
	ctx := context.Background()

	d, err := mgr.CreateDraft(ctx, "race", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	const total = 200
	for i := 0; i < total; i++ {
		role, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "bug-hunter"})
		if err != nil {
			t.Fatalf("CreateRole: %v", err)
		}

		var wg sync.WaitGroup
		var updateStatus, deleteStatus int
		wg.Add(2)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/roles/"+role.ID, strings.NewReader(`{"name":"renamed"}`))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("PATCH: %v", err)
				return
			}
			updateStatus = resp.StatusCode
			resp.Body.Close()
		}()
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/roles/"+role.ID, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("DELETE: %v", err)
				return
			}
			deleteStatus = resp.StatusCode
			resp.Body.Close()
		}()
		wg.Wait()

		if updateStatus != http.StatusOK && updateStatus != http.StatusNotFound {
			t.Fatalf("iteration %d: PATCH = %d, want 200 or 404", i, updateStatus)
		}
		if deleteStatus != http.StatusNoContent && deleteStatus != http.StatusNotFound {
			t.Fatalf("iteration %d: DELETE = %d, want 204 or 404", i, deleteStatus)
		}
	}
}
