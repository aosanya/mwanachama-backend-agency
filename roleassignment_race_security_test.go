package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG53_OpenHole_TwoAccountablesHoldOneObjective(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, o, owner := seedPublishableAgency(t, mgr, ctx)

	rival, err := mgr.CreateRole(ctx, models.Role{DraftID: o.DraftID, Name: "rival"})
	if err != nil {
		t.Fatalf("precondition: CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: o.DraftID, RoleID: rival.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable,
	}); !errors.Is(err, mwanachamaagency.ErrDuplicateAccountable) {
		t.Fatalf("precondition: a sequential second Accountable should be refused with ErrDuplicateAccountable, got %v", err)
	}

	insertErr := db.WithContext(ctx).Table(tableOf(t, s, "role_assignment")).Create(map[string]any{
		"id":           "concurrent-accountable",
		"draft_id":     o.DraftID,
		"role_id":      rival.ID,
		"objective_id": o.ID,
		"label":        string(models.RACILabelAccountable),
		"created_at":   models.NowRFC3339(),
		"last_updated": models.NowRFC3339(),
	}).Error

	held, err := mgr.ListRoleAssignmentsForObjective(ctx, o.ID, o.DraftID, mwanachamaagency.RoleAssignmentFilter{Label: string(models.RACILabelAccountable)})
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForObjective: %v", err)
	}
	_, publishErr := mgr.PublishAgency(ctx, o.DraftID, "two-accountables", "production")

	if insertErr == nil && !errors.Is(publishErr, mwanachamaagency.ErrPublishValidationFailed) {
		t.Errorf("AG53 (P2) open hole: a second Accountable row on objective %s, written as a concurrent CreateRoleAssignment writes it after both counted zero, was accepted by the store (insert error %v), the objective now holds %d Accountables (%s and %s), and PublishAgency answered %v; want the store to refuse the second row, or the publish gate to refuse an objective with more than one Accountable", o.ID, insertErr, len(held), owner.ID, rival.ID, publishErr)
	}
}
