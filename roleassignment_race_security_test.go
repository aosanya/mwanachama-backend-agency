package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestConcurrentCreateRoleAssignmentWithOneRoleAndLabelStoresOneAndClassifiesTheRest(t *testing.T) {
	mgr := newSharedCacheManager(t, "rr")
	ctx := context.Background()
	s := seedRaci(t, mgr, ctx)

	for round, label := range []models.RACILabel{models.RACILabelResponsible, models.RACILabelAccountable, models.RACILabelConsulted, models.RACILabelInformed} {
		created, refused := raceWriters(6, func() error {
			_, err := mgr.CreateRoleAssignment(ctx, s.assignment(s.roleA.ID, label))
			return err
		})
		if created != 1 {
			t.Fatalf("round %d: %d assignments stored for role %s label %s on one step, want exactly 1", round, created, s.roleA.ID, label)
		}
		for _, err := range refused {
			if !errors.Is(err, mwanachamaagency.ErrDuplicateRoleAssignment) && !errors.Is(err, mwanachamaagency.ErrConflict) {
				t.Errorf("round %d: the losing writer saw %v, want ErrDuplicateRoleAssignment or ErrConflict", round, err)
			}
		}
		held, err := mgr.ListRoleAssignmentsForWorkItem(ctx, s.step.ID, s.draftID, mwanachamaagency.RoleAssignmentFilter{Label: string(label)})
		if err != nil {
			t.Fatalf("ListRoleAssignmentsForWorkItem: %v", err)
		}
		if len(held) != 1 {
			t.Fatalf("round %d: the step holds %d %s assignments for the role, want 1", round, len(held), label)
		}
	}
}
