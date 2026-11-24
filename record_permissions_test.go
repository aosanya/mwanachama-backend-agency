package mwanachamaagency_test

import (
	"reflect"
	"slices"
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

func tenderRecord() models.Record {
	prompts := []string{"Tender title", "Issuer", "Where it was found", "Country", "Service line", "Closing date", "Estimated value", "Eligibility conditions we may not meet", "Entered by", "Outcome"}
	codes := []string{"tender_title", "issuer", "where_found", "country", "service_line", "closing_date", "estimated_value", "eligibility", "entered_by", "outcome"}
	questions := make([]models.StepQuestion, len(prompts))
	for i := range prompts {
		questions[i] = models.StepQuestion{Key: "f_000000000000000" + string(rune('0'+i)), Code: codes[i], Prompt: prompts[i], AnswerType: models.AnswerFreeText}
	}
	return models.Record{Code: "tender", Name: "Tenders", ItemName: "Tender", Questions: questions}
}

func fieldCodesOf(record models.Record) []string {
	codes := make([]string, len(record.Questions))
	for i, q := range record.Questions {
		codes[i] = q.Code
	}
	return codes
}

func expectedFieldLevelCodes(recordCode string, fields []string) []string {
	var want []string
	for _, action := range []string{"add", "edit", "view"} {
		for _, field := range fields {
			want = append(want, "record."+recordCode+"."+action+"."+field)
		}
		want = append(want, "record."+recordCode+"."+action+".*")
	}
	return append(want, "record."+recordCode+".delete")
}

func TestRecordPermissions_ATenderOfTenFieldsGivesThirtyFourFieldLevelCodes(t *testing.T) {
	tender := tenderRecord()
	codes, _ := codesAndNames(mwanachamaagency.RecordPermissionsOf(tender))
	want := expectedFieldLevelCodes("tender", fieldCodesOf(tender))
	if len(want) != 34 {
		t.Fatalf("test setup: want %d codes, expected 34", len(want))
	}
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("codes = %v, want %v", codes, want)
	}
}

func TestRecordPermissions_NoPlainAddEditOrViewCodeExists(t *testing.T) {
	codes, _ := codesAndNames(mwanachamaagency.RecordPermissionsOf(tenderRecord()))
	for _, code := range codes {
		for _, plain := range []string{"record.tender.add", "record.tender.edit", "record.tender.view"} {
			if code == plain {
				t.Errorf("the plain code %q is still given", plain)
			}
		}
	}
}

func TestRecordPermissions_TheFieldPartIsTheStoredCodeNotThePrompt(t *testing.T) {
	record := models.Record{Code: "tender", ItemName: "Tender", Questions: []models.StepQuestion{
		{Key: "f_0000000000000001", Code: "bid_deadline", Prompt: "Closing date (UTC)!", AnswerType: models.AnswerFreeText},
	}}
	codes, _ := codesAndNames(mwanachamaagency.RecordPermissionsOf(record))
	for _, want := range []string{"record.tender.add.bid_deadline", "record.tender.edit.bid_deadline", "record.tender.view.bid_deadline"} {
		if !slices.Contains(codes, want) {
			t.Errorf("codes = %v, want %q", codes, want)
		}
	}
	for _, code := range codes {
		if strings.Contains(code, "closing") {
			t.Errorf("code %q was worked out from the prompt", code)
		}
	}
}

func TestRecordPermissions_AFieldWithNoStoredCodeGivesNoFieldCode(t *testing.T) {
	record := models.Record{Code: "tender", ItemName: "Tender", Questions: []models.StepQuestion{{Prompt: "Closing date", AnswerType: models.AnswerFreeText}}}
	codes, _ := codesAndNames(mwanachamaagency.RecordPermissionsOf(record))
	want := expectedFieldLevelCodes("tender", nil)
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("codes = %v, want only the stars and delete %v", codes, want)
	}
}

func TestRecordPermissions_DeleteHasNoFieldPartAndNoStar(t *testing.T) {
	codes, _ := codesAndNames(mwanachamaagency.RecordPermissionsOf(tenderRecord()))
	var deletes []string
	for _, code := range codes {
		if strings.Contains(code, ".delete") {
			deletes = append(deletes, code)
		}
	}
	if !reflect.DeepEqual(deletes, []string{"record.tender.delete"}) {
		t.Fatalf("delete codes = %v, want record.tender.delete only", deletes)
	}
}

func TestRecordPermissions_ARecordWithNoFieldsGivesTheThreeStarsAndDelete(t *testing.T) {
	codes, _ := codesAndNames(mwanachamaagency.RecordPermissionsOf(models.Record{Code: "tender", ItemName: "Tender"}))
	want := []string{"record.tender.add.*", "record.tender.edit.*", "record.tender.view.*", "record.tender.delete"}
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("codes = %v, want %v", codes, want)
	}
}

func TestRecordPermissions_NoFieldLevelCodeEqualsACatalogueCode(t *testing.T) {
	catalogue := map[string]bool{}
	for _, p := range models.PermissionCatalog() {
		catalogue[p.Code] = true
	}
	for _, p := range mwanachamaagency.RecordPermissionsOf(tenderRecord()) {
		if catalogue[p.Code] {
			t.Errorf("derived code %q equals a catalogue code", p.Code)
		}
	}
}

func TestRecordPermissions_EveryFieldLevelPermissionHasAName(t *testing.T) {
	seen := map[string]string{}
	for _, p := range mwanachamaagency.RecordPermissionsOf(tenderRecord()) {
		if strings.TrimSpace(p.Name) == "" {
			t.Errorf("%s has no name", p.Code)
		}
		if other, dup := seen[p.Name]; dup {
			t.Errorf("%s and %s share the name %q", p.Code, other, p.Name)
		}
		seen[p.Name] = p.Code
	}
}
