package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type recordedWorld struct {
	world  stepWorld
	tender models.Record
	day    models.Record
}

func workflowWithAStep(t *testing.T, w stepWorld, name string) models.Workflow {
	t.Helper()
	ctx := context.Background()
	wf := w.workflow(t, name)
	step, err := w.mgr.CreateWorkItem(ctx, w.item(wf, 1, stepSpec{title: "Open", kind: models.StepKindFillForm, role: coordinator, form: "Open form", questions: []models.StepQuestion{ask("Note", models.AnswerFreeText)}}))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	assignOn(t, w.mgr, ctx, w.draftID, w.roles[coordinator], step)
	return wf
}

func buildRecordedWorld(t *testing.T) recordedWorld {
	t.Helper()
	w := newStepWorld(t)
	ctx := context.Background()
	wf := workflowWithAStep(t, w, "Bids")
	tender, err := w.mgr.CreateRecord(ctx, models.Record{
		DraftID: w.draftID, WorkflowID: wf.ID, Name: "Tender", ItemName: "Tender",
		FormName: "Tender", Questions: []models.StepQuestion{
			ask("Title", models.AnswerFreeText),
			ask("Issuer", models.AnswerFreeText),
			ask("Closing date", models.AnswerDate),
		},
	})
	if err != nil {
		t.Fatalf("CreateRecord tender: %v", err)
	}
	day, err := w.mgr.CreateRecord(ctx, models.Record{
		DraftID: w.draftID, WorkflowID: wf.ID, Name: "Day worked",
		FormName: "Day worked", Questions: []models.StepQuestion{ask("Hours", models.AnswerNumber)},
	})
	if err != nil {
		t.Fatalf("CreateRecord day worked: %v", err)
	}
	mayAdd(t, w.mgr, tender)
	mayAdd(t, w.mgr, day)
	return recordedWorld{world: w, tender: tender, day: day}
}

func publishedRecordCodes(pub models.AgencyPublication) []string {
	codes := make([]string, len(pub.Document.RecordPermissions))
	for i, p := range pub.Document.RecordPermissions {
		codes[i] = p.Code
	}
	return codes
}

func TestRecordPublication_ThePublicationListsTheFieldLevelCodesOfEveryRecord(t *testing.T) {
	r := buildRecordedWorld(t)
	pub, err := r.world.mgr.PublishAgency(context.Background(), r.world.draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	want := append(
		expectedFieldLevelCodes("tender", []string{"title", "issuer", "closing_date"}),
		expectedFieldLevelCodes("day_worked", []string{"hours"})...,
	)
	got := publishedRecordCodes(pub)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listed codes = %v, want %v", got, want)
	}
	tenderCodes := 0
	for _, code := range got {
		if strings.HasPrefix(code, "record.tender.") {
			tenderCodes++
		}
	}
	if tenderCodes != 13 {
		t.Errorf("tender gives %d codes, want 13", tenderCodes)
	}
}

func TestRecordPublication_TheListHasStarsAsWrittenNoPlainCodeAndNoTagGroupOrSeat(t *testing.T) {
	r := buildRecordedWorld(t)
	pub, err := r.world.mgr.PublishAgency(context.Background(), r.world.draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	listed := map[string]bool{}
	for _, code := range publishedRecordCodes(pub) {
		listed[code] = true
	}
	for _, star := range []string{"record.tender.add.*", "record.tender.edit.*", "record.tender.view.*", "record.tender.delete"} {
		if !listed[star] {
			t.Errorf("%s is not listed", star)
		}
	}
	for _, plain := range []string{"record.tender.add", "record.tender.edit", "record.tender.view"} {
		if listed[plain] {
			t.Errorf("plain code %s is listed", plain)
		}
	}
	for _, p := range pub.Document.RecordPermissions {
		for _, word := range []string{"tag", "group", "seat"} {
			if strings.Contains(p.Code, word) {
				t.Errorf("%s names a %s", p.Code, word)
			}
		}
	}
}

func TestRecordPublication_ADraftWithNoRecordsWritesNoList(t *testing.T) {
	w := newStepWorld(t)
	workflowWithAStep(t, w, "Bids")
	pub, err := w.mgr.PublishAgency(context.Background(), w.draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.RecordPermissions) != 0 {
		t.Fatalf("listed %v for a draft with no records", publishedRecordCodes(pub))
	}
	raw, err := json.Marshal(pub.Document)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(raw), "record_permissions") {
		t.Errorf("the document carries record_permissions: %s", raw)
	}
}

