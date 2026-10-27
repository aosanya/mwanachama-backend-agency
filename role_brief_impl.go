package mwanachamaagency

import (
	"context"
	"fmt"
	"strings"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const briefScopeOrganization = "organization"

func (m *agencyManager) GetRoleBrief(ctx context.Context, roleID string) (models.RoleBrief, error) {
	pub, err := m.GetPublicationForChannel(ctx, models.DefaultChannel)
	if err != nil {
		return models.RoleBrief{}, err
	}
	return briefFromDocument(pub, roleID)
}

func targetKey(t models.TargetType, code string) string {
	return string(t) + "/" + code
}

func briefFromDocument(pub models.AgencyPublication, roleID string) (models.RoleBrief, error) {
	doc := pub.Document
	var role *models.DocumentRole
	for i := range doc.Roles {
		if doc.Roles[i].ID == roleID || doc.Roles[i].Code == roleID {
			role = &doc.Roles[i]
			break
		}
	}
	if role == nil {
		return models.RoleBrief{}, ErrRoleNotFound
	}
	brief := models.RoleBrief{
		PublicationVersion: pub.Version,
		PublicationTag:     pub.Tag,
		RoleID:             role.ID,
		RoleCode:           role.Code,
		Name:               role.Name,
		Description:        role.Description,
	}

	goals := map[string]models.DocumentGoal{}
	for _, g := range doc.Goals {
		goals[g.ID] = g
	}
	objectives := map[string]models.DocumentObjective{}
	for _, o := range doc.Objectives {
		objectives[o.ID] = o
	}

	scopeByTarget := map[string]string{targetKey(models.TargetTypeRole, role.Code): "seat"}
	objectiveIDs := map[string]bool{}
	for _, a := range doc.RoleAssignments {
		o, ok := objectives[a.ObjectiveID]
		if a.RoleID != role.ID || !ok || objectiveIDs[o.ID] {
			continue
		}
		objectiveIDs[o.ID] = true
		g := goals[o.GoalID]
		brief.Objectives = append(brief.Objectives, models.BriefObjective{
			Code: o.Code, Title: o.Title, Description: o.Description, Label: a.Label,
			GoalCode: g.Code, GoalTitle: g.Title, GoalDescription: g.Description,
		})
		scopeByTarget[targetKey(models.TargetTypeObjective, o.Code)] = "objective " + o.Code
		if g.Code != "" {
			scopeByTarget[targetKey(models.TargetTypeGoal, g.Code)] = "goal " + g.Code
		}
	}

	brief.Values = briefStatements(doc.Values, doc.ValueAttachments, func(a models.DocumentAttachment) string { return a.ValueCode }, scopeByTarget)
	brief.Rules = briefStatements(doc.Rules, doc.RuleAttachments, func(a models.DocumentAttachment) string { return a.RuleCode }, scopeByTarget)

	for _, w := range doc.WorkItems {
		o, ok := objectives[w.ObjectiveID]
		if !ok || !objectiveIDs[o.ID] || w.Prompt == "" {
			continue
		}
		brief.WorkItems = append(brief.WorkItems, models.BriefWorkItem{
			Code: w.Code, Title: w.Title, ObjectiveCode: o.Code, Prompt: w.Prompt,
		})
	}

	brief.Text = renderBrief(brief)
	return brief, nil
}

func briefStatements(items []models.DocumentStatement, attachments []models.DocumentAttachment, codeOf func(models.DocumentAttachment) string, scopeByTarget map[string]string) []models.BriefStatement {
	attached := map[string][]models.DocumentAttachment{}
	for _, a := range attachments {
		attached[codeOf(a)] = append(attached[codeOf(a)], a)
	}
	var out []models.BriefStatement
	for _, it := range items {
		as := attached[it.Code]
		if len(as) == 0 {
			out = append(out, models.BriefStatement{Code: it.Code, Text: it.Text, Scope: briefScopeOrganization})
			continue
		}
		var scopes []string
		for _, a := range as {
			if s, ok := scopeByTarget[a.TargetType+"/"+a.TargetCode]; ok {
				scopes = append(scopes, s)
			}
		}
		if len(scopes) > 0 {
			out = append(out, models.BriefStatement{Code: it.Code, Text: it.Text, Scope: strings.Join(scopes, ", ")})
		}
	}
	return out
}

func renderBrief(b models.RoleBrief) string {
	var s strings.Builder
	fmt.Fprintf(&s, "Seat: %s (%s)\n", b.Name, b.RoleCode)
	if b.Description != "" {
		fmt.Fprintf(&s, "%s\n", b.Description)
	}
	s.WriteString("\nObjectives you are answerable for:\n")
	if len(b.Objectives) == 0 {
		s.WriteString("- none\n")
	}
	for _, o := range b.Objectives {
		fmt.Fprintf(&s, "- %s %s [%s] under goal %s %s\n", o.Code, o.Title, o.Label, o.GoalCode, o.GoalTitle)
	}
	writeStatements(&s, "Values", b.Values)
	writeStatements(&s, "Rules", b.Rules)
	s.WriteString("\nWork items:\n")
	if len(b.WorkItems) == 0 {
		s.WriteString("- none\n")
	}
	for _, w := range b.WorkItems {
		fmt.Fprintf(&s, "- %s %s (objective %s): %s\n", w.Code, w.Title, w.ObjectiveCode, w.Prompt)
	}
	return s.String()
}

func writeStatements(s *strings.Builder, heading string, items []models.BriefStatement) {
	fmt.Fprintf(s, "\n%s:\n", heading)
	if len(items) == 0 {
		s.WriteString("- none\n")
	}
	for _, it := range items {
		fmt.Fprintf(s, "- %s (%s): %s\n", it.Code, it.Scope, it.Text)
	}
}
