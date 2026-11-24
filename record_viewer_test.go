package mwanachamaagency_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func paymentsSeenByFarmers(w stepWorld, workflow models.Workflow) models.Record {
	r := paymentsRecord(w, workflow)
	r.ViewerRoleID = w.roles[farmer].ID
	return r
}

func TestRecordViewer_IsKeptOnTheRecordAndMayBeLeftEmptyOrCleared(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")

	bare, err := w.mgr.CreateRecord(ctx, paymentsRecord(w, wf))
	if err != nil || bare.ViewerRoleID != "" {
		t.Fatalf("a record with no viewer: %+v, %v", bare, err)
	}

	bare.ViewerRoleID = w.roles[farmer].ID
	set, err := w.mgr.UpdateRecord(ctx, bare)
	if err != nil || set.ViewerRoleID != w.roles[farmer].ID {
		t.Fatalf("UpdateRecord with a viewer = %+v, %v", set, err)
	}
	got, err := w.mgr.GetRecord(ctx, bare.ID)
	if err != nil || got.ViewerRoleID != w.roles[farmer].ID || got.RoleID != w.roles[coordinator].ID {
		t.Fatalf("GetRecord = %+v, %v; want the Farmer sees and the Coordinator adds", got, err)
	}

	got.ViewerRoleID = ""
	cleared, err := w.mgr.UpdateRecord(ctx, got)
	if err != nil || cleared.ViewerRoleID != "" {
		t.Fatalf("UpdateRecord clearing the viewer = %+v, %v", cleared, err)
	}
}

func TestRecordViewer_IsRefusedWhenItIsNotOneRoleOfTheDraft(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")
	base := paymentsSeenByFarmers(w, wf)

	two := base
	two.ViewerRoleID = w.roles[farmer].ID + "," + w.roles[supplier].ID
	if _, err := w.mgr.CreateRecord(ctx, two); !errors.Is(err, mwanachamaagency.ErrInvalidRecord) {
		t.Errorf("two viewer roles: want ErrInvalidRecord, got %v", err)
	}

	missing := base
	missing.ViewerRoleID = "no-such-role"
	if _, err := w.mgr.CreateRecord(ctx, missing); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("a viewer role that does not exist: want ErrRoleNotFound, got %v", err)
	}

	made, err := w.mgr.CreateRecord(ctx, base)
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	made.ViewerRoleID = "no-such-role"
	if _, err := w.mgr.UpdateRecord(ctx, made); !errors.Is(err, mwanachamaagency.ErrRoleNotFound) {
		t.Errorf("update to a viewer role that does not exist: want ErrRoleNotFound, got %v", err)
	}
}

func TestRecordViewer_AForkAndAPublicationCarryItOntoTheCopiedRole(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")
	seen, err := w.mgr.CreateRecord(ctx, paymentsSeenByFarmers(w, wf))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	mayAdd(t, w.mgr, seen)
	step := w.item(wf, 1, groupOrderSteps[0])
	madeStep, err := w.mgr.CreateWorkItem(ctx, step)
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	assignOn(t, w.mgr, ctx, w.draftID, w.roles[coordinator], madeStep)

	viewerNameIn := func(draftID string) string {
		records, err := w.mgr.ListRecords(ctx, draftID, mwanachamaagency.RecordFilter{})
		if err != nil || len(records) != 1 {
			t.Fatalf("ListRecords(%s) = %+v, %v", draftID, records, err)
		}
		roles, _ := w.mgr.ListRoles(ctx, draftID, mwanachamaagency.RoleFilter{})
		for _, r := range roles {
			if r.ID == records[0].ViewerRoleID {
				return r.Name
			}
		}
		return ""
	}

	fork, err := w.mgr.CreateDraft(ctx, "copy", w.draftID, "draft")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	if got := viewerNameIn(fork.ID); got != farmer {
		t.Errorf("the fork's record is seen by %q, want the forked %s", got, farmer)
	}

	pub, err := w.mgr.PublishAgency(ctx, w.draftID, "release-1", "")
	if err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if len(pub.Document.Records) != 1 || pub.Document.Records[0].ViewerRoleID != w.roles[farmer].ID {
		t.Fatalf("the publication document holds records %+v", pub.Document.Records)
	}
	back, err := w.mgr.CreateDraft(ctx, "revisit", "release-1", "publication")
	if err != nil {
		t.Fatalf("CreateDraft from the publication: %v", err)
	}
	if got := viewerNameIn(back.ID); got != farmer {
		t.Errorf("the draft forked from the publication has the record seen by %q, want the restored %s", got, farmer)
	}
}

func TestRecordViewer_ChangingOnlyTheViewerIsAChangeToPublish(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")
	record, err := w.mgr.CreateRecord(ctx, paymentsRecord(w, wf))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	mayAdd(t, w.mgr, record)
	step, err := w.mgr.CreateWorkItem(ctx, w.item(wf, 1, groupOrderSteps[0]))
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	assignOn(t, w.mgr, ctx, w.draftID, w.roles[coordinator], step)
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-1", ""); err != nil {
		t.Fatalf("PublishAgency: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-2", ""); !errors.Is(err, mwanachamaagency.ErrNoChangesDetected) {
		t.Fatalf("publishing the same content: want ErrNoChangesDetected, got %v", err)
	}
	record.ViewerRoleID = w.roles[farmer].ID
	if _, err := w.mgr.UpdateRecord(ctx, record); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if _, err := w.mgr.PublishAgency(ctx, w.draftID, "release-2", ""); err != nil {
		t.Fatalf("a changed viewer is a change: %v", err)
	}
}

func TestRecordViewer_TheIssueScanNamesAViewerRoleThatNoLongerExists(t *testing.T) {
	w := newStepWorld(t)
	ctx := context.Background()
	wf := w.workflow(t, "Group order")
	seen, err := w.mgr.CreateRecord(ctx, paymentsSeenByFarmers(w, wf))
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	mayAdd(t, w.mgr, seen)
	if err := w.mgr.DeleteRole(ctx, w.roles[farmer].ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	issues, err := w.mgr.ListIssues(ctx, w.draftID, mwanachamaagency.IssueFilter{EntityType: "record"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "is seen by Role") {
		t.Fatalf("issues = %+v, want one naming the missing viewer role", issues)
	}
}
