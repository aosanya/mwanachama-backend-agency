package mwanachamaagency_test

import (
	"context"
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

	techLead := seed.Roles["tech-lead"]
	if len(seed.WorkItems) != 3 {
		t.Fatalf("expected three seeded work items, got %d", len(seed.WorkItems))
	}
	for _, wi := range seed.WorkItems {
		if wi.AccountableRoleID != techLead.ID {
			t.Errorf("expected the tech lead accountable for %q, got %q", wi.Title, wi.AccountableRoleID)
		}
	}
	if reviewItem.AccountableRoleID != techLead.ID {
		t.Errorf("expected the review item to name the tech lead as accountable, got %+v", reviewItem)
	}

	if len(seed.WorkflowObjectives) != len(seed.Objectives) {
		t.Fatalf("expected the workflow linked to each of the %d objectives, got %+v", len(seed.Objectives), seed.WorkflowObjectives)
	}
	links, err := mgr.ListWorkflowObjectives(ctx, seed.Draft.ID, mwanachamaagency.WorkflowObjectiveFilter{})
	if err != nil || len(links) != len(seed.Objectives) {
		t.Fatalf("ListWorkflowObjectives: %v, %+v", err, links)
	}

	pub, err := mgr.PublishAgency(ctx, seed.Draft.ID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if pub.Version != 1 || pub.ContentHash == "" {
		t.Fatalf("unexpected first publication: %+v", pub)
	}
}
