package mwanachamaagency_test

import (
	"context"
	"slices"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

var proposalPrompts = []string{
	"Closing date",
	"closing date",
	"  Amount (USD)! ",
	"???",
	"!!",
	"2nd date",
	"Closing date",
}

func TestProposeFieldCodes_NumbersAClashAndNamesAnEmptyPrompt(t *testing.T) {
	got := mwanachamaagency.ProposeFieldCodes(proposalPrompts)
	want := []string{"closing_date", "closing_date_2", "amount_usd", "field", "field_2", "field_2nd_date", "closing_date_3"}
	if !slices.Equal(got, want) {
		t.Errorf("ProposeFieldCodes = %v, want %v", got, want)
	}
}

func TestProposeFieldCodes_GiveTheCodesTheCreatePathGives(t *testing.T) {
	o := buildRoutedOrder(t)
	w := o.world
	in := paymentsRecord(w, o.workflow)
	in.Name = "Proposal twins"
	in.Questions = nil
	for _, prompt := range proposalPrompts {
		in.Questions = append(in.Questions, ask(prompt, models.AnswerDate))
	}
	made, err := w.mgr.CreateRecord(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	created := make([]string, len(made.Questions))
	for i, q := range made.Questions {
		created[i] = q.Code
	}
	if proposed := mwanachamaagency.ProposeFieldCodes(proposalPrompts); !slices.Equal(proposed, created) {
		t.Errorf("ProposeFieldCodes = %v, CreateRecord gave %v", proposed, created)
	}
}

func TestProposeFieldCodes_AnEmptyPromptIsNamedField(t *testing.T) {
	got := mwanachamaagency.ProposeFieldCodes([]string{"", ""})
	if want := []string{"field", "field_2"}; !slices.Equal(got, want) {
		t.Errorf("ProposeFieldCodes = %v, want %v", got, want)
	}
}

func TestProposeFieldCodes_AnEmptyListProposesNothing(t *testing.T) {
	if got := mwanachamaagency.ProposeFieldCodes(nil); len(got) != 0 {
		t.Errorf("ProposeFieldCodes(nil) = %v, want none", got)
	}
}
