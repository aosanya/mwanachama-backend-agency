package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func consultantsRecord(w stepWorld, workflow models.Workflow) models.Record {
	return models.Record{
		DraftID: w.draftID, WorkflowID: workflow.ID, Name: "Consultants",
		FormName: "Consultant", Questions: []models.StepQuestion{
			{Prompt: "Expertise areas", AnswerType: models.AnswerMultiSelect, Required: true, Options: []string{"Financial inclusion", "Sustainable finance", "Banking and central banking"}},
			{Prompt: "Languages", AnswerType: models.AnswerMultiSelect, Options: []string{"English", "French", "Swahili"}},
		},
	}
}

func TestRecordRow_AMultiSelectAnswerIsAListOfTheQuestionsOptionsJoinedWithASemicolon(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	record, err := w.mgr.CreateRecord(ctx, consultantsRecord(w, w.workflow(t, "Consultant pool")))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	mayAdd(t, w.mgr, record)
	row := func(expertise, languages string) models.RecordRow {
		answers := map[string]string{"Expertise areas": expertise}
		if languages != "" {
			answers["Languages"] = languages
		}
		return models.RecordRow{DraftID: w.draftID, RecordID: record.ID, RoleID: w.roles[coordinator].ID, Answers: answers}
	}

	refused := map[string]models.RecordRow{
		"an option the question does not offer":  row("Financial inclusion; Food security", ""),
		"one offered option and one that is not": row("Financial inclusion", "English; Klingon"),
		"the same option twice":                  row("Financial inclusion; Financial inclusion", ""),
		"a list with an empty place":             row("Financial inclusion; ; Sustainable finance", ""),
		"a substring of an option":               row("Financial", ""),
		"options joined with a comma instead":    row("Financial inclusion, Sustainable finance", ""),
		"an option in another letter case":       row("financial inclusion", ""),
	}
	for name, in := range refused {
		if _, err := w.mgr.CreateRecordRow(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidRecordRow) {
			t.Errorf("%s: want ErrInvalidRecordRow, got %v", name, err)
		}
	}

	kept, err := w.mgr.CreateRecordRow(ctx, row("Financial inclusion; Sustainable finance", "English; French; Swahili"))
	if err != nil {
		t.Fatalf("a list of offered options: %v", err)
	}
	if answerTo(record, kept, "Expertise areas") != "Financial inclusion; Sustainable finance" {
		t.Errorf("stored %q, want the text as sent", answerTo(record, kept, "Expertise areas"))
	}
	if _, err := w.mgr.CreateRecordRow(ctx, row("Banking and central banking", "")); err != nil {
		t.Errorf("one offered option: %v", err)
	}
}

func TestRouteStep_ADayOperatorNeedsAWrittenDay(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	route := o.steps["Sort the answers"]
	with := func(operator models.RouteOperator, value string) models.WorkItem {
		in := route
		r := joinsYes(o.steps["Open"].ID)
		r.Rules[0].Operator, r.Rules[0].Value = operator, value
		in.Routes = []models.StepRoute{r}
		return in
	}
	for _, operator := range []models.RouteOperator{models.RouteOnOrBefore, models.RouteOnOrAfter} {
		if _, err := o.world.mgr.UpdateWorkItem(ctx, with(operator, "soon")); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("%s with a word: want ErrInvalidWorkItem, got %v", operator, err)
		}
		if _, err := o.world.mgr.UpdateWorkItem(ctx, with(operator, "2027-03-01")); err != nil {
			t.Errorf("%s with a written day: %v", operator, err)
		}
	}
	if !models.RouteOnOrBefore.Valid() || !models.RouteOnOrAfter.Valid() {
		t.Error("on_or_before and on_or_after are route operators")
	}
}

func TestRecordAndStep_AMultiSelectOptionThatCanNeverBeChosenIsRefusedWhenSaved(t *testing.T) {
	unreachable := map[string][]string{
		"an option holding a semicolon":      {"Research; evaluation", "Audit"},
		"an option with a leading space":     {" Kenya", "Rwanda"},
		"an option with a trailing space":    {"Kenya ", "Rwanda"},
		"an option holding only a semicolon": {";", "Audit"},
	}
	for name, options := range unreachable {
		t.Run(name, func(t *testing.T) {
			w := newStepWorld(t)
			ctx := context.Background()
			workflow := w.workflow(t, "Consultant pool")
			questions := []models.StepQuestion{{Prompt: "Expertise areas", AnswerType: models.AnswerMultiSelect, Required: true, Options: options}}

			saved, err := w.mgr.CreateRecord(ctx, consultantsRecord(w, workflow))
			if err != nil {
				t.Fatalf("CreateRecord with offered options: %v", err)
			}
			saved.Questions = questions
			if _, err := w.mgr.UpdateRecord(ctx, saved); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
				t.Errorf("UpdateRecord: want ErrInvalidRecord, got %v", err)
			}

			other := consultantsRecord(w, workflow)
			other.Name, other.Questions = "Other consultants", questions
			if _, err := w.mgr.CreateRecord(ctx, other); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
				t.Errorf("CreateRecord: want ErrInvalidRecord, got %v", err)
			}

			step := w.item(workflow, 1, stepSpec{"Ask", models.StepKindFillForm, coordinator, "Ask form", questions, "Fills the form"})
			if _, err := w.mgr.CreateWorkItem(ctx, step); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
				t.Errorf("CreateWorkItem: want ErrInvalidWorkItem, got %v", err)
			}
		})
	}
}

func TestRecord_ASingleChoiceOptionMayHoldASemicolon(t *testing.T) {
	w := newStepWorld(t)
	record := consultantsRecord(w, w.workflow(t, "Consultant pool"))
	record.Questions = []models.StepQuestion{{Prompt: "Kind of work", AnswerType: models.AnswerSingleChoice, Required: true, Options: []string{"Research; evaluation", "Audit"}}}
	if _, err := w.mgr.CreateRecord(context.Background(), record); err != nil {
		t.Errorf("a single choice is not split on a semicolon: %v", err)
	}
}
