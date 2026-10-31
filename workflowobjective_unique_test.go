package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func TestRV1_WorkflowObjectiveHasNoUniqueIndex(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	seed := seedWorkflowObjectives(t, mgr, ctx)
	if _, err := mgr.CreateWorkflowObjective(ctx, seed.link(seed.workflowA, seed.objectiveA)); err != nil {
		t.Fatal(err)
	}
	table := tableOf(t, s, "workflow_objective")
	err := db.Table(table).Create(map[string]any{
		"id": "dup-row", "draft_id": seed.draftID, "workflow_id": seed.workflowA.ID,
		"objective_id": seed.objectiveA.ID, "created_at": "2026-01-01T00:00:00Z", "last_updated": "2026-01-01T00:00:00Z", "deleted": false,
	}).Error
	if err == nil {
		t.Fatalf("a second (workflow, objective) row went in at the storage layer: only the app-level count guards the pair, so two concurrent writers on a read-committed database both pass it")
	}
}

func TestCreateWorkflowObjective_ARowThatLandsAfterTheCheckReadsAsDuplicateNotConflict(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	seed := seedWorkflowObjectives(t, mgr, ctx)
	table := tableOf(t, s, "workflow_objective")
	planted := false
	err := db.Callback().Create().Before("gorm:create").Register("land_a_rival_link", func(tx *gorm.DB) {
		if planted || tx.Statement.Table != table {
			return
		}
		planted = true
		_, rivalErr := tx.Statement.ConnPool.ExecContext(ctx,
			`insert into `+table+` (id, draft_id, workflow_id, objective_id, created_at, last_updated, deleted) values ('rival', ?, ?, ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', false)`,
			seed.draftID, seed.workflowA.ID, seed.objectiveA.ID)
		if rivalErr != nil {
			t.Errorf("land the rival link: %v", rivalErr)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.CreateWorkflowObjective(ctx, seed.link(seed.workflowA, seed.objectiveA))

	if !planted {
		t.Fatal("the rival link never landed")
	}
	if !errors.Is(err, mwanachamaagency.ErrDuplicateWorkflowObjective) {
		t.Fatalf("a link that loses to a rival written after the check: %v, want ErrDuplicateWorkflowObjective", err)
	}
}
