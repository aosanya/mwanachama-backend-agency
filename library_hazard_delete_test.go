package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestDeleteHazard_RefusedWhileAConsequenceNamesIt(t *testing.T) {
	mgr := newLibraryManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	h, err := mgr.CreateHazard(ctx, models.Hazard{DraftID: o.DraftID, Title: "Fire"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, HazardCode: h.Code, Title: "Burns"})
	if err != nil {
		t.Fatal(err)
	}
	err = mgr.DeleteHazard(ctx, h.ID)
	if !errors.Is(err, mwanachamaagency.ErrLibraryPartInUse) {
		t.Fatalf("delete while a consequence names it err = %v", err)
	}
	if !strings.Contains(err.Error(), c.Code) {
		t.Fatalf("message %q does not name consequence %s", err.Error(), c.Code)
	}
	if err := mgr.DeleteConsequence(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if err := mgr.DeleteHazard(ctx, h.ID); err != nil {
		t.Fatalf("delete after the consequence is gone: %v", err)
	}
}

func TestHazardUsage_CountsRisksAndConsequences(t *testing.T) {
	mgr := newLibraryManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	h, err := mgr.CreateHazard(ctx, models.Hazard{DraftID: o.DraftID, Title: "Fire"})
	if err != nil {
		t.Fatal(err)
	}
	idle, err := mgr.CreateHazard(ctx, models.Hazard{DraftID: o.DraftID, Title: "Flood"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := mgr.CreateRisk(ctx, models.Risk{DraftID: o.DraftID, Title: "Blaze", TopEvent: "Ignition", HazardCode: h.Code})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, HazardCode: h.Code, Title: "Burns"})
	if err != nil {
		t.Fatal(err)
	}

	usage, err := mgr.(libraryUsageReader).HazardUsage(ctx, o.DraftID)
	if err != nil {
		t.Fatal(err)
	}
	byCode := map[string]mwanachamaagency.HazardUsage{}
	for _, u := range usage {
		byCode[u.Code] = u
	}
	got := byCode[h.Code]
	if len(usage) != 2 || got.Uses != 2 || len(got.RiskCodes) != 1 || got.RiskCodes[0] != r.Code || len(got.ConsequenceCodes) != 1 || got.ConsequenceCodes[0] != c.Code {
		t.Fatalf("unexpected usage %+v", usage)
	}
	if byCode[idle.Code].Uses != 0 {
		t.Fatalf("idle hazard usage %+v", byCode[idle.Code])
	}
	raw, _ := json.Marshal(got)
	if !strings.Contains(string(raw), `"used_in":2`) {
		t.Fatalf("json %s", raw)
	}
}
