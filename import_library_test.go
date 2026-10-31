package mwanachamaagency_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func oldPathwayDocument() models.AgencyDocument {
	return models.AgencyDocument{
		Agency: models.DocumentAgency{Name: "Old", Enabled: true},
		Risks: []models.DocumentRisk{
			{ID: "r1", Code: "RK-1", Title: "Unsold harvest", TopEvent: "A harvest goes unsold"},
			{ID: "r2", Code: "RK-2", Title: "Late payment", TopEvent: "Farmers are paid late"},
		},
		RiskPathways: []models.DocumentRiskPathway{
			{ID: "p1", Code: "RP-1", RiskID: "r1", Side: models.RiskSideThreat, Title: "No buyer is listed", Scenario: "Nobody asks"},
			{ID: "p2", Code: "RP-2", RiskID: "r2", Side: models.RiskSideThreat, Title: "No buyer is listed"},
			{ID: "p3", Code: "RP-3", RiskID: "r1", Side: models.RiskSideConsequence, Title: "Produce spoils"},
		},
		Controls: []models.DocumentControl{{ID: "c1", Code: "C-1", Text: "List a buyer first"}, {ID: "c2", Code: "C-2", Text: "Cold storage"}},
		ControlAttachments: []models.DocumentControlAttachment{
			{ControlCode: "C-1", TargetType: "risk_pathway", TargetCode: "RP-1"},
			{ControlCode: "C-1", TargetType: "risk_pathway", TargetCode: "RP-2"},
			{ControlCode: "C-2", TargetType: "risk_pathway", TargetCode: "RP-3"},
		},
	}
}

func TestImportDocument_OldPathwaysBecomeLibraryPartsAndLinks(t *testing.T) {
	ctx := context.Background()
	mgr, db, s := newTestManagerDB(t)
	f := libraryFixture{db: db, s: s}

	out, err := mgr.ImportDocument(ctx, "old", oldPathwayDocument())
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	draftID := out.Draft.ID

	threats := f.rows(t, "threat", draftID)
	if len(threats) != 1 || threats[0]["title"] != "No buyer is listed" {
		t.Fatalf("one threat expected for the shared title, got %+v", threats)
	}
	consequences := f.rows(t, "consequence", draftID)
	if len(consequences) != 1 || consequences[0]["title"] != "Produce spoils" {
		t.Fatalf("consequences: %+v", consequences)
	}
	threatCode, consequenceCode := threats[0]["code"], consequences[0]["code"]

	threatLinks := f.rows(t, "risk_threat", draftID)
	if len(threatLinks) != 2 {
		t.Fatalf("both risks must use the shared threat: %+v", threatLinks)
	}
	for _, l := range threatLinks {
		if l["threat_code"] != threatCode {
			t.Fatalf("link names the wrong threat: %+v", l)
		}
	}
	consequenceLinks := f.rows(t, "risk_consequence", draftID)
	if len(consequenceLinks) != 1 || consequenceLinks[0]["risk_code"] != "RK-1" || consequenceLinks[0]["consequence_code"] != consequenceCode {
		t.Fatalf("consequence links: %+v", consequenceLinks)
	}

	if left := f.rows(t, "risk_pathway", draftID); len(left) != 0 {
		t.Fatalf("pathways must be turned into parts, %d remain", len(left))
	}

	controls := controlsByCode(t, mgr, draftID)
	guarding := func(code string) []models.ControlAttachment {
		attached, err := mgr.ListControlAttachments(ctx, controls[code].ID)
		if err != nil {
			t.Fatalf("ListControlAttachments(%s): %v", code, err)
		}
		return attached
	}
	if a := guarding("C-1"); len(a) != 1 || a[0].TargetType != models.TargetTypeThreat || a[0].TargetCode != threatCode {
		t.Fatalf("C-1 must stand on the one threat: %+v", a)
	}
	if a := guarding("C-2"); len(a) != 1 || a[0].TargetType != models.TargetTypeConsequence || a[0].TargetCode != consequenceCode {
		t.Fatalf("C-2 must stand on the consequence: %+v", a)
	}
}

func TestImportDocument_ADocumentThatCarriesALibraryIsTakenAsItIs(t *testing.T) {
	ctx := context.Background()
	mgr, db, s := newTestManagerDB(t)
	f := libraryFixture{db: db, s: s}
	doc := oldPathwayDocument()
	doc.Threats = []models.DocumentThreat{{ID: "t1", Code: "T-9", Title: "No buyer is listed"}}
	doc.RiskThreats = []models.DocumentRiskThreat{{RiskCode: "RK-1", ThreatCode: "T-9"}}

	out, err := mgr.ImportDocument(ctx, "new", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	threats := f.rows(t, "threat", out.Draft.ID)
	if len(threats) != 1 || threats[0]["code"] != "T-9" {
		t.Fatalf("a document that carries a library is taken as it is: %+v", threats)
	}
	if links := f.rows(t, "risk_threat", out.Draft.ID); len(links) != 1 {
		t.Fatalf("links: %+v", links)
	}
}

func TestImportDocument_WithoutALibraryStillLoads(t *testing.T) {
	ctx := context.Background()
	mgr, db, s := newTestManagerDB(t)
	f := libraryFixture{db: db, s: s}
	doc := oldPathwayDocument()
	doc.RiskPathways, doc.ControlAttachments = nil, nil

	out, err := mgr.ImportDocument(ctx, "bare", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	for _, role := range []string{"hazard", "threat", "consequence", "risk_threat", "risk_consequence"} {
		if rows := f.rows(t, role, out.Draft.ID); len(rows) != 0 {
			t.Fatalf("%s rows invented: %+v", role, rows)
		}
	}
	if risks := f.rows(t, "risk", out.Draft.ID); len(risks) != 2 {
		t.Fatalf("risks: %+v", risks)
	}
}

func TestImportDocument_ImportedLibraryForks(t *testing.T) {
	ctx := context.Background()
	mgr, db, s := newTestManagerDB(t)
	f := libraryFixture{db: db, s: s}
	out, err := mgr.ImportDocument(ctx, "old", oldPathwayDocument())
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}

	fork, err := mgr.CreateDraft(ctx, "fork", out.Draft.ID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if len(f.rows(t, "threat", fork.ID)) != 1 || len(f.rows(t, "risk_threat", fork.ID)) != 2 {
		t.Fatalf("the imported library must fork")
	}
}
