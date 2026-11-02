package mwanachamaagency

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const declaredRoles = 38

func TestBlueprintParses(t *testing.T) {
	b, err := Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	if len(b.Roles()) != declaredRoles {
		t.Errorf("roles = %d, want %d: %v", len(b.Roles()), declaredRoles, b.Roles())
	}
}

func TestEveryDeclaredColumnHasACarrierField(t *testing.T) {
	b, err := Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}

	carriers := map[string]any{
		"agency":             models.Agency{},
		"draft":              models.Draft{},
		"goal":               models.Goal{},
		"objective":          models.Objective{},
		"workflow":           models.Workflow{},
		"work_item":          models.WorkItem{},
		"event_flow_step":    models.EventFlowStep{},
		"role":               models.Role{},
		"workflow_objective": models.WorkflowObjective{},
		"role_assignment":    models.RoleAssignment{},
		"module":             models.Module{},
		"value":              models.Value{},
		"value_attachment":   models.ValueAttachment{},
		"rule":               models.Rule{},
		"rule_attachment":    models.RuleAttachment{},
		"control":            models.Control{},
		"control_attachment": models.ControlAttachment{},
		"issue":              models.Issue{},
		"scenario":           models.Scenario{},
		"scenario_actor":     models.ScenarioActor{},
		"scenario_input":     models.ScenarioInput{},
		"test_run":           models.TestRun{},
		"review":             models.Review{},
		"review_area":        models.ReviewArea{},
		"review_finding":     models.ReviewFinding{},
		"review_comment":     models.ReviewComment{},
	}

	for role, carrier := range carriers {
		t.Run(role, func(t *testing.T) {
			o, _ := b.Object(role)
			carried := specstore.ColumnsOf(reflect.TypeOf(carrier))
			for _, f := range o.Fields {
				if !carried[f.Name] {
					t.Errorf("%s declares %q, which %T does not carry", role, f.Name, carrier)
				}
			}
			for column := range carried {
				if !declaresColumn(o, column) {
					t.Errorf("%T carries %q, which the object %q does not declare", carrier, column, o.Name)
				}
			}
		})
	}

	if _, ok := carriers["agency_publication"]; ok {
		t.Error("agency_publication has a carrier now — fold it into the table above")
	}
	if reflect.TypeOf(models.AgencyPublication{}).Field(6).Type.Kind() != reflect.Struct {
		t.Error("AgencyPublication.Document is no longer a struct — the carrier gap this records is closed")
	}
}

func TestEveryNamedPatternIsSupplied(t *testing.T) {
	b, err := Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	for _, role := range b.Roles() {
		o, _ := b.Object(role)
		for _, f := range o.Fields {
			if f.Matches == "" {
				continue
			}
			if _, ok := patterns[f.Matches]; !ok {
				t.Errorf("%s.%s names the pattern %q, which this module does not supply", role, f.Name, f.Matches)
			}
		}
	}
}

func TestVocabularyMatchesTheBlueprint(t *testing.T) {
	inGo := map[string][]string{
		"draft.status": {
			string(models.DraftStatusOpen),
			string(models.DraftStatusArchived),
		},
		"role_assignment.label": {
			string(models.RACILabelResponsible),
			string(models.RACILabelAccountable),
			string(models.RACILabelConsulted),
			string(models.RACILabelInformed),
		},
		"work_item.kind": {
			string(models.StepKindFillForm),
			string(models.StepKindSendForm),
			string(models.StepKindNotify),
			string(models.StepKindYesNo),
			string(models.StepKindRoute),
			string(models.StepKindRunWorkflow),
			string(models.StepKindChat),
		},
		"workflow.opens_in": {
			string(models.OpensInForm),
			string(models.OpensInChat),
		},
		"scenario.result": {
			string(models.ScenarioNotRun), string(models.ScenarioWaiting), string(models.ScenarioRunning), string(models.ScenarioPassed),
			string(models.ScenarioFailed), string(models.ScenarioStopped),
		},
		"scenario_actor.kind": {
			string(models.ActorKindRole), string(models.ActorKindAgent), string(models.ActorKindSystem),
		},
		"test_run.result": {
			string(models.TestRunWaiting), string(models.TestRunRunning), string(models.TestRunPassed),
			string(models.TestRunFailed), string(models.TestRunStopped),
		},
		"review.status": {
			string(models.ReviewRunning), string(models.ReviewDone), string(models.ReviewFailed),
		},
		"review_area.area": {
			string(models.ReviewAreaProfile), string(models.ReviewAreaGoals), string(models.ReviewAreaObjectives), string(models.ReviewAreaRolesAgents),
			string(models.ReviewAreaWorkflows), string(models.ReviewAreaRisks), string(models.ReviewAreaTests),
		},
		"review_area.status": {
			string(models.ReviewAreaReading), string(models.ReviewAreaDone), string(models.ReviewAreaFailed),
		},
		"review_finding.status": {
			string(models.FindingOpen), string(models.FindingApplied), string(models.FindingDismissed),
		},
		"review_finding.fate": {
			string(models.FateNew), string(models.FateFixed), string(models.FateStillOpen), string(models.FateWithdrawn),
		},
		"issue.severity": {
			string(models.IssueSeverityInfo),
			string(models.IssueSeverityWarning),
			string(models.IssueSeverityError),
		},
		"value_attachment.target_type": {
			string(models.TargetTypeRole),
			string(models.TargetTypeGoal),
			string(models.TargetTypeObjective),
			string(models.TargetTypeWorkflow),
			string(models.TargetTypeWorkItem),
		},
		"rule_attachment.target_type": {
			string(models.TargetTypeRole),
			string(models.TargetTypeGoal),
			string(models.TargetTypeObjective),
			string(models.TargetTypeWorkflow),
			string(models.TargetTypeWorkItem),
		},
		"control_attachment.target_type": {
			string(models.TargetTypeRole),
			string(models.TargetTypeGoal),
			string(models.TargetTypeObjective),
			string(models.TargetTypeWorkflow),
			string(models.TargetTypeWorkItem),
			string(models.TargetTypeRiskPathway),
			string(models.TargetTypeThreat),
			string(models.TargetTypeConsequence),
		},
	}

	b, err := Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}

	for _, role := range b.Roles() {
		o, _ := b.Object(role)
		for _, f := range o.Fields {
			if f.Type != spec.TypeEnum {
				continue
			}
			key := role + "." + f.Name
			want, named := inGo[key]
			if !named {
				t.Errorf("the blueprint declares the enum %s with the values %s, and Go names no constants for them",
					key, strings.Join(f.Values, ", "))
				continue
			}
			delete(inGo, key)
			if !sameValues(want, f.Values) {
				t.Errorf("%s: Go names %s, the blueprint declares %s — a stored value outlives a rename, so these have to agree",
					key, strings.Join(sorted(want), ", "), strings.Join(sorted(f.Values), ", "))
			}
		}
	}

	for key := range inGo {
		t.Errorf("Go names constants for %s, which the blueprint declares no enum for", key)
	}
}

func sameValues(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := sorted(a), sorted(b)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func sorted(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}