func TestRecordPublication_TheListIsReadBackFromTheStoredVersion(t *testing.T) {
	r := buildRecordedWorld(t)
	ctx := context.Background()
	pub, err := r.world.mgr.PublishAgency(ctx, r.world.draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	read, err := r.world.mgr.GetPublication(ctx, pub.Version)
	if err != nil {
		t.Fatalf("GetPublication: %v", err)
	}
	if !reflect.DeepEqual(read.Document.RecordPermissions, pub.Document.RecordPermissions) || len(read.Document.RecordPermissions) != 20 {
		t.Fatalf("stored list = %d codes, want the 20 published", len(read.Document.RecordPermissions))
	}
}

func TestRecordPublication_AForkRestoresTheRecordsAndTheNextPublishWritesTheListAgain(t *testing.T) {
	r := buildRecordedWorld(t)
	ctx := context.Background()
	first, err := r.world.mgr.PublishAgency(ctx, r.world.draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	fork, err := r.world.mgr.CreateDraft(ctx, "revisit", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	forked, err := r.world.mgr.ListRecords(ctx, fork.ID, mwanachamaagency.RecordFilter{})
	if err != nil || len(forked) != 2 {
		t.Fatalf("forked records = %d, %v; want 2", len(forked), err)
	}
	for _, record := range forked {
		if record.Code == "tender" {
			record.Questions[2].Code = "deadline"
			if _, err := r.world.mgr.UpdateRecord(ctx, record); err != nil {
				t.Fatalf("UpdateRecord: %v", err)
			}
		}
	}
	second, err := r.world.mgr.PublishAgency(ctx, fork.ID, "release-2", "")
	if err != nil {
		t.Fatalf("PublishAgency fork: %v", err)
	}
	if len(second.Document.RecordPermissions) != len(first.Document.RecordPermissions) {
		t.Fatalf("the fork lists %d codes, the first version %d", len(second.Document.RecordPermissions), len(first.Document.RecordPermissions))
	}
	listed := map[string]bool{}
	for _, code := range publishedRecordCodes(second) {
		listed[code] = true
	}
	if !listed["record.tender.edit.deadline"] || listed["record.tender.edit.closing_date"] {
		t.Errorf("the fork lists %v, want the renamed field code", publishedRecordCodes(second))
	}
}

func TestRecordPublication_AnUnchangedForkOfAPublicationListsTheSameCodes(t *testing.T) {
	r := buildRecordedWorld(t)
	ctx := context.Background()
	first, err := r.world.mgr.PublishAgency(ctx, r.world.draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	fork, err := r.world.mgr.CreateDraft(ctx, "copy", r.world.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	second, err := r.world.mgr.PublishAgency(ctx, fork.ID, "release-2", "")
	if err != nil {
		t.Fatalf("PublishAgency fork: %v", err)
	}
	if !reflect.DeepEqual(publishedRecordCodes(second), publishedRecordCodes(first)) {
		t.Fatalf("the fork lists %v, the first version %v", publishedRecordCodes(second), publishedRecordCodes(first))
	}
}

func TestRecordPublication_ARenamedFieldCodeANewFieldAndAChangedItemNameEachAreAChangeToPublish(t *testing.T) {
	changes := map[string]func(*models.Record){
		"a renamed field code": func(rec *models.Record) { rec.Questions[2].Code = "deadline" },
		"a new field":          func(rec *models.Record) { rec.Questions = append(rec.Questions, ask("Country", models.AnswerFreeText)) },
		"a changed item name":  func(rec *models.Record) { rec.ItemName = "Bid" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			r := buildRecordedWorld(t)
			ctx := context.Background()
			w := r.world
			if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-1", ""); err != nil {
				t.Fatalf("PublishAgency: %v", err)
			}
			if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-2", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
				t.Fatalf("same content: want ErrNoChangesDetected, got %v", err)
			}
			tender, err := w.mgr.GetRecord(ctx, r.tender.ID)
			if err != nil {
				t.Fatalf("GetRecord: %v", err)
			}
			change(&tender)
			if _, err := w.mgr.UpdateRecord(ctx, tender); err != nil {
				t.Fatalf("UpdateRecord: %v", err)
			}
			if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-2", ""); err != nil {
				t.Fatalf("%s is a change: %v", name, err)
			}
		})
	}
}
