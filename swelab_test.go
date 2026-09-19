package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestSeedSWELab(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	seed, err := mwanachamaagency.SeedSWELab(ctx, mgr)
	if err != nil {
		t.Fatalf("SeedSWELab: %v", err)
	}

	if len(seed.Roles) != len(mwanachamaagency.SWELabPersonas) {
		t.Fatalf("expected %d roles, got %d", len(mwanachamaagency.SWELabPersonas), len(seed.Roles))
	}
	for _, name := range mwanachamaagency.SWELabPersonas {
		if _, ok := seed.Roles[name]; !ok {
			t.Fatalf("expected a seeded role for persona %q", name)
		}
	}

	if len(seed.Modules) != len(models.ModuleCatalog) {
		t.Fatalf("expected %d modules, got %d", len(models.ModuleCatalog), len(seed.Modules))
	}
	for _, entry := range models.ModuleCatalog {
		mod, ok := seed.Modules[entry.Code]
		if !ok {
			t.Fatalf("expected a seeded module for code %q", entry.Code)
		}
		if mod.Enabled != entry.Enabled {
			t.Fatalf("expected seeded module %q Enabled=%v, got %+v", entry.Code, entry.Enabled, mod)
		}
		if mod.Locked != entry.Locked {
			t.Fatalf("expected seeded module %q Locked=%v, got %+v", entry.Code, entry.Locked, mod)
		}
	}

	var reviewItem models.WorkItem
	for _, wi := range seed.WorkItems {
		if wi.Title == "Review pull request" {
			reviewItem = wi
		}
	}
	if reviewItem.ID == "" {
		t.Fatal("expected a \"Review pull request\" work item to have been seeded")
	}

	assignments, err := mgr.ListRoleAssignmentsForObjective(ctx, reviewItem.ObjectiveID, seed.Draft.ID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForObjective: %v", err)
	}
	if len(assignments) != 4 {
		t.Fatalf("expected all four RACI labels on the review item, got %d assignments", len(assignments))
	}
	byLabel := map[models.RACILabel]bool{}
	for _, a := range assignments {
		byLabel[a.Label] = true
	}
	for _, want := range []models.RACILabel{
		models.RACILabelResponsible, models.RACILabelAccountable,
		models.RACILabelConsulted, models.RACILabelInformed,
	} {
		if !byLabel[want] {
			t.Errorf("expected a %q assignment on the review item, got %+v", want, assignments)
		}
	}

	// The at-most-one-Accountable rule still holds against seeded data.
	otherRole := seed.Roles["business-analyst"]
	_, err = mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: seed.Draft.ID, RoleID: otherRole.ID, ObjectiveID: reviewItem.ObjectiveID, Label: models.RACILabelAccountable,
	})
	if !errors.Is(err, mwanachamaagency.ErrDuplicateAccountable) {
		t.Fatalf("expected ErrDuplicateAccountable adding a second accountable, got %v", err)
	}

	pub, err := mgr.PublishAgency(ctx, seed.Draft.ID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if pub.Version != 1 || pub.ContentHash == "" {
		t.Fatalf("unexpected first publication: %+v", pub)
	}
}
