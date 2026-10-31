package mwanachamaagency_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestConcurrentUpdateRoleAssignmentToOneLabelClassifiesTheLoser(t *testing.T) {
	mgr := newSharedCacheManager(t, "ru")
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)
	for round := 0; round < 40; round++ {
		step, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: s.draftID, WorkflowID: s.step.WorkflowID, Title: fmt.Sprint("s", round)})
		if err != nil {
			t.Fatal(err)
		}
		a1, _ := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: step.ID, RoleID: s.roleA.ID, Label: models.RACILabelResponsible})
		a2, _ := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: step.ID, RoleID: s.roleA.ID, Label: models.RACILabelConsulted})
		var wg sync.WaitGroup
		errs := make([]error, 2)
		start := make(chan struct{})
		for i, a := range []models.RoleAssignment{a1, a2} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				a.Label = models.RACILabelAccountable
				_, errs[i] = mgr.UpdateRoleAssignment(ctx, a)
			}()
		}
		close(start)
		wg.Wait()
		for _, e := range errs {
			if e != nil && !errors.Is(e, mwanachamaagency.ErrDuplicateRoleAssignment) && !errors.Is(e, mwanachamaagency.ErrConflict) {
				t.Fatalf("round %d: losing updater saw unclassified %v", round, e)
			}
		}
	}
}

func TestCreateRoleAssignmentRacingDeleteRoleLeavesNoAssignmentOfADeletedRole(t *testing.T) {
	mgr := newSharedCacheManager(t, "rd")
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)
	for round := 0; round < 300; round++ {
		role, err := mgr.CreateRole(ctx, models.Role{DraftID: s.draftID, Name: fmt.Sprint("r", round)})
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			mgr.CreateRoleAssignment(ctx, models.RoleAssignment{DraftID: s.draftID, WorkItemID: s.step.ID, RoleID: role.ID, Label: models.RACILabelInformed})
		}()
		go func() {
			defer wg.Done()
			<-start
			mgr.DeleteRole(ctx, role.ID)
		}()
		close(start)
		wg.Wait()
		held, _ := mgr.ListRoleAssignments(ctx, s.draftID, mwanachamaagency.RoleAssignmentFilter{RoleID: role.ID})
		if _, err := mgr.GetRole(ctx, role.ID); err != nil && len(held) > 0 {
			t.Fatalf("round %d: a role_assignment survives for a deleted role", round)
		}
	}
}
