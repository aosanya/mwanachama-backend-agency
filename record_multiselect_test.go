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
		DraftID: w.draftID, WorkflowID: workflow.ID, Name: "Consultants", RoleID: w.roles[coordinator].ID,
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
	if kept.Answers["Expertise areas"] != "Financial inclusion; Sustainable finance" {
		t.Errorf("stored %q, want the text as sent", kept.Answers["Expertise areas"])
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
