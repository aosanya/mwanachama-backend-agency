package models

import "testing"

func TestNestHomeEntries_RebuildsTheDeclarationFromFlatRowsInPlaceOrder(t *testing.T) {
	rows := []HomeEntry{
		{ID: "c", ParentID: "a", Key: "second", Title: "Second", TargetType: HomeTargetFiles, Ordinality: 1},
		{ID: "a", Key: "top", Title: "Top", Ordinality: 0},
		{ID: "b", ParentID: "a", Key: "first", Title: "First", TargetType: HomeTargetRecord, Target: "RC-1", Ordinality: 0},
		{ID: "d", Key: "last", Title: "Last", TargetType: HomeTargetBacklog, Target: "x", Ordinality: 1},
	}
	got := NestHomeEntries(rows)
	if len(got) != 2 || got[0].Key != "top" || got[1].Key != "last" {
		t.Fatalf("top = %+v, want top then last", got)
	}
	if len(got[0].Entries) != 2 || got[0].Entries[0].Key != "first" || got[0].Entries[1].Key != "second" {
		t.Fatalf("inside top = %+v, want first then second", got[0].Entries)
	}
	if got[0].Entries[0].Target == nil || got[0].Entries[0].Target.Ref != "RC-1" || got[0].Target != nil {
		t.Fatalf("targets = %+v, want only the entries to carry one", got[0])
	}
}

func TestValidateHomeDocument_AcceptsFiveLevelsAndRefusesSix(t *testing.T) {
	deep := func(levels int) []DocumentHomeNode {
		node := DocumentHomeNode{Key: "n", Title: "N"}
		for level := levels - 1; level >= 1; level-- {
			node = DocumentHomeNode{Key: "n" + string(rune('a'+level)), Title: "N", Entries: []DocumentHomeNode{node}}
		}
		return []DocumentHomeNode{node}
	}
	if err := ValidateHomeDocument(deep(HomeMaxDepth)); err != nil {
		t.Fatalf("%d levels = %v, want it accepted", HomeMaxDepth, err)
	}
	if err := ValidateHomeDocument(deep(HomeMaxDepth + 1)); err == nil {
		t.Fatalf("%d levels accepted, want it refused", HomeMaxDepth+1)
	}
}

func TestValidateHomeDocument_RefusesMoreThanTheTotalLimit(t *testing.T) {
	var nodes []DocumentHomeNode
	for i := 0; i < HomeMaxEntries/HomeMaxPerSection+1; i++ {
		section := DocumentHomeNode{Key: "s" + string(rune('a'+i)), Title: "S"}
		for j := 0; j < HomeMaxPerSection; j++ {
			section.Entries = append(section.Entries, DocumentHomeNode{Key: section.Key + "-" + string(rune('a'+j%26)) + string(rune('a'+j/26)), Title: "E", Target: &HomeTarget{Type: HomeTargetFiles}})
		}
		nodes = append(nodes, section)
	}
	if err := ValidateHomeDocument(nodes); err == nil {
		t.Fatalf("%d rows accepted, want the %d limit to refuse them", len(nodes)*(HomeMaxPerSection+1), HomeMaxEntries)
	}
}

func TestValidateHomeEntry_ATitleAndATranslatedTitleHaveALength(t *testing.T) {
	long := make([]rune, HomeMaxTitleLength+1)
	for i := range long {
		long[i] = 'a'
	}
	if err := ValidateHomeEntry(HomeEntry{Key: "a", Title: string(long)}); err == nil {
		t.Fatal("a title past the limit was accepted")
	}
	if err := ValidateHomeEntry(HomeEntry{Key: "a", Title: "A", Titles: map[string]string{"sw": string(long)}}); err == nil {
		t.Fatal("a translated title past the limit was accepted")
	}
	if err := ValidateHomeEntry(HomeEntry{Key: "a", Title: "A", Titles: map[string]string{"sw": " "}}); err == nil {
		t.Fatal("an empty translated title was accepted")
	}
	if err := ValidateHomeEntry(HomeEntry{Key: "a", Title: "A", Titles: map[string]string{"sw": "B"}}); err != nil {
		t.Fatalf("a good translated title = %v", err)
	}
}
