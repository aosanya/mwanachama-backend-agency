package mwanachamaagency

import (
	"sort"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
)

var markedReferences = map[string]string{
	"goal_id":      "agency.goal",
	"objective_id": "agency.objective",
	"workflow_id":  "agency.workflow",
	"work_item_id": "agency.work_item",
	"role_id":      "agency.role",
	"value_id":     "agency.value",
	"rule_id":      "agency.rule",
}

func TestEveryArgumentNamingAThingAnOperatorWorksWithIsMarkedWithWhatItRefersTo(t *testing.T) {
	s, err := dispatch.Parse(Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	marked := map[string]int{}
	for _, name := range sortedOperationNames(s) {
		for _, a := range s.Operations[name].Args {
			want, isReference := markedReferences[a.As]
			isReference = isReference && !a.Ignored
			if isReference && a.Refers != want {
				t.Errorf("%s: argument %s refers %q, want %q", name, a.As, a.Refers, want)
			}
			if !isReference && a.Refers != "" {
				t.Errorf("%s: argument %s refers %q but is not one of the things an operator works with", name, a.As, a.Refers)
			}
			if isReference {
				marked[a.As]++
			}
		}
	}
	for as := range markedReferences {
		if marked[as] == 0 {
			t.Errorf("no operation declares %s, so nothing proves it is marked", as)
		}
	}
}

func TestReferencesListsTheMarkedArgumentsAndNoScopeOrJoinRowId(t *testing.T) {
	s, err := dispatch.Parse(Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	references := s.References()
	if len(references) < 27 {
		t.Fatalf("References lists %d arguments, want every goal, objective, workflow, work item, role, value and rule id the file declares", len(references))
	}
	seen := map[string]bool{}
	for _, r := range references {
		seen[r.As] = true
		if markedReferences[r.As] != r.Refers {
			t.Errorf("%s.%s is listed as referring %q", r.Operation, r.As, r.Refers)
		}
	}
	var missing []string
	for as := range markedReferences {
		if !seen[as] {
			missing = append(missing, as)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("References never lists %v", missing)
	}
	for _, scope := range []string{"draft_id", "assignment_id", "attachment_id", "module_id", "instance_id"} {
		if seen[scope] {
			t.Errorf("%s is a scope or a join row, not a reference", scope)
		}
	}
}

func sortedOperationNames(s *dispatch.Spec) []string {
	names := make([]string, 0, len(s.Operations))
	for name := range s.Operations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
