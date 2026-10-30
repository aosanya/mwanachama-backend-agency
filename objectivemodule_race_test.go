package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG61_TwoObjectiveModulesHoldTheSameObjectiveAndModule(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	mod, err := seededModule(mgr, ctx, o.DraftID, "auth")
	if err != nil {
		t.Fatalf("precondition: CreateModule: %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{
		DraftID: o.DraftID, ObjectiveID: o.ID, ModuleID: mod.ID, Rationale: "needs auth",
	}); err != nil {
		t.Fatalf("precondition: CreateObjectiveModule: %v", err)
	}

	insertErr := db.WithContext(ctx).Table(tableOf(t, s, "objective_module")).Create(map[string]any{
		"id":           "concurrent-objective-module",
		"draft_id":     o.DraftID,
		"objective_id": o.ID,
		"module_id":    mod.ID,
		"rationale":    "a concurrent writer's reason",
		"created_at":   models.NowRFC3339(),
		"last_updated": models.NowRFC3339(),
	}).Error

	held, err := mgr.ListObjectiveModules(ctx, o.DraftID, mwanachamaagency.ObjectiveModuleFilter{ObjectiveID: o.ID})
	if err != nil {
		t.Fatalf("ListObjectiveModules: %v", err)
	}

	if insertErr == nil && len(held) > 1 {
		t.Errorf("AG61: a second objective_module row linking objective %s to module %s, written as a concurrent CreateObjectiveModule writes it after both counted zero, was accepted by the store (insert error %v) and the objective now holds %d links to the same module; want the store to refuse the second row, the same way CreateObjectiveModule's own pre-check means to", o.ID, mod.ID, insertErr, len(held))
	}
}
