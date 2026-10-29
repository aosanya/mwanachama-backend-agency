package mwanachamaagency

import (
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type TableNames struct {
	Agencies           string
	Drafts             string
	Goals              string
	Objectives         string
	Workflows          string
	WorkItems          string
	EventFlowSteps     string
	AgencyPublications string
	Roles              string
	RoleAssignments    string
	Modules            string
	ObjectiveModules   string
	Values             string
	ValueAttachments   string
	Rules              string
	RuleAttachments    string
	Issues             string
	CodeSequences      string
	Records            string
	RecordRows         string
}

func TablesFor(s *spec.Spec) TableNames {
	table := func(role string) string {
		o, ok := s.ByRole(role)
		if !ok {
			return ""
		}
		return s.TableFor(o)
	}
	return TableNames{
		Agencies:           table(roleAgency),
		Drafts:             table(roleDraft),
		Goals:              table(roleGoal),
		Objectives:         table(roleObjective),
		Workflows:          table(roleWorkflow),
		WorkItems:          table(roleWorkItem),
		EventFlowSteps:     table(roleEventFlowStep),
		AgencyPublications: table(rolePublication),
		Roles:              table(roleRole),
		RoleAssignments:    table(roleRoleAssignment),
		Modules:            table(roleModule),
		ObjectiveModules:   table(roleObjectiveModule),
		Values:             table(roleValue),
		ValueAttachments:   table(roleValueAttachment),
		Rules:              table(roleRule),
		RuleAttachments:    table(roleRuleAttachment),
		Issues:             table(roleIssue),
		CodeSequences:      table(roleCodeSequence),
		Records:            table(roleRecord),
		RecordRows:         table(roleRecordRow),
	}
}
