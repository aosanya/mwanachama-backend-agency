package mwanachamaagency_test

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type libraryFixture struct {
	db   *gorm.DB
	s    *spec.Spec
	risk models.Risk
}

func (f libraryFixture) insert(t *testing.T, role string, row map[string]any) {
	t.Helper()
	row["id"] = role + "-" + row["draft_id"].(string) + "-" + row["key"].(string)
	delete(row, "key")
	row["created_at"] = models.NowRFC3339()
	if err := f.db.Table(tableOf(t, f.s, role)).Create(row).Error; err != nil {
		t.Fatalf("insert %s: %v", role, err)
	}
}

func (f libraryFixture) part(t *testing.T, role, draftID, code, title string) {
	t.Helper()
	row := map[string]any{
		"key": code, "code": code, "draft_id": draftID, "title": title,
		"description": title + " in full", "last_updated": models.NowRFC3339(), "deleted": false,
	}
	if role == "consequence" {
		row["hazard_code"] = "H-1"
	}
	f.insert(t, role, row)
}

func (f libraryFixture) link(t *testing.T, role, partColumn, draftID, riskCode, partCode string) {
	t.Helper()
	f.insert(t, role, map[string]any{
		"key": riskCode + partCode, "draft_id": draftID, "risk_code": riskCode, partColumn: partCode,
	})
}

func (f libraryFixture) rows(t *testing.T, role, draftID string) []map[string]any {
	t.Helper()
	var out []map[string]any
	if err := f.db.Table(tableOf(t, f.s, role)).Where("draft_id = ?", draftID).Order("created_at").Find(&out).Error; err != nil {
		t.Fatalf("read %s: %v", role, err)
	}
	return out
}

func libraryDraftWithParts(t *testing.T, ctx context.Context) (mwanachamaagency.AgencyManager, libraryFixture, string) {
	t.Helper()
	mgr, db, s := newTestManagerDB(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	risk, _, _ := seedRiskWithBothArms(t, mgr, ctx, o.DraftID)
	f := libraryFixture{db: db, s: s, risk: risk}
	f.part(t, "hazard", o.DraftID, "H-1", "Perishable produce")
	f.part(t, "threat", o.DraftID, "T-1", "No buyer is listed")
	f.part(t, "consequence", o.DraftID, "Q-1", "Produce spoils")
	f.link(t, "risk_threat", "threat_code", o.DraftID, risk.Code, "T-1")
	f.link(t, "risk_consequence", "consequence_code", o.DraftID, risk.Code, "Q-1")
	if err := db.Table(tableOf(t, s, "risk")).Where("id = ?", risk.ID).Update("hazard_code", "H-1").Error; err != nil {
		t.Fatalf("set hazard_code: %v", err)
	}
	return mgr, f, o.DraftID
}

func TestCreateDraft_ForksTheLibraryKeepingCodes(t *testing.T) {
	ctx := context.Background()
	mgr, f, source := libraryDraftWithParts(t, ctx)

	fork, err := mgr.CreateDraft(ctx, "fork", source, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	for role, want := range map[string]string{"hazard": "H-1", "threat": "T-1", "consequence": "Q-1"} {
		got := f.rows(t, role, fork.ID)
		if len(got) != 1 || got[0]["code"] != want || got[0]["draft_id"] != fork.ID {
			t.Fatalf("forked %s rows: %+v", role, got)
		}
		original := f.rows(t, role, source)
		if len(original) != 1 || original[0]["id"] == got[0]["id"] {
			t.Fatalf("fork must mint a new %s row and leave the source: %+v", role, original)
		}
		if role == "consequence" && got[0]["hazard_code"] != "H-1" {
			t.Fatalf("forked consequence lost its hazard_code: %+v", got[0])
		}
		if got[0]["title"] != original[0]["title"] || got[0]["description"] != original[0]["description"] {
			t.Fatalf("forked %s lost its words: %+v", role, got[0])
		}
	}

	threatLinks := f.rows(t, "risk_threat", fork.ID)
	if len(threatLinks) != 1 || threatLinks[0]["risk_code"] != f.risk.Code || threatLinks[0]["threat_code"] != "T-1" {
		t.Fatalf("forked threat links: %+v", threatLinks)
	}
	consequenceLinks := f.rows(t, "risk_consequence", fork.ID)
	if len(consequenceLinks) != 1 || consequenceLinks[0]["risk_code"] != f.risk.Code || consequenceLinks[0]["consequence_code"] != "Q-1" {
		t.Fatalf("forked consequence links: %+v", consequenceLinks)
	}

	forkedRisks := f.rows(t, "risk", fork.ID)
	if len(forkedRisks) != 1 || forkedRisks[0]["hazard_code"] != "H-1" {
		t.Fatalf("the risk must keep naming its hazard: %+v", forkedRisks)
	}
}

func TestCreateDraft_ForkOfADraftWithoutALibraryStillWorks(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	_, o, _ := seedPublishableAgency(t, mgr, ctx)

	if _, err := mgr.CreateDraft(ctx, "fork", o.DraftID, "draft"); err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
}
