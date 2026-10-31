package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type raciSeed struct {
	draftID string
	step    models.WorkItem
	other   models.WorkItem
	roleA   models.Role
	roleB   models.Role
}

func seedRaci(t *testing.T, mgr mwanachamaagency.AgencyManager, ctx context.Context) raciSeed {
	t.Helper()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	w, err := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: d.ID, Name: "Onboarding"})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	step, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: d.ID, WorkflowID: w.ID, Title: "Write charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	other, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: d.ID, WorkflowID: w.ID, Title: "Review charter"})
	if err != nil {
		t.Fatalf("CreateWorkItem(other): %v", err)
	}
	roleA, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "bug-hunter"})
	if err != nil {
		t.Fatalf("CreateRole(A): %v", err)
	}
	roleB, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "tech-lead"})
	if err != nil {
		t.Fatalf("CreateRole(B): %v", err)
	}
	return raciSeed{draftID: d.ID, step: step, other: other, roleA: roleA, roleB: roleB}
}

func (s raciSeed) assignment(roleID string, label models.RACILabel) models.RoleAssignment {
	return models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.step.ID, RoleID: roleID, Label: label}
}

func TestCreateRoleAssignment_InvalidLabel(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	_, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, "bogus"))
	if !errors.Is(err, mwanachamaagency.ErrInvalidRACILabel) {
		t.Fatalf("expected ErrInvalidRACILabel, got %v", err)
	}
}

func TestCreateRoleAssignment_RoleNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	_, err := mgr.CreateRoleAssignment(ctx, s.assignment("unknown", models.RACILabelInformed))
	if !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestCreateRoleAssignment_WorkItemNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	_, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: "unknown", RoleID: s.roleA.ID, Label: models.RACILabelInformed})
	if !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Fatalf("expected ErrWorkItemNotFound, got %v", err)
	}
}

func TestCreateRoleAssignment_ScopeMismatch(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	d, err := mgr.CreateDraft(ctx, "second draft", s.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	draftRole, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "bug-hunter-two"})
	if err != nil {
		t.Fatalf("CreateRole(draft-scoped): %v", err)
	}

	_, err = mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: d.ID, WorkItemID: s.step.ID, RoleID: draftRole.ID, Label: models.RACILabelInformed})
	if !errors.Is(err, mwanachamaagency.ErrWorkItemNotFound) {
		t.Fatalf("expected ErrWorkItemNotFound for a step that lives in another draft, got %v", err)
	}

	_, err = mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: d.ID, WorkItemID: s.step.ID, RoleID: s.roleA.ID, Label: models.RACILabelInformed})
	if err == nil {
		t.Fatalf("expected a refusal for a role and step that live in another draft, got none")
	}
}

func TestCreateRoleAssignment_TwoAccountablesOnOneStepAreAllowed(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	if _, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelAccountable)); err != nil {
		t.Fatalf("CreateRoleAssignment(A accountable): %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleB.ID, models.RACILabelAccountable)); err != nil {
		t.Fatalf("a second Accountable on one step must be allowed, got %v", err)
	}

	held, err := mgr.ListRoleAssignmentsForWorkItem(ctx, s.step.ID, s.draftID, mwanachamaagency.RoleAssignmentFilter{Label: string(models.RACILabelAccountable)})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForWorkItem: %v", err)
	}
	if len(held) != 2 {
		t.Fatalf("expected 2 Accountable assignments on the step, got %d", len(held))
	}
}

func TestCreateRoleAssignment_OneRoleHoldsTwoLabelsOnOneStep(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	for _, label := range []models.RACILabel{models.RACILabelResponsible, models.RACILabelAccountable, models.RACILabelConsulted, models.RACILabelInformed} {
		if _, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, label)); err != nil {
			t.Fatalf("one role holding label %s alongside its other labels must be allowed, got %v", label, err)
		}
	}

	held, err := mgr.ListRoleAssignmentsForWorkItem(ctx, s.step.ID, s.draftID, mwanachamaagency.RoleAssignmentFilter{RoleID: s.roleA.ID})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForWorkItem: %v", err)
	}
	if len(held) != 4 {
		t.Fatalf("expected the role to hold 4 labels on the step, got %d", len(held))
	}
}

func TestCreateRoleAssignment_SameRoleAndLabelOnOneStepIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	if _, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelConsulted)); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	_, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelConsulted))
	if !errors.Is(err, mwanachamaagency.ErrDuplicateRoleAssignment) {
		t.Fatalf("expected ErrDuplicateRoleAssignment, got %v", err)
	}

	held, err := mgr.ListRoleAssignmentsForWorkItem(ctx, s.step.ID, s.draftID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForWorkItem: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("the refused duplicate left %d rows, want 1", len(held))
	}
}

