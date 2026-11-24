package mwanachamaagency

import "github.com/aosanya/mwanachama-backend-agency/models"

const legacyAccountableLabel = "accountable"

func foldLegacyRoleAssignments(doc models.AgencyDocument) models.AgencyDocument {
	if len(doc.RoleAssignments) == 0 {
		return doc
	}
	steps := make([]models.DocumentWorkItem, len(doc.WorkItems))
	copy(steps, doc.WorkItems)
	stepAt := make(map[string]int, len(steps))
	for i, s := range steps {
		stepAt[s.ID] = i
	}
	roleKnown := make(map[string]bool, len(doc.Roles))
	for _, r := range doc.Roles {
		roleKnown[r.ID] = true
	}
	for _, a := range doc.RoleAssignments {
		at, stepKnown := stepAt[a.WorkItemID]
		if a.WorkItemID == "" || !stepKnown || !roleKnown[a.RoleID] || a.Label != legacyAccountableLabel {
			continue
		}
		step := &steps[at]
		if step.AccountableRoleID != "" || step.RoleID == a.RoleID {
			continue
		}
		step.AccountableRoleID = a.RoleID
	}
	doc.WorkItems = steps
	doc.RoleAssignments = nil
	return doc
}
