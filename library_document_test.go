package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestPublishAgency_DocumentCarriesTheLibrary(t *testing.T) {
	ctx := context.Background()
	mgr, f, draftID := libraryDraftWithParts(t, ctx)

	pub, err := mgr.PublishAgency(ctx, draftID, "release-1", "production")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	doc := pub.Document
	if len(doc.Hazards) != 1 || doc.Hazards[0].Code != "H-1" || doc.Hazards[0].Title != "Perishable produce" || doc.Hazards[0].Description == "" {
		t.Fatalf("document hazards: %+v", doc.Hazards)
	}
	if len(doc.Threats) != 1 || doc.Threats[0].Code != "T-1" || doc.Threats[0].Title != "No buyer is listed" {
		t.Fatalf("document threats: %+v", doc.Threats)
	}
	if len(doc.Consequences) != 1 || doc.Consequences[0].Code != "Q-1" || doc.Consequences[0].Title != "Produce spoils" {
		t.Fatalf("document consequences: %+v", doc.Consequences)
	}
	if len(doc.RiskThreats) != 1 || doc.RiskThreats[0].RiskCode != f.risk.Code || doc.RiskThreats[0].ThreatCode != "T-1" {
		t.Fatalf("document threat links: %+v", doc.RiskThreats)
	}
	if len(doc.RiskConsequences) != 1 || doc.RiskConsequences[0].RiskCode != f.risk.Code || doc.RiskConsequences[0].ConsequenceCode != "Q-1" {
		t.Fatalf("document consequence links: %+v", doc.RiskConsequences)
	}
	if len(doc.Risks) != 1 || doc.Risks[0].HazardCode != "H-1" {
		t.Fatalf("document risks: %+v", doc.Risks)
	}

	read, err := mgr.GetPublication(ctx, pub.Version)
	if err != nil {
		t.Fatalf("GetPublication: %v", err)
	}
	if len(read.Document.Threats) != 1 || len(read.Document.RiskConsequences) != 1 || read.Document.Risks[0].HazardCode != "H-1" {
		t.Fatalf("the stored document lost the library: %+v", read.Document)
	}
}

func TestCreateDraft_FromATagRestoresTheLibrary(t *testing.T) {
	ctx := context.Background()
	mgr, f, draftID := libraryDraftWithParts(t, ctx)
	if _, err := mgr.PublishAgency(ctx, draftID, "release-1", "production"); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}

	revisit, err := mgr.CreateDraft(ctx, "revisit", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	for role, want := range map[string]string{"hazard": "H-1", "threat": "T-1", "consequence": "Q-1"} {
		got := f.rows(t, role, revisit.ID)
		if len(got) != 1 || got[0]["code"] != want || got[0]["title"] == "" {
			t.Fatalf("restored %s rows: %+v", role, got)
		}
	}
	if links := f.rows(t, "risk_threat", revisit.ID); len(links) != 1 || links[0]["threat_code"] != "T-1" {
		t.Fatalf("restored threat links: %+v", links)
	}
	if links := f.rows(t, "risk_consequence", revisit.ID); len(links) != 1 || links[0]["consequence_code"] != "Q-1" {
		t.Fatalf("restored consequence links: %+v", links)
	}
	if risks := f.rows(t, "risk", revisit.ID); len(risks) != 1 || risks[0]["hazard_code"] != "H-1" {
		t.Fatalf("restored risks: %+v", risks)
	}
}

func TestPublishAgency_LibraryChangeIsAChange(t *testing.T) {
	ctx := context.Background()
	mgr, f, draftID := libraryDraftWithParts(t, ctx)
	if _, err := mgr.PublishAgency(ctx, draftID, "release-1", "production"); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := mgr.PublishAgency(ctx, draftID, "release-2", "production"); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("unchanged republish = %v, want ErrNoChangesDetected", err)
	}

	edits := []struct {
		name string
		do   func() error
	}{
		{"a threat title", func() error {
			return f.db.Table(tableOf(t, f.s, "threat")).Where("draft_id = ? AND code = ?", draftID, "T-1").Update("title", "Nobody answers the listing").Error
		}},
		{"a hazard description", func() error {
			return f.db.Table(tableOf(t, f.s, "hazard")).Where("draft_id = ? AND code = ?", draftID, "H-1").Update("description", "Rots within days").Error
		}},
		{"a consequence title", func() error {
			return f.db.Table(tableOf(t, f.s, "consequence")).Where("draft_id = ? AND code = ?", draftID, "Q-1").Update("title", "Produce is thrown away").Error
		}},
		{"a risk hazard code", func() error {
			return f.db.Table(tableOf(t, f.s, "risk")).Where("id = ?", f.risk.ID).Update("hazard_code", "H-2").Error
		}},
		{"a threat link", func() error {
			return f.db.Table(tableOf(t, f.s, "risk_threat")).Where("draft_id = ?", draftID).Delete(map[string]any{}).Error
		}},
	}
	for i, e := range edits {
		if err := e.do(); err != nil {
			t.Fatalf("%s: %v", e.name, err)
		}
		tag := "edit-" + string(rune('a'+i))
		if _, err := mgr.PublishAgency(ctx, draftID, tag, "production"); err != nil {
			t.Fatalf("changing %s must be a change, got %v", e.name, err)
		}
	}
}
