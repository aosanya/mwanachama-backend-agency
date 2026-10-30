package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG62_TwoModulesShareTheSameCodeInOneDraft(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	first, err := seededModule(mgr, ctx, o.DraftID, "auth")
	if err != nil {
		t.Fatalf("precondition: seeded module: %v", err)
	}
	if _, err := mgr.CreateModule(ctx, models.Module{DraftID: o.DraftID, Code: "auth", Name: "Auth again"}); err == nil {
		t.Fatalf("precondition: a sequential second Module with code %q should be refused with ErrDuplicateModule", first.Code)
	} else if err.Error() == "" {
		t.Fatalf("precondition: unexpected nil-message error: %v", err)
	}

	insertErr := db.WithContext(ctx).Table(tableOf(t, s, "module")).Create(map[string]any{
		"id":           "concurrent-module",
		"draft_id":     o.DraftID,
		"code":         "auth",
		"name":         "Auth (concurrent writer)",
		"created_at":   models.NowRFC3339(),
		"last_updated": models.NowRFC3339(),
	}).Error

	mods, err := mgr.ListModules(ctx, o.DraftID, mwanachamaagency.ModuleFilter{})
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	count := 0
	for _, m := range mods {
		if m.Code == "auth" {
			count++
		}
	}

	if insertErr == nil && count > 1 {
		t.Errorf("AG62: a second Module row with code %q, written as a concurrent CreateModule writes it after both counted zero, was accepted by the store (insert error %v) and the draft now holds %d Modules with that code; want the store to refuse the second row, the same way CreateModule's own pre-check means to", first.Code, insertErr, count)
	}
}
