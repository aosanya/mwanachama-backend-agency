package models

import (
	"encoding/json"
	"errors"
	"testing"
)

func reefParts() HomeParts {
	return HomeParts{
		Records:   []HomePart{{Code: "RC-1", Name: "Corals"}, {Code: "RC-2", Name: "Tide log"}, {Code: "RC-3", Name: "Dives"}, {Code: "RC-4", Name: "dives"}},
		Workflows: []HomePart{{Code: "W-1", Name: "Survey"}, {Code: "W-2", Name: "Repair"}},
		Steps: []HomeStepPart{
			{Code: "WI-1", Name: "Count corals", WorkflowCode: "W-1"},
			{Code: "WI-2", Name: "Review", WorkflowCode: "W-1"},
			{Code: "WI-3", Name: "Review", WorkflowCode: "W-2"},
			{Code: "WI-4", Name: "Sign off", WorkflowCode: "W-2"},
			{Code: "WI-5", Name: "Sign off", WorkflowCode: "W-2"},
		},
		Roles: []HomePart{{Code: "R-1", Name: "Diver"}, {Code: "R-2", Name: "Skipper"}, {Code: "R-3", Name: "Skipper"}},
	}
}

func TestResolveHomePart_ACodeIsMatchedBeforeAName(t *testing.T) {
	parts := []HomePart{{Code: "RC-1", Name: "RC-2"}, {Code: "RC-2", Name: "Other"}}
	got, err := ResolveHomePart(parts, "RC-2")
	if err != nil || got.Code != "RC-2" || got.Name != "Other" {
		t.Fatalf("RC-2 = %+v, %v, want the part whose code is RC-2", got, err)
	}
}

func TestResolveHomePart_ANameIsMatchedWhateverItsCaseAndPadding(t *testing.T) {
	got, err := ResolveHomePart(reefParts().Records, "  tide LOG ")
	if err != nil || got.Code != "RC-2" {
		t.Fatalf("tide LOG = %+v, %v, want RC-2", got, err)
	}
}

func TestResolveHomePart_ANameThatMatchesNothingIsRefused(t *testing.T) {
	if _, err := ResolveHomePart(reefParts().Records, "Anchors"); !errors.Is(err, ErrHomeRefUnmatched) {
		t.Fatalf("Anchors = %v, want ErrHomeRefUnmatched", err)
	}
}

func TestResolveHomePart_ANameThatMatchesTwoPartsIsRefused(t *testing.T) {
	if _, err := ResolveHomePart(reefParts().Records, "Dives"); !errors.Is(err, ErrHomeRefAmbiguous) {
		t.Fatalf("Dives = %v, want ErrHomeRefAmbiguous", err)
	}
}

func TestResolveHomePart_ABlankNameNeverMatchesABlankPart(t *testing.T) {
	if _, err := ResolveHomePart([]HomePart{{Code: "R-1"}}, " "); !errors.Is(err, ErrHomeRefUnmatched) {
		t.Fatalf("blank = %v, want ErrHomeRefUnmatched", err)
	}
}

func TestHomePartsStep_ANameIsLookedUpAmongTheStepsOfItsWorkflowOnly(t *testing.T) {
	parts := reefParts()
	first, err := parts.Step("W-1", "Review")
	if err != nil || first.Code != "WI-2" {
		t.Fatalf("Review in W-1 = %+v, %v, want WI-2", first, err)
	}
	second, err := parts.Step("W-2", "review")
	if err != nil || second.Code != "WI-3" {
		t.Fatalf("review in W-2 = %+v, %v, want WI-3", second, err)
	}
}

func TestHomePartsStep_ANameTwoStepsOfTheWorkflowShareIsRefused(t *testing.T) {
	if _, err := reefParts().Step("W-2", "Sign off"); !errors.Is(err, ErrHomeRefAmbiguous) {
		t.Fatalf("Sign off = %v, want ErrHomeRefAmbiguous", err)
	}
}

