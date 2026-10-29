package mwanachamaagency_test

import (
	"context"
	"reflect"
	"sort"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type sendBackWorld struct {
	scenarioWorld
	buyRule   models.Rule
	groupRule models.Rule
	idleRule  models.Rule
	extraRole models.Role
	onboard   models.Workflow
}

func newSendBackWorld(t *testing.T) sendBackWorld {
	t.Helper()
	w := sendBackWorld{scenarioWorld: newScenarioWorld(t)}
	ctx := context.Background()
	rule := func(text string, target models.TargetType, id string) models.Rule {
		r, err := w.mgr.CreateRule(ctx, models.Rule{DraftID: w.draftID, Text: text})
		if err != nil {
			t.Fatalf("CreateRule: %v", err)
		}
		if _, err := w.mgr.CreateRuleAttachment(ctx, r.ID, target, id); err != nil {
			t.Fatalf("CreateRuleAttachment: %v", err)
		}
		return r
	}
	groupStart := w.stepsOf(t, w.draftID, w.groupOrd)[0]
	w.buyRule = rule("Ask the shop once", models.TargetTypeWorkflow, w.buyNow.ID)
	w.groupRule = rule("Hold the minimum", models.TargetTypeWorkItem, groupStart.ID)
	all, _ := w.mgr.ListWorkflows(ctx, w.draftID, mwanachamaagency.WorkflowFilter{})
	for _, wf := range all {
		if wf.Name == "Onboarding" {
			w.onboard = wf
		}
	}
	w.idleRule = rule("Charter first", models.TargetTypeWorkflow, w.onboard.ID)
	r, err := w.mgr.CreateRole(ctx, models.Role{DraftID: w.draftID, Name: "Extra"})
	if err != nil {
		t.Fatal(err)
	}
	w.extraRole = r

	w.scenario(t, "Buy", w.buyNow, w.actor("Farmer 1", farmer), w.actor("Shop", supplier))
	w.scenario(t, "Group", w.groupOrd, w.actor("Coordinator", coordinator), w.actor("Farmer 1", farmer), w.actor("Shop", supplier))
	w.scenario(t, "Extra", w.buyNow, models.ScenarioActor{Name: "Spare", RoleID: r.ID})
	w.runAll(t)
	return w
}

func (w sendBackWorld) runAll(t *testing.T) {
	t.Helper()
	if _, err := w.mgr.RunScenarios(context.Background(), w.draftID, nil); err != nil {
		t.Fatalf("RunScenarios: %v", err)
	}
	for name, result := range w.results(t) {
		if result != models.ScenarioPassed {
			t.Fatalf("%s = %q before the change, want passed", name, result)
		}
	}
}

func (w sendBackWorld) notRun(t *testing.T) []string {
	t.Helper()
	var out []string
	for name, result := range w.results(t) {
		if result == models.ScenarioNotRun {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func TestAChange_SendsBackOnlyTheScenariosThatUseIt(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		change func(t *testing.T, w sendBackWorld)
		want   []string
	}{
		{"a rule on the workflow is edited", func(t *testing.T, w sendBackWorld) {
			r := w.buyRule
			r.Text = "Ask the shop twice"
			if _, err := w.mgr.UpdateRule(ctx, r); err != nil {
				t.Fatal(err)
			}
		}, []string{"Buy", "Extra"}},
		{"a rule on a step is edited", func(t *testing.T, w sendBackWorld) {
			r := w.groupRule
			r.Text = "Hold the minimum twice"
			if _, err := w.mgr.UpdateRule(ctx, r); err != nil {
				t.Fatal(err)
			}
		}, []string{"Group"}},
		{"a rule on a workflow nobody tests is edited", func(t *testing.T, w sendBackWorld) {
			r := w.idleRule
			r.Text = "Charter second"
			if _, err := w.mgr.UpdateRule(ctx, r); err != nil {
				t.Fatal(err)
			}
		}, nil},
		{"a rule is removed", func(t *testing.T, w sendBackWorld) {
			if err := w.mgr.DeleteRule(ctx, w.buyRule.ID); err != nil {
				t.Fatal(err)
			}
		}, []string{"Buy", "Extra"}},
		{"a rule is attached to the workflow", func(t *testing.T, w sendBackWorld) {
			r, _ := w.mgr.CreateRule(ctx, models.Rule{DraftID: w.draftID, Text: "New"})
			if _, err := w.mgr.CreateRuleAttachment(ctx, r.ID, models.TargetTypeWorkflow, w.groupOrd.ID); err != nil {
				t.Fatal(err)
			}
		}, []string{"Group"}},
		{"a rule is detached from the workflow", func(t *testing.T, w sendBackWorld) {
			atts, err := w.mgr.ListRuleAttachments(ctx, w.buyRule.ID)
			if err != nil || len(atts) != 1 {
				t.Fatalf("ListRuleAttachments: %v %v", err, atts)
			}
			if err := w.mgr.DeleteRuleAttachment(ctx, atts[0].ID); err != nil {
				t.Fatal(err)
			}
		}, []string{"Buy", "Extra"}},
		{"a workflow is edited", func(t *testing.T, w sendBackWorld) {
			wf := w.groupOrd
			wf.Description = "changed"
			if _, err := w.mgr.UpdateWorkflow(ctx, wf); err != nil {
				t.Fatal(err)
			}
		}, []string{"Group"}},
		{"a workflow is removed", func(t *testing.T, w sendBackWorld) {
			if err := w.mgr.DeleteWorkflow(ctx, w.groupOrd.ID); err != nil {
				t.Fatal(err)
			}
		}, []string{"Group"}},
		{"another workflow is edited", func(t *testing.T, w sendBackWorld) {
			wf := w.onboard
			wf.Description = "changed"
			if _, err := w.mgr.UpdateWorkflow(ctx, wf); err != nil {
				t.Fatal(err)
			}
		}, nil},
		{"a step is edited", func(t *testing.T, w sendBackWorld) {
			step := w.stepsOf(t, w.draftID, w.groupOrd)[4]
			step.Title = "Open it"
			if _, err := w.mgr.UpdateWorkItem(ctx, step); err != nil {
				t.Fatal(err)
			}
		}, []string{"Group"}},
		{"a step is added", func(t *testing.T, w sendBackWorld) {
			item := w.item(w.buyNow, 4, stepSpec{"Rate", models.StepKindNotify, supplier, "Rating", []models.StepQuestion{ask("Write the message", models.AnswerFreeText)}, ""})
			if _, err := w.mgr.CreateWorkItem(ctx, item); err != nil {
				t.Fatal(err)
			}
		}, []string{"Buy", "Extra"}},
		{"a step is removed", func(t *testing.T, w sendBackWorld) {
			if err := w.mgr.DeleteWorkItem(ctx, w.stepsOf(t, w.draftID, w.groupOrd)[7].ID); err != nil {
				t.Fatal(err)
			}
		}, []string{"Group"}},
		{"a step of an unrelated workflow is edited", func(t *testing.T, w sendBackWorld) {
			all, _ := w.mgr.ListWorkItems(ctx, w.draftID, mwanachamaagency.WorkItemFilter{})
			for _, wi := range all {
				if wi.WorkflowID == w.onboard.ID {
					wi.Title = "Write the charter"
					if _, err := w.mgr.UpdateWorkItem(ctx, wi); err != nil {
						t.Fatal(err)
					}
				}
			}
		}, nil},
		{"a role every scenario uses is edited", func(t *testing.T, w sendBackWorld) {
			r := w.roles[farmer]
			r.Description = "changed"
			if _, err := w.mgr.UpdateRole(ctx, r); err != nil {
				t.Fatal(err)
			}
		}, []string{"Buy", "Extra", "Group"}},
		{"a role only a step of one workflow uses is edited", func(t *testing.T, w sendBackWorld) {
			r := w.roles[coordinator]
			r.Description = "changed"
			if _, err := w.mgr.UpdateRole(ctx, r); err != nil {
				t.Fatal(err)
			}
		}, []string{"Group"}},
		{"a role only an actor holds is edited", func(t *testing.T, w sendBackWorld) {
			r := w.extraRole
			r.Description = "changed"
			if _, err := w.mgr.UpdateRole(ctx, r); err != nil {
				t.Fatal(err)
			}
		}, []string{"Extra"}},
		{"a role nobody holds is edited", func(t *testing.T, w sendBackWorld) {
			r := w.agentRole
			r.Description = "changed"
			if _, err := w.mgr.UpdateRole(ctx, r); err != nil {
				t.Fatal(err)
			}
		}, nil},
		{"a role is removed", func(t *testing.T, w sendBackWorld) {
			if err := w.mgr.DeleteRole(ctx, w.roles[coordinator].ID); err != nil {
				t.Fatal(err)
			}
		}, []string{"Group"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newSendBackWorld(t)
			c.change(t, w)
			if got := w.notRun(t); !reflect.DeepEqual(got, c.want) {
				t.Errorf("sent back to Not run yet = %v, want %v", got, c.want)
			}
		})
	}
}

func TestAChange_ReachesAScenarioThroughAWorkflowItRuns(t *testing.T) {
	ctx := context.Background()
	w := newSendBackWorld(t)
	starter := stepSpec{"Start the purchase", models.StepKindRunWorkflow, theAgency, "", nil, ""}
	item := w.item(w.groupOrd, 9, starter)
	item.RunWorkflowID = w.buyNow.ID
	if _, err := w.mgr.CreateWorkItem(ctx, item); err != nil {
		t.Fatal(err)
	}
	w.runAll(t)

	r := w.buyRule
	r.Text = "Ask the shop three times"
	if _, err := w.mgr.UpdateRule(ctx, r); err != nil {
		t.Fatal(err)
	}
	if got, want := w.notRun(t), []string{"Buy", "Extra", "Group"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sent back = %v, want %v: Group runs Buy now", got, want)
	}
}
