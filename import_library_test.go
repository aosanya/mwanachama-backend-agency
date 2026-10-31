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

func hazardousPathwayDocument() models.AgencyDocument {
	return models.AgencyDocument{
		Agency: models.DocumentAgency{Name: "Old", Enabled: true},
		Hazards: []models.DocumentHazard{
			{ID: "h1", Code: "H-1", Title: "Fire"},
			{ID: "h2", Code: "H-2", Title: "Flood"},
		},
		Risks: []models.DocumentRisk{
			{ID: "r1", Code: "RK-1", Title: "Barn fire", TopEvent: "Barn burns", HazardCode: "H-1"},
			{ID: "r2", Code: "RK-2", Title: "River flood", TopEvent: "River rises", HazardCode: "H-2"},
			{ID: "r3", Code: "RK-3", Title: "Second barn fire", TopEvent: "Barn burns again", HazardCode: "H-1"},
			{ID: "r4", Code: "RK-4", Title: "Loose risk", TopEvent: "Something"},
		},
		RiskPathways: []models.DocumentRiskPathway{
			{ID: "p1", Code: "RP-1", RiskID: "r1", Side: models.RiskSideConsequence, Title: "Produce spoils"},
			{ID: "p2", Code: "RP-2", RiskID: "r2", Side: models.RiskSideConsequence, Title: "Produce spoils"},
			{ID: "p3", Code: "RP-3", RiskID: "r3", Side: models.RiskSideConsequence, Title: "Produce spoils"},
			{ID: "p4", Code: "RP-4", RiskID: "r4", Side: models.RiskSideConsequence, Title: "Stock lost"},
		},
		Controls: []models.DocumentControl{{ID: "c1", Code: "C-1", Text: "Cold storage"}, {ID: "c2", Code: "C-2", Text: "Insure stock"}},
		ControlAttachments: []models.DocumentControlAttachment{
			{ControlCode: "C-1", TargetType: "risk_pathway", TargetCode: "RP-1"},
			{ControlCode: "C-1", TargetType: "risk_pathway", TargetCode: "RP-2"},
			{ControlCode: "C-2", TargetType: "risk_pathway", TargetCode: "RP-4"},
		},
	}
}

func consequenceOf(t *testing.T, rows []map[string]any, hazard, title string) map[string]any {
	t.Helper()
	for _, r := range rows {
		if r["hazard_code"] == hazard && r["title"] == title {
			return r
		}
	}
	t.Fatalf("no consequence %q under hazard %q in %+v", title, hazard, rows)
	return nil
}

