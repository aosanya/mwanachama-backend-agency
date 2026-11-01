package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestWorkItemTool_AStepARoleActsOnNamesItsOneTool(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")
	in := w.item(wf, 1, buyNowSteps[0])
	in.Tool = "market_ask"

	made, err := w.mgr.CreateWorkItem(ctx, in)
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	got, err := w.mgr.GetWorkItem(ctx, made.ID)
	if err != nil || got.Tool != "market_ask" {
		t.Fatalf("read back tool = %q, %v", got.Tool, err)
	}
}

func TestWorkItemTool_IsRefusedWhereNobodyUsesIt(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")

	agencyStep := w.item(wf, 1, groupOrderSteps[1])
	agencyStep.Tool = "market_ask"
	if _, err := w.mgr.CreateWorkItem(ctx, agencyStep); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("tool on an agency step: want ErrInvalidWorkItem, got %v", err)
	}

	instruction := models.WorkItem{DraftID: w.draftID, WorkflowID: wf.ID, Title: "Think", Ordinality: 2, Tool: "market_ask"}
	if _, err := w.mgr.CreateWorkItem(ctx, instruction); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
		t.Fatalf("tool on an instruction: want ErrInvalidWorkItem, got %v", err)
	}
}

func TestWorkItemTool_IsOneWord(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")
	for _, tool := range []string{"market ask", "market_ask,market_buy_now"} {
		in := w.item(wf, 1, buyNowSteps[0])
		in.Tool = tool
		if _, err := w.mgr.CreateWorkItem(ctx, in); !errors.Is(err, mwanachamaagency.ErrInvalidWorkItem) {
			t.Errorf("tool %q: want ErrInvalidWorkItem, got %v", tool, err)
		}
	}
}

func TestWorkItemTool_SurvivesAFork(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Buy now")
	in := w.item(wf, 1, buyNowSteps[0])
	in.Tool = "market_ask"
	if _, err := w.mgr.CreateWorkItem(ctx, in); err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}

	fork, err := w.mgr.CreateDraft(ctx, "fork", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	items, _ := w.mgr.ListWorkItems(ctx, fork.ID, mwanachamaagency.WorkItemFilter{})
	for _, wi := range items {
		if wi.Kind == models.StepKindFillForm && wi.Tool != "market_ask" {
			t.Errorf("forked step tool = %q, want market_ask", wi.Tool)
		}
	}
}