func TestHomePartsStep_ACodeIsFoundInAnyWorkflowAsItWasBeforeNames(t *testing.T) {
	got, err := reefParts().Step("W-1", "WI-4")
	if err != nil || got.Code != "WI-4" {
		t.Fatalf("WI-4 = %+v, %v, want the step whose code is WI-4", got, err)
	}
}

func homeOver(entries ...DocumentHomeNode) []DocumentHomeNode { return entries }

func TestCheckHomeReferences_EveryKindOfReferenceIsResolvedOrTheWholeHomeIsRefused(t *testing.T) {
	record := func(ref string) DocumentHomeNode {
		return DocumentHomeNode{Key: "e", Title: "E", Target: &HomeTarget{Type: HomeTargetRecord, Ref: ref}}
	}
	workflow := func(ref, step string) DocumentHomeNode {
		return DocumentHomeNode{Key: "e", Title: "E", Target: &HomeTarget{Type: HomeTargetWorkflow, Ref: ref, Step: step}}
	}
	cases := []struct {
		name  string
		nodes []DocumentHomeNode
		valid bool
	}{
		{"a record by code", homeOver(record("RC-1")), true},
		{"a record by name", homeOver(record("corals")), true},
		{"a record nobody holds", homeOver(record("RC-9")), false},
		{"a record name two records share", homeOver(record("Dives")), false},
		{"a workflow by name and its step by name", homeOver(workflow("Survey", "Count corals")), true},
		{"a workflow by code and its step by code", homeOver(workflow("W-1", "WI-1")), true},
		{"a workflow nobody holds", homeOver(workflow("W-9", "")), false},
		{"a step the workflow does not hold by name", homeOver(workflow("Survey", "Sign off")), false},
		{"a step name two steps of the workflow share", homeOver(workflow("Repair", "Sign off")), false},
		{"a row page by name", homeOver(DocumentHomeNode{Key: "p", Title: "P", PageOf: "Tide log"}), true},
		{"a row page of a record nobody holds", homeOver(DocumentHomeNode{Key: "p", Title: "P", PageOf: "Anchors"}), false},
		{"a role by name", homeOver(DocumentHomeNode{Key: "r", Title: "R", Roles: []string{"Diver"}, Target: &HomeTarget{Type: HomeTargetFiles}}), true},
		{"a role name two roles share", homeOver(DocumentHomeNode{Key: "r", Title: "R", Roles: []string{"Skipper"}, Target: &HomeTarget{Type: HomeTargetFiles}}), false},
		{"a role nobody holds", homeOver(DocumentHomeNode{Key: "r", Title: "R", Roles: []string{"R-9"}, Target: &HomeTarget{Type: HomeTargetFiles}}), false},
		{"a backlog is a name nobody checks", homeOver(DocumentHomeNode{Key: "b", Title: "B", Target: &HomeTarget{Type: HomeTargetBacklog, Ref: "Anything"}}), true},
		{"a reference nested in a section", homeOver(DocumentHomeNode{Key: "s", Title: "S", Entries: homeOver(record("Anchors"))}), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := reefParts().CheckHome(c.nodes)
			if c.valid && err != nil {
				t.Fatalf("refused: %v", err)
			}
			if !c.valid && err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestCheckHomeRoles_OnlyRolesAreCheckedWhenOnlyRolesAreKnown(t *testing.T) {
	parts := HomeParts{Roles: reefParts().Roles}
	nodes := homeOver(DocumentHomeNode{Key: "e", Title: "E", Roles: []string{"Diver"}, Target: &HomeTarget{Type: HomeTargetRecord, Ref: "RC-77"}})
	if err := parts.CheckHomeRoles(nodes); err != nil {
		t.Fatalf("a role by name with an unchecked record ref = %v, want it accepted", err)
	}
	nodes[0].Roles = []string{"Skipper"}
	if err := parts.CheckHomeRoles(nodes); err == nil {
		t.Fatal("an ambiguous role name was accepted")
	}
}

func decodeHome(t *testing.T, text string) error {
	t.Helper()
	var nodes []DocumentHomeNode
	return json.Unmarshal([]byte(text), &nodes)
}

func TestDocumentHomeNode_AShapeTheDeclarationDoesNotHoldIsRefusedWhenReadFromJSON(t *testing.T) {
	cases := map[string]string{
		"a field the node does not have, such as a where beside the target": `[{"key":"a","title":"A","where":[],"target":{"type":"record","ref":"RC-1"}}]`,
		"a field the target does not have":                                  `[{"key":"a","title":"A","target":{"type":"files","colour":"red"}}]`,
		"a field the condition does not have":                               `[{"key":"p","title":"P","page_of":"RC-1","entries":[{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","where":[{"question":"Q","operator":"is","value":"x","also":1}]}}]}]`,
		"a blank page_of":                                                   `[{"key":"a","title":"A","page_of":"","entries":[]}]`,
		"a blank step":                                                      `[{"key":"a","title":"A","target":{"type":"workflow","ref":"W-1","step":""}}]`,
		"a blank scope":                                                     `[{"key":"a","title":"A","target":{"type":"workflow","ref":"W-1","scope":""}}]`,
		"a blank ref":                                                       `[{"key":"a","title":"A","target":{"type":"files","ref":""}}]`,
		"a blank permission":                                                `[{"key":"a","title":"A","permission":"","target":{"type":"files"}}]`,
		"a placeholder with a second key":                                   `[{"key":"p","title":"P","page_of":"RC-1","entries":[{"key":"a","title":"A","target":{"type":"record","ref":"RC-1","where":[{"question":"Q","operator":"is","value":{"row":"Q","also":1}}]}}]}]`,
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			if err := decodeHome(t, text); err == nil {
				t.Fatal("read without an error")
			}
		})
	}
}

func TestDocumentHomeNode_TheShapeTheDeclarationHoldsIsReadAsWritten(t *testing.T) {
	text := `[{"key":"a","title":"A","titles":{"sw":"B"},"icon":"folder","permission":"task.read","roles":["R-1"],"entries":[
		{"key":"p","title":"P","page_of":"RC-1","entries":[]},
		{"key":"e","title":"E","target":{"type":"record","ref":"RC-1","where":[{"question":"Q","operator":"is","value":{"row":"Q"}}]}}]}]`
	if err := decodeHome(t, text); err != nil {
		t.Fatalf("read = %v, want it accepted", err)
	}
}

func TestValidateHomeDocument_ATargetBesideAnEmptyListOfEntriesIsRefusedLikeATargetBesideEntries(t *testing.T) {
	var nodes []DocumentHomeNode
	if err := json.Unmarshal([]byte(`[{"key":"a","title":"A","entries":[],"target":{"type":"files"}}]`), &nodes); err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := ValidateHomeDocument(nodes); err == nil {
		t.Fatal("a target beside an empty entries list was accepted")
	}
}

func TestHomePartsOf_APartTheFileMarksDeletedIsNotAPartTheHomeMayName(t *testing.T) {
	doc := AgencyDocument{
		Roles:     []DocumentRole{{Code: "R-1", Name: "Diver"}, {Code: "R-2", Name: "Retired", Deleted: true}},
		Records:   []DocumentRecord{{Code: "RC-1", Name: "Corals"}, {Code: "RC-2", Name: "Old", Deleted: true}},
		Workflows: []DocumentWorkflow{{ID: "w1", Code: "W-1", Name: "Survey"}, {ID: "w2", Code: "W-2", Name: "Old", Deleted: true}},
		WorkItems: []DocumentWorkItem{{Code: "WI-1", Title: "Count", WorkflowID: "w1"}, {Code: "WI-2", Title: "Old", WorkflowID: "w1", Deleted: true}},
	}
	parts := HomePartsOf(doc)
	if len(parts.Roles) != 1 || len(parts.Records) != 1 || len(parts.Workflows) != 1 || len(parts.Steps) != 1 {
		t.Fatalf("parts = %+v, want only the live part of each kind", parts)
	}
}
