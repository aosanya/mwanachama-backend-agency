package mwanachamaagency_test

import (
	"reflect"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func codesAndNames(permissions []models.Permission) ([]string, []string) {
	var codes, names []string
	for _, p := range permissions {
		codes = append(codes, p.Code)
		names = append(names, p.Name)
	}
	return codes, names
}

func TestRecordPermissions_TenderGivesFourCodesAndNames(t *testing.T) {
	got := mwanachamaagency.RecordPermissionsOf(models.Record{Code: "tender", Name: "Tenders", ItemName: "Tender"})
	codes, names := codesAndNames(got)
	wantCodes := []string{"record.tender.add", "record.tender.edit", "record.tender.delete", "record.tender.view"}
	wantNames := []string{"Add a tender", "Edit a tender", "Delete a tender", "See every tender"}
	if !reflect.DeepEqual(codes, wantCodes) {
		t.Fatalf("codes = %v, want %v", codes, wantCodes)
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("names = %v, want %v", names, wantNames)
	}
}

func TestRecordPermissions_AreUnlockedWithEmptyActionsAndNoModule(t *testing.T) {
	for _, p := range mwanachamaagency.RecordPermissionsOf(models.Record{Code: "tender", ItemName: "Tender"}) {
		if p.Locked {
			t.Errorf("%s is locked", p.Code)
		}
		if p.Module != "" {
			t.Errorf("%s has module %q", p.Code, p.Module)
		}
		if p.Actions == nil || len(p.Actions) != 0 {
			t.Errorf("%s actions = %#v, want an empty slice", p.Code, p.Actions)
		}
	}
}

func TestRecordPermissions_ItemNameFallsBackToFormNameThenName(t *testing.T) {
	cases := []struct {
		record models.Record
		want   string
	}{
		{models.Record{Code: "day_worked", ItemName: "Day worked", FormName: "Log a day", Name: "Days"}, "Add a day worked"},
		{models.Record{Code: "bid_decision", FormName: "Bid decision", Name: "Decisions"}, "Add a bid decision"},
		{models.Record{Code: "office_holder", Name: "Office holder"}, "Add an office holder"},
	}
	for _, c := range cases {
		got := mwanachamaagency.RecordPermissionsOf(c.record)
		if got[0].Name != c.want {
			t.Errorf("%s: add name = %q, want %q", c.record.Code, got[0].Name, c.want)
		}
	}
}

func TestRecordPermissions_DayWorkedAndBidDecisionCodes(t *testing.T) {
	for _, code := range []string{"day_worked", "bid_decision", "RC-1"} {
		codes, _ := codesAndNames(mwanachamaagency.RecordPermissionsOf(models.Record{Code: code, Name: "Anything"}))
		want := []string{"record." + code + ".add", "record." + code + ".edit", "record." + code + ".delete", "record." + code + ".view"}
		if !reflect.DeepEqual(codes, want) {
			t.Errorf("%s: codes = %v, want %v", code, codes, want)
		}
	}
}

func TestRecordPermissionCode_IsOneForm(t *testing.T) {
	if got := mwanachamaagency.RecordPermissionCode("tender", "add"); got != "record.tender.add" {
		t.Fatalf("RecordPermissionCode = %q", got)
	}
}

func TestRecordPermissions_NoDerivedCodeEqualsACatalogueCode(t *testing.T) {
	catalogue := map[string]bool{}
	for _, p := range models.PermissionCatalog() {
		catalogue[p.Code] = true
		if strings.HasPrefix(p.Code, "record.") {
			t.Errorf("catalogue code %q starts with record.", p.Code)
		}
	}
	var recordCodes []string
	for _, p := range models.PermissionCatalog() {
		recordCodes = append(recordCodes, p.Code)
	}
	recordCodes = append(recordCodes, "tender", "day_worked", "bid_decision")
	for _, recordCode := range recordCodes {
		for _, p := range mwanachamaagency.RecordPermissionsOf(models.Record{Code: recordCode, Name: "Item"}) {
			if catalogue[p.Code] {
				t.Errorf("derived code %q equals a catalogue code", p.Code)
			}
		}
	}
}