func TestCreateRoleAssignment_SameRoleAndLabelOnAnotherStepIsAllowed(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	if _, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelConsulted)); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	onOther := models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.other.ID, RoleID: s.roleA.ID, Label: models.RACILabelConsulted}
	if _, err := mgr.CreateRoleAssignment(ctx, onOther); err != nil {
		t.Fatalf("the same role and label on a different step must be allowed, got %v", err)
	}
}

func TestUpdateRoleAssignment_ToADuplicateIsRefusedAndToItselfIsAllowed(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	if _, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelAccountable)); err != nil {
		t.Fatalf("CreateRoleAssignment(accountable): %v", err)
	}
	consulted, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelConsulted))
	if err != nil {
		t.Fatalf("CreateRoleAssignment(consulted): %v", err)
	}

	consulted.Label = models.RACILabelAccountable
	if _, err := mgr.UpdateRoleAssignment(ctx, consulted); !errors.Is(err, mwanachamaagency.ErrDuplicateRoleAssignment) {
		t.Fatalf("relabelling onto a role+label the step already holds: expected ErrDuplicateRoleAssignment, got %v", err)
	}

	consulted.Label = models.RACILabelConsulted
	if _, err := mgr.UpdateRoleAssignment(ctx, consulted); err != nil {
		t.Fatalf("saving a row unchanged must not collide with itself, got %v", err)
	}
}

func TestRoleAssignment_UpdateAndDelete(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)
	assignment, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelResponsible))
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}

	assignment.Label = models.RACILabelAccountable
	updated, err := mgr.UpdateRoleAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("UpdateRoleAssignment: %v", err)
	}
	if updated.Label != models.RACILabelAccountable || updated.WorkItemID != s.step.ID {
		t.Fatalf("unexpected updated assignment: %+v", updated)
	}

	if err := mgr.DeleteRoleAssignment(ctx, assignment.ID); err != nil {
		t.Fatalf("DeleteRoleAssignment: %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, assignment.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("expected ErrRoleAssignmentNotFound after delete, got %v", err)
	}
}

func TestListRoleAssignmentsForWorkItem(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	if _, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelResponsible)); err != nil {
		t.Fatalf("CreateRoleAssignment(A): %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleB.ID, models.RACILabelAccountable)); err != nil {
		t.Fatalf("CreateRoleAssignment(B): %v", err)
	}
	onOther := models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.other.ID, RoleID: s.roleA.ID, Label: models.RACILabelInformed}
	if _, err := mgr.CreateRoleAssignment(ctx, onOther); err != nil {
		t.Fatalf("CreateRoleAssignment(other step): %v", err)
	}

	assignments, err := mgr.ListRoleAssignmentsForWorkItem(ctx, s.step.ID, s.draftID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForWorkItem: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments for the step, got %d", len(assignments))
	}

	all, err := mgr.ListRoleAssignments(ctx, s.draftID, mwanachamaagency.RoleAssignmentFilter{WorkItemID: s.other.ID})
	if err != nil {
		t.Fatalf("ListRoleAssignments: %v", err)
	}
	if len(all) != 1 || all[0].WorkItemID != s.other.ID {
		t.Fatalf("expected only the other step's assignment, got %+v", all)
	}
}

func TestRoleAssignment_DeletingTheStepOrTheRoleTakesItsAssignments(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	onStep, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelResponsible))
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	onOther, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.other.ID, RoleID: s.roleB.ID, Label: models.RACILabelAccountable})
	if err != nil {
		t.Fatalf("CreateRoleAssignment(other): %v", err)
	}

	if err := mgr.DeleteWorkItem(ctx, s.step.ID); err != nil {
		t.Fatalf("DeleteWorkItem: %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, onStep.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("deleting the step must take its assignments, got %v", err)
	}

	if err := mgr.DeleteRole(ctx, s.roleB.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, onOther.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("deleting the role must take its assignments, got %v", err)
	}
}

func TestRoleAssignment_DeletingTheWorkflowTakesTheAssignmentsOfItsSteps(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	a, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, models.RACILabelResponsible))
	if err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	if err := mgr.DeleteWorkflow(ctx, s.step.WorkflowID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	if _, err := mgr.GetRoleAssignment(ctx, a.ID); !errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound) {
		t.Fatalf("deleting the workflow must take the assignments of its steps, got %v", err)
	}
	left, err := mgr.ListRoleAssignments(ctx, s.draftID, mwanachamaagency.RoleAssignmentFilter{})
	if err != nil {
		t.Fatalf("ListRoleAssignments: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected no assignments left, got %+v", left)
	}
}
