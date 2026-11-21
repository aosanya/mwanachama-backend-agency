package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func kelpHomeWithRoles(roles ...string) []models.DocumentHomeNode {
	return []models.DocumentHomeNode{{Key: "a", Title: "A", Roles: roles, Target: &models.HomeTarget{Type: models.HomeTargetFiles}}}
}

func kelpDocument(roles []models.DocumentRole, home []models.DocumentHomeNode) models.AgencyDocument {
	return models.AgencyDocument{Agency: models.DocumentAgency{Name: "Kelp co-op"}, Roles: roles, Home: home}
}

func TestImportDocument_ARoleOnTheHomeMayBeNamedByItsNameAndTheNameIsKeptAsWritten(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	doc := kelpDocument([]models.DocumentRole{{ID: "r1", Code: "R-1", Name: "Diver"}}, kelpHomeWithRoles("diver"))
	out, err := mgr.ImportDocument(ctx, "kelp", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if got := homeKeyed(homeRowsOf(t, mgr, ctx, out.Draft.ID), "a").Roles; len(got) != 1 || got[0] != "diver" {
		t.Fatalf("roles = %v, want the name kept as the file wrote it", got)
	}
}

func TestImportDocument_ARoleNameThatMatchesNothingOrTwoRolesRefusesTheWholeFile(t *testing.T) {
	roles := []models.DocumentRole{{ID: "r1", Code: "R-1", Name: "Diver"}, {ID: "r2", Code: "R-2", Name: "Skipper"}, {ID: "r3", Code: "R-3", Name: "skipper"}}
	for name, wanted := range map[string]string{"no role named so": "Boatswain", "two roles named so": "Skipper"} {
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t)
			if _, err := mgr.ImportDocument(context.Background(), "kelp", kelpDocument(roles, kelpHomeWithRoles(wanted))); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
				t.Fatalf("ImportDocument = %v, want ErrInvalidHomeEntry", err)
			}
		})
	}
}

func TestImportDocument_ARoleCodeStillWinsOverARoleWithThatNameAsItsText(t *testing.T) {
	roles := []models.DocumentRole{{ID: "r1", Code: "R-1", Name: "R-2"}, {ID: "r2", Code: "R-2", Name: "Skipper"}}
	if _, err := newTestManager(t).ImportDocument(context.Background(), "kelp", kelpDocument(roles, kelpHomeWithRoles("R-2"))); err != nil {
		t.Fatalf("ImportDocument = %v, want the code R-2 resolved first", err)
	}
}

func TestCreateHomeEntryAndSetHome_ARoleMayBeNamedByItsNameAndAnAmbiguousNameIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, g, _, _, _ := seedLiveAgency(t, mgr, ctx)
	draftID := g.DraftID
	for _, name := range []string{"Diver", "Skipper"} {
		if _, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: name}); err != nil {
			t.Fatalf("CreateRole %s: %v", name, err)
		}
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "a", Title: "A", Roles: []string{"diver"}}); err != nil {
		t.Fatalf("CreateHomeEntry with a role name: %v", err)
	}
	if _, err := mgr.CreateHomeEntry(ctx, models.HomeEntry{DraftID: draftID, Key: "b", Title: "B", Roles: []string{"Boatswain"}}); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("an unknown role name = %v, want ErrInvalidHomeEntry", err)
	}
	if _, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "Diver"}); err != nil {
		t.Fatalf("CreateRole second Diver: %v", err)
	}
	if _, err := mgr.SetHome(ctx, draftID, []map[string]any{{"key": "c", "title": "C", "roles": []string{"Diver"}, "target": map[string]any{"type": "files"}}}); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("a role name two roles share = %v, want ErrInvalidHomeEntry", err)
	}
}

func TestImportDocument_ARoleTheFileMarksDeletedIsNotARoleTheHomeMayName(t *testing.T) {
	roles := []models.DocumentRole{{ID: "r1", Code: "R-1", Name: "Diver"}, {ID: "r2", Code: "R-2", Name: "Retired", Deleted: true}}
	mgr := newTestManager(t)
	for _, wanted := range []string{"R-2", "Retired"} {
		if _, err := mgr.ImportDocument(context.Background(), "kelp", kelpDocument(roles, kelpHomeWithRoles(wanted))); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
			t.Fatalf("naming %q: ImportDocument = %v, want ErrInvalidHomeEntry", wanted, err)
		}
	}
}

func TestImportDocument_ATargetBesideAnEmptyEntriesListRefusesTheWholeFile(t *testing.T) {
	home := []models.DocumentHomeNode{{Key: "a", Title: "A", Entries: []models.DocumentHomeNode{}, Target: &models.HomeTarget{Type: models.HomeTargetFiles}}}
	if _, err := newTestManager(t).ImportDocument(context.Background(), "kelp", kelpDocument(nil, home)); !errors.Is(err, mwanachamaagency.ErrInvalidHomeEntry) {
		t.Fatalf("ImportDocument = %v, want ErrInvalidHomeEntry", err)
	}
}
