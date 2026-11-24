package models

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrHomeRefUnmatched = errors.New("matches no part")
	ErrHomeRefAmbiguous = errors.New("matches more than one part")
)

type HomePart struct {
	Code string
	Alt  string
	Name string
}

type HomeStepPart struct {
	Code         string
	Name         string
	WorkflowCode string
}

type HomeParts struct {
	Records   []HomePart
	Workflows []HomePart
	Steps     []HomeStepPart
	Roles     []HomePart
}

func ResolveHomePart(parts []HomePart, ref string) (HomePart, error) {
	ref = strings.TrimSpace(ref)
	for _, part := range parts {
		if part.Code == ref || (part.Alt != "" && part.Alt == ref) {
			return part, nil
		}
	}
	var found []HomePart
	for _, part := range parts {
		if strings.TrimSpace(part.Name) != "" && strings.EqualFold(strings.TrimSpace(part.Name), ref) {
			found = append(found, part)
		}
	}
	switch len(found) {
	case 0:
		return HomePart{}, ErrHomeRefUnmatched
	case 1:
		return found[0], nil
	}
	return HomePart{}, ErrHomeRefAmbiguous
}

func (p HomeParts) Step(workflowCode, ref string) (HomeStepPart, error) {
	ref = strings.TrimSpace(ref)
	for _, step := range p.Steps {
		if step.Code == ref {
			return step, nil
		}
	}
	var held []HomePart
	byCode := map[string]HomeStepPart{}
	for _, step := range p.Steps {
		if step.WorkflowCode == workflowCode {
			held = append(held, HomePart{Code: step.Code, Name: step.Name})
			byCode[step.Code] = step
		}
	}
	part, err := ResolveHomePart(held, ref)
	if err != nil {
		return HomeStepPart{}, err
	}
	return byCode[part.Code], nil
}

func HomePartsOf(doc AgencyDocument) HomeParts {
	parts := HomeParts{}
	workflowCodes := map[string]string{}
	for _, record := range doc.Records {
		if record.Deleted {
			continue
		}
		parts.Records = append(parts.Records, HomePart{Code: record.Code, Alt: record.Key, Name: record.Name})
	}
	for _, workflow := range doc.Workflows {
		if workflow.Deleted {
			continue
		}
		parts.Workflows = append(parts.Workflows, HomePart{Code: workflow.Code, Name: workflow.Name})
		workflowCodes[workflow.ID] = workflow.Code
	}
	for _, step := range doc.WorkItems {
		if step.Deleted {
			continue
		}
		parts.Steps = append(parts.Steps, HomeStepPart{Code: step.Code, Name: step.Title, WorkflowCode: workflowCodes[step.WorkflowID]})
	}
	for _, role := range doc.Roles {
		if role.Deleted {
			continue
		}
		parts.Roles = append(parts.Roles, HomePart{Code: role.Code, Name: role.Name})
	}
	return parts
}

func (p HomeParts) CheckHomeRoles(nodes []DocumentHomeNode) error {
	for _, n := range nodes {
		if err := p.checkRoles(n); err != nil {
			return err
		}
		if err := p.CheckHomeRoles(n.Entries); err != nil {
			return err
		}
	}
	return nil
}

func (p HomeParts) CheckHome(nodes []DocumentHomeNode) error {
	for _, n := range nodes {
		if err := p.checkHomeNode(n); err != nil {
			return err
		}
		if err := p.CheckHome(n.Entries); err != nil {
			return err
		}
	}
	return nil
}

func (p HomeParts) checkRoles(n DocumentHomeNode) error {
	for _, role := range n.Roles {
		if _, err := ResolveHomePart(p.Roles, role); err != nil {
			return homeRefProblem("role", n.Key, role, err)
		}
	}
	return nil
}

func (p HomeParts) checkHomeNode(n DocumentHomeNode) error {
	if err := p.checkRoles(n); err != nil {
		return err
	}
	if n.PageOf != "" {
		if _, err := ResolveHomePart(p.Records, n.PageOf); err != nil {
			return homeRefProblem("record", n.Key, n.PageOf, err)
		}
	}
	if n.Target == nil {
		return nil
	}
	switch n.Target.Type {
	case HomeTargetRecord:
		if _, err := ResolveHomePart(p.Records, n.Target.Ref); err != nil {
			return homeRefProblem("record", n.Key, n.Target.Ref, err)
		}
	case HomeTargetWorkflow:
		workflow, err := ResolveHomePart(p.Workflows, n.Target.Ref)
		if err != nil {
			return homeRefProblem("workflow", n.Key, n.Target.Ref, err)
		}
		if n.Target.Step != "" {
			if _, err := p.Step(workflow.Code, n.Target.Step); err != nil {
				return homeRefProblem("step", n.Key, n.Target.Step, err)
			}
		}
	}
	return nil
}

func homeRefProblem(kind, key, ref string, err error) error {
	return fmt.Errorf("the home entry %q names the %s %q, which %w", key, kind, ref, err)
}

func (p HomeParts) RecordKeyOf(ref string) string {
	part, err := ResolveHomePart(p.Records, ref)
	switch {
	case err != nil:
		return ref
	case part.Alt != "":
		return part.Alt
	}
	return part.Code
}

func (p HomeParts) KeyRecordRefs(nodes []DocumentHomeNode) []DocumentHomeNode {
	if nodes == nil {
		return nil
	}
	out := make([]DocumentHomeNode, len(nodes))
	for i, n := range nodes {
		if n.PageOf != "" {
			n.PageOf = p.RecordKeyOf(n.PageOf)
		}
		if n.Target != nil {
			target := *n.Target
			if target.Type == HomeTargetRecord && target.Ref != "" {
				target.Ref = p.RecordKeyOf(target.Ref)
			}
			n.Target = &target
		}
		n.Entries = p.KeyRecordRefs(n.Entries)
		out[i] = n
	}
	return out
}

func (p HomeParts) KeyRecordRefsOfEntry(e HomeEntry) HomeEntry {
	if e.PageOf != "" {
		e.PageOf = p.RecordKeyOf(e.PageOf)
	}
	if e.TargetType == HomeTargetRecord && e.Target != "" {
		e.Target = p.RecordKeyOf(e.Target)
	}
	return e
}
