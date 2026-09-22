package routes_test

import (
	"context"
	"fmt"
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

// TestPinsAG34_UpdateRoleCanReport200OverARoleDeleteConcurrentlyRemoved
// pins board row AG34: PATCH /roles/{id} (UpdateRole) reads the row, checks
// it's draft-scoped, then writes a targeted column update with no lock
// held across that gap — the same TOCTOU shape as taskmanager's W19/W20/
// W21 and assetmanager's A14. DELETE /roles/{id} (DeleteRole) can run its
// own read-check-delete in the same window, so the two race: UpdateRole's
// Updates(...).Where("id = ?", id) matches zero rows once DeleteRole has
// already committed, but GORM reports no error for that — UpdateRole
// returns 200 with a "successfully renamed" Role that no longer exists.
//
// This test pins the CURRENT (broken) behaviour: it asserts the race is
// reproducible at all (staleUpdateSucceeded > 0). Once UpdateRole/
// DeleteRole share a lock (e.g. both wrapped in one transaction that
// re-checks existence immediately before the write), this assertion
// should go red — at that point either replace it with an assertion that
// UpdateRole reliably returns 404/409 instead of 200 for the raced case,
// or delete this test if the fix already has its own coverage.
func TestPinsAG34_UpdateRoleCanReport200OverARoleDeleteConcurrentlyRemoved(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	// A pooled sqlite :memory: connection hands concurrent goroutines
	// different anonymous in-memory databases unless pinned to one
	// connection (see mwanachama-backend-git/CLAUDE.md's identical note).
	sqlDB.SetMaxOpenConns(1)
	tables := mwanachamaagency.DefaultTableNames("ag34_race_test")
	if err := mwanachamaagency.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, tables)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	if _, err := mgr.CreateAgency(context.Background(), models.Agency{Name: "AG34 Probe Co"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}

	mux := http.NewServeMux()
	for _, rt := range routes.Routes(mgr, routes.ResourceNames{}) {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	const total = 200
	staleUpdateSucceeded := 0
	for i := 0; i < total; i++ {
		// AGD-006 (AG33-35) removed "live" as a fork type — "" now means
		// "start empty", which is what this loop wants each iteration.
		d, err := mgr.CreateDraft(context.Background(), fmt.Sprintf("d%d", i), "", "")
		if err != nil {
			t.Fatalf("CreateDraft: %v", err)
		}
		role, err := mgr.CreateRole(context.Background(), models.Role{DraftID: d.ID, Name: "bug-hunter"})
		if err != nil {
			t.Fatalf("CreateRole: %v", err)
		}

		var wg sync.WaitGroup
		var updateStatus int
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
			resp.Body.Close()
		}()
		wg.Wait()

		if updateStatus != http.StatusOK {
			continue
		}
		// UpdateRole reported success. Is the role actually still there?
		getReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/roles/"+role.ID, nil)
		resp, err := http.DefaultClient.Do(getReq)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		code := resp.StatusCode
		resp.Body.Close()
		if code == http.StatusNotFound {
			staleUpdateSucceeded++
		}
	}

	t.Logf("UpdateRole returned 200 concurrently with DeleteRole succeeding, and the role "+
		"no longer exists afterward: %d/%d iterations", staleUpdateSucceeded, total)
	if staleUpdateSucceeded == 0 {
		t.Fatal("expected at least one iteration where UpdateRole reported 200 for a role " +
			"DeleteRole had already removed (AG34) — if this never reproduces, the race may " +
			"have been closed; replace this test with one asserting the fixed behaviour")
	}
}