func TestImportDocument_PathwayConsequenceTakesTheHazardOfItsRisk(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	f := libraryFixture{db: db, s: s}
	doc := hazardousPathwayDocument()
	doc.RiskPathways = doc.RiskPathways[:1]
	doc.ControlAttachments = nil

	out, err := mgr.ImportDocument(context.Background(), "old", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	rows := f.rows(t, "consequence", out.Draft.ID)
	if len(rows) != 1 || rows[0]["hazard_code"] != "H-1" {
		t.Fatalf("consequence must take H-1 from its risk: %+v", rows)
	}
}

func TestImportDocument_SharedPathwayTitleSplitsPerHazard(t *testing.T) {
	ctx := context.Background()
	mgr, db, s := newTestManagerDB(t)
	f := libraryFixture{db: db, s: s}

	out, err := mgr.ImportDocument(ctx, "old", hazardousPathwayDocument())
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	draftID := out.Draft.ID
	rows := f.rows(t, "consequence", draftID)
	if len(rows) != 3 {
		t.Fatalf("Produce spoils under H-1 and H-2 plus Stock lost: %+v", rows)
	}
	fire := consequenceOf(t, rows, "H-1", "Produce spoils")["code"]
	flood := consequenceOf(t, rows, "H-2", "Produce spoils")["code"]
	if fire == flood {
		t.Fatalf("one consequence per hazard, got %v twice", fire)
	}

	linked := map[string]string{}
	var fireLinks int
	for _, l := range f.rows(t, "risk_consequence", draftID) {
		linked[l["risk_code"].(string)] = l["consequence_code"].(string)
		if l["consequence_code"] == fire {
			fireLinks++
		}
	}
	if linked["RK-1"] != fire || linked["RK-3"] != fire || linked["RK-2"] != flood || fireLinks != 2 {
		t.Fatalf("each risk must link the consequence of its own hazard: %+v", linked)
	}

	controls := controlsByCode(t, mgr, draftID)
	attached, err := mgr.ListControlAttachments(ctx, controls["C-1"].ID)
	if err != nil {
		t.Fatalf("ListControlAttachments: %v", err)
	}
	targets := map[string]bool{}
	for _, a := range attached {
		if a.TargetType != models.TargetTypeConsequence {
			t.Fatalf("attachment must target a consequence: %+v", a)
		}
		targets[a.TargetCode] = true
	}
	if len(attached) != 2 || !targets[fire.(string)] || !targets[flood.(string)] {
		t.Fatalf("C-1 must stand on both new consequences: %+v", attached)
	}
}

func TestImportDocument_PathwayOfRiskWithoutHazardImportsWithoutOne(t *testing.T) {
	ctx := context.Background()
	mgr, db, s := newTestManagerDB(t)
	f := libraryFixture{db: db, s: s}

	out, err := mgr.ImportDocument(ctx, "old", hazardousPathwayDocument())
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	stock := consequenceOf(t, f.rows(t, "consequence", out.Draft.ID), "", "Stock lost")
	if hazards := f.rows(t, "hazard", out.Draft.ID); len(hazards) != 2 {
		t.Fatalf("no hazard may be invented: %+v", hazards)
	}
	controls := controlsByCode(t, mgr, out.Draft.ID)
	attached, err := mgr.ListControlAttachments(ctx, controls["C-2"].ID)
	if err != nil {
		t.Fatalf("ListControlAttachments: %v", err)
	}
	if len(attached) != 1 || attached[0].TargetCode != stock["code"] {
		t.Fatalf("C-2 must follow the hazardless consequence: %+v", attached)
	}
}

func TestImportDocument_LibraryConsequenceOfSameTitleAndHazardWinsOverPathway(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	f := libraryFixture{db: db, s: s}
	doc := hazardousPathwayDocument()
	doc.RiskPathways = doc.RiskPathways[:2]
	doc.ControlAttachments = nil
	doc.Consequences = []models.DocumentConsequence{{ID: "q1", Code: "Q-9", Title: "produce spoils", HazardCode: "H-1"}}

	out, err := mgr.ImportDocument(context.Background(), "old", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	rows := f.rows(t, "consequence", out.Draft.ID)
	if len(rows) != 2 {
		t.Fatalf("Q-9 plus one new consequence for H-2: %+v", rows)
	}
	links := map[string]any{}
	for _, l := range f.rows(t, "risk_consequence", out.Draft.ID) {
		links[l["risk_code"].(string)] = l["consequence_code"]
	}
	if links["RK-1"] != "Q-9" || links["RK-2"] == "Q-9" || links["RK-2"] == nil {
		t.Fatalf("the pathway only adds its link to Q-9: %+v", links)
	}
}

func TestImportDocument_ConsequenceWithHazardCodeImportsAsWritten(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	f := libraryFixture{db: db, s: s}
	doc := hazardousPathwayDocument()
	doc.RiskPathways, doc.ControlAttachments = nil, nil
	doc.Consequences = []models.DocumentConsequence{
		{ID: "q1", Code: "Q-1", Title: "Produce spoils", HazardCode: "H-1"},
		{ID: "q2", Code: "Q-2", Title: "Produce spoils", HazardCode: "H-2"},
	}
	doc.RiskConsequences = []models.DocumentRiskConsequence{{RiskCode: "RK-1", ConsequenceCode: "Q-1"}}

	out, err := mgr.ImportDocument(context.Background(), "new", doc)
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	rows := f.rows(t, "consequence", out.Draft.ID)
	if len(rows) != 2 {
		t.Fatalf("both consequences, one per hazard: %+v", rows)
	}
	if consequenceOf(t, rows, "H-1", "Produce spoils")["code"] != "Q-1" || consequenceOf(t, rows, "H-2", "Produce spoils")["code"] != "Q-2" {
		t.Fatalf("hazard_code must survive as written: %+v", rows)
	}
}
