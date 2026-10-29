package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (w scenarioWorld) removeWorkflowRow(t *testing.T, id string) {
	t.Helper()
	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.db.Exec("delete from "+mwanachamaagency.TablesFor(s).Workflows+" where id = ?", id).Error; err != nil {
		t.Fatalf("removing the workflow row: %v", err)
	}
}

func TestDeleteWorkflow_RefusedWhileAScenarioMirrorsItNamingEveryScenario(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	second := w.scenario(t, "Zebra buys", w.buyNow)
	first := w.scenario(t, "Apple buys", w.buyNow)
	w.scenario(t, "Group", w.groupOrd)

	err := w.mgr.DeleteWorkflow(ctx, w.buyNow.ID)
	if !errors.Is(err, mwanachamaagency.ErrUsedByScenario) {
		t.Fatalf("DeleteWorkflow: %v, want ErrUsedByScenario", err)
	}
	for _, name := range []string{"Apple buys", "Zebra buys"} {
		if !strings.Contains(err.Error(), `"`+name+`"`) {
			t.Errorf("%q does not name %q", err, name)
		}
	}
	if strings.Contains(err.Error(), `"Group"`) {
		t.Errorf("%q names a scenario of another workflow", err)
	}
	if got := w.stepsOf(t, w.draftID, w.buyNow); len(got) == 0 {
		t.Errorf("the refused delete took the workflow's steps")
	}

	if err := w.mgr.DeleteScenario(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	err = w.mgr.DeleteWorkflow(ctx, w.buyNow.ID)
	if !errors.Is(err, mwanachamaagency.ErrUsedByScenario) || strings.Contains(err.Error(), "Apple buys") || !strings.Contains(err.Error(), "Zebra buys") {
		t.Fatalf("after one scenario went: %v, want only the remaining one named", err)
	}

	if err := w.mgr.DeleteScenario(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	if err := w.mgr.DeleteWorkflow(ctx, w.buyNow.ID); err != nil {
		t.Fatalf("DeleteWorkflow once no scenario mirrors it: %v", err)
	}
	if _, err := w.mgr.GetWorkflow(ctx, w.buyNow.ID); !errors.Is(err, mwanachamaagency.ErrWorkflowNotFound) {
		t.Errorf("GetWorkflow after delete: %v, want ErrWorkflowNotFound", err)
	}
}

func TestDeleteRole_RefusedWhileAScenarioActorHoldsItNamingEveryScenario(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	second := w.scenario(t, "Zebra buys", w.buyNow, w.actor("Farmer 1", farmer))
	first := w.scenario(t, "Apple buys", w.buyNow, w.actor("Farmer 2", farmer))
	w.scenario(t, "Group", w.groupOrd, w.actor("Coordinator", coordinator))

	err := w.mgr.DeleteRole(ctx, w.roles[farmer].ID)
	if !errors.Is(err, mwanachamaagency.ErrUsedByScenario) {
		t.Fatalf("DeleteRole: %v, want ErrUsedByScenario", err)
	}
	for _, name := range []string{"Apple buys", "Zebra buys"} {
		if !strings.Contains(err.Error(), `"`+name+`"`) {
			t.Errorf("%q does not name %q", err, name)
		}
	}
	if strings.Contains(err.Error(), `"Group"`) {
		t.Errorf("%q names a scenario whose actors do not hold the role", err)
	}
	if _, err := w.mgr.GetRole(ctx, w.roles[farmer].ID); err != nil {
		t.Errorf("the refused delete took the role: %v", err)
	}

	if err := w.mgr.DeleteScenario(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := w.mgr.DeleteRole(ctx, w.roles[farmer].ID); !errors.Is(err, mwanachamaagency.ErrUsedByScenario) {
		t.Fatalf("after one scenario went: %v, want the other still to refuse", err)
	}

	if err := w.mgr.DeleteScenario(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	if err := w.mgr.DeleteRole(ctx, w.roles[farmer].ID); err != nil {
		t.Fatalf("DeleteRole once no actor holds it: %v", err)
	}
	if _, err := w.mgr.GetRole(ctx, w.roles[farmer].ID); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("GetRole after delete: %v, want ErrRoleNotFound", err)
	}
}

func TestDeleteRole_AnAgentOrStepRoleNoActorHoldsStillGoes(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	w.scenario(t, "Shop says yes", w.buyNow)
	if err := w.mgr.DeleteRole(ctx, w.agentRole.ID); err != nil {
		t.Fatalf("DeleteRole of a role no stored actor holds: %v", err)
	}
}

func TestRenameOfARoleAndAWorkflow_IsAllowedAndScenariosReadTheNewNames(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	s := w.scenario(t, "Shop says yes", w.buyNow, w.actor("Farmer 1", farmer))

	role := w.roles[farmer]
	role.Name = "Smallholder"
	if _, err := w.mgr.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole rename: %v", err)
	}
	flow := w.buyNow
	flow.Name = "Purchase"
	if _, err := w.mgr.UpdateWorkflow(ctx, flow); err != nil {
		t.Fatalf("UpdateWorkflow rename: %v", err)
	}

	got := w.get(t, s.ID)
	if got.Workflow != "Purchase" {
		t.Errorf("scenario workflow = %q, want the new name", got.Workflow)
	}
	var held models.ScenarioActor
	for _, a := range got.Actors {
		if a.Name == "Farmer 1" {
			held = a
		}
	}
	if held.RoleName != "Smallholder" {
		t.Errorf("actor role name = %q, want the new name", held.RoleName)
	}
	if err := w.mgr.DeleteRole(ctx, role.ID); !errors.Is(err, mwanachamaagency.ErrUsedByScenario) {
		t.Errorf("the renamed role is still refused: %v", err)
	}
}
