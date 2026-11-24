package mwanachamaagency_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type olderDatabase struct {
	db     *gorm.DB
	s      *spec.Spec
	mgr    mwanachamaagency.AgencyManager
	draftA string
	draftB string
	w1, w2 models.Workflow
	w3     models.Workflow
	o1, o2 models.Objective
	items  string
	links  string
}

func newOlderDatabase(t *testing.T) olderDatabase {
	t.Helper()
	db := openDB(t)
	s, err := mwanachamaagency.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, s)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	ctx := t.Context()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Older"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	a, err := mgr.CreateDraft(ctx, "a", "", "")
	if err != nil {
		t.Fatalf("CreateDraft a: %v", err)
	}
	b, err := mgr.CreateDraft(ctx, "b", "", "")
	if err != nil {
		t.Fatalf("CreateDraft b: %v", err)
	}
	o := olderDatabase{
		db: db, s: s, mgr: mgr, draftA: a.ID, draftB: b.ID,
		items: tableOf(t, s, "work_item"),
		links: tableOf(t, s, "workflow_objective"),
	}
	g, _ := mgr.CreateGoal(ctx, models.Goal{DraftID: a.ID, Title: "G"})
	o.o1, _ = mgr.CreateObjective(ctx, models.Objective{DraftID: a.ID, GoalID: g.ID, Title: "O1"})
	o.o2, _ = mgr.CreateObjective(ctx, models.Objective{DraftID: a.ID, GoalID: g.ID, Title: "O2"})
	o.w1, _ = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: a.ID, Name: "W1"})
	o.w2, _ = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: a.ID, Name: "W2"})
	o.w3, _ = mgr.CreateWorkflow(ctx, models.Workflow{DraftID: a.ID, Name: "W3"})
	wb, _ := mgr.CreateWorkflow(ctx, models.Workflow{DraftID: b.ID, Name: "WB"})

	if err := db.Exec(`alter table ` + o.items + ` add column objective_id text`).Error; err != nil {
		t.Fatalf("give the steps their older objective_id column: %v", err)
	}
	step := func(draftID, workflowID, objectiveID string, deleted bool) {
		created, err := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: draftID, WorkflowID: workflowID, Title: "s"})
		if err != nil {
			t.Fatalf("CreateWorkItem: %v", err)
		}
		if err := db.Exec(`update `+o.items+` set objective_id = ?, deleted = ? where id = ?`, objectiveID, deleted, created.ID).Error; err != nil {
			t.Fatalf("plant the older objective_id: %v", err)
		}
	}
	step(a.ID, o.w1.ID, o.o1.ID, false)
	step(a.ID, o.w1.ID, o.o1.ID, false)
	step(a.ID, o.w1.ID, o.o2.ID, false)
	step(a.ID, o.w2.ID, o.o1.ID, false)
	step(a.ID, o.w2.ID, "", false)
	step(a.ID, o.w3.ID, o.o2.ID, true)
	step(a.ID, o.w3.ID, "no-such-objective", false)
	step(b.ID, wb.ID, o.o1.ID, false)
	foreign, _ := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: a.ID, WorkflowID: o.w3.ID, Title: "moved"})
	if err := db.Exec(`update `+o.items+` set workflow_id = ?, objective_id = ? where id = ?`, wb.ID, o.o1.ID, foreign.ID).Error; err != nil {
		t.Fatalf("plant a step naming another draft's workflow: %v", err)
	}
	missing, _ := mgr.CreateWorkItem(ctx, models.WorkItem{DraftID: a.ID, WorkflowID: o.w3.ID, Title: "orphan"})
	if err := db.Exec(`update `+o.items+` set workflow_id = 'no-such-workflow', objective_id = ? where id = ?`, o.o1.ID, missing.ID).Error; err != nil {
		t.Fatalf("plant a step naming a missing workflow: %v", err)
	}

	if err := db.Exec(`delete from ` + o.links).Error; err != nil {
		t.Fatalf("clear the links an older database never had: %v", err)
	}
	return o
}

func (o olderDatabase) pairs(t *testing.T) []string {
	t.Helper()
	var rows []struct{ DraftID, WorkflowID, ObjectiveID string }
	if err := o.db.Raw(`select draft_id, workflow_id, objective_id from ` + o.links).Scan(&rows).Error; err != nil {
		t.Fatalf("read the links: %v", err)
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, fmt.Sprintf("%s|%s|%s", r.DraftID, r.WorkflowID, r.ObjectiveID))
	}
	sort.Strings(out)
	return out
}

func (o olderDatabase) wantPairs() []string {
	want := []string{
		fmt.Sprintf("%s|%s|%s", o.draftA, o.w1.ID, o.o1.ID),
		fmt.Sprintf("%s|%s|%s", o.draftA, o.w1.ID, o.o2.ID),
		fmt.Sprintf("%s|%s|%s", o.draftA, o.w2.ID, o.o1.ID),
	}
	sort.Strings(want)
	return want
}

func (o olderDatabase) objectivelessWorkflows(t *testing.T) []string {
	t.Helper()
	issues, err := o.mgr.ListIssues(t.Context(), o.draftA, mwanachamaagency.IssueFilter{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	var names []string
	for _, i := range issues {
		if i.EntityType == "workflow" && strings.Contains(i.Message, "has no Objective") {
			names = append(names, i.EntityID)
		}
	}
	return names
}

func TestProvisionCarriesStepObjectivesForwardIntoWorkflowObjectiveLinks(t *testing.T) {
	o := newOlderDatabase(t)

	if err := mwanachamaagency.Provision(o.db, o.s); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	got, want := o.pairs(t), o.wantPairs()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("links = %v, want %v", got, want)
	}
	objectiveless := o.objectivelessWorkflows(t)
	for _, id := range objectiveless {
		if id == o.w1.ID || id == o.w2.ID {
			t.Errorf("workflow %s still reports no Objective after Provision", id)
		}
	}
	if len(objectiveless) != 1 || objectiveless[0] != o.w3.ID {
		t.Errorf("workflows with no Objective = %v, want only %s whose steps are deleted, foreign or unlinked", objectiveless, o.w3.ID)
	}
}

func TestProvisionCarryingStepObjectivesForwardTwiceAddsNoLink(t *testing.T) {
	o := newOlderDatabase(t)

	for i := 0; i < 3; i++ {
		if err := mwanachamaagency.Provision(o.db, o.s); err != nil {
			t.Fatalf("Provision %d: %v", i+1, err)
		}
	}

	if got := o.pairs(t); len(got) != 3 {
		t.Fatalf("links = %v, want 3 after repeated Provision", got)
	}
}

func TestProvisionCarriesStepObjectivesForwardWhenTheLinkTableDoesNotExistYet(t *testing.T) {
	o := newOlderDatabase(t)
	if err := o.db.Exec(`drop table ` + o.links).Error; err != nil {
		t.Fatalf("drop the link table an older database never had: %v", err)
	}

	if err := mwanachamaagency.Provision(o.db, o.s); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	got, want := o.pairs(t), o.wantPairs()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("links = %v, want %v", got, want)
	}
}

func TestProvisionKeepsStepObjectiveIDs(t *testing.T) {
	o := newOlderDatabase(t)

	if err := mwanachamaagency.Provision(o.db, o.s); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	var withObjective int64
	if err := o.db.Raw(`select count(*) from ` + o.items + ` where objective_id is not null and objective_id <> ''`).Scan(&withObjective).Error; err != nil || withObjective != 9 {
		t.Errorf("steps still carrying objective_id = %d, err %v, want 9", withObjective, err)
	}
}

func TestProvisionCollapsesDuplicateWorkflowObjectiveLinksKeepingTheOldestBeforeTheUniqueIndex(t *testing.T) {
	o := newOlderDatabase(t)
	if err := mwanachamaagency.Provision(o.db, o.s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	var index string
	if err := o.db.Raw(`select name from sqlite_master where type = 'index' and tbl_name = ? and name like '%wf_obj_unique%'`, o.links).Scan(&index).Error; err != nil || index == "" {
		t.Fatalf("the unique index is missing: %q %v", index, err)
	}
	if err := o.db.Exec(`drop index ` + index).Error; err != nil {
		t.Fatalf("drop the unique index an earlier build never had: %v", err)
	}
	insert := func(id, created string) {
		t.Helper()
		err := o.db.Exec(`insert into `+o.links+` (id, draft_id, workflow_id, objective_id, created_at, last_updated, deleted) values (?, ?, ?, ?, ?, ?, false)`,
			id, o.draftA, o.w1.ID, o.o1.ID, created, created).Error
		if err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}
	insert("newest", "2026-03-01T00:00:00Z")
	insert("oldest", "2026-01-01T00:00:00Z")
	insert("middle", "2026-02-01T00:00:00Z")

	if err := mwanachamaagency.Provision(o.db, o.s); err != nil {
		t.Fatalf("Provision over a database that already holds duplicate links: %v", err)
	}

	var ids []string
	if err := o.db.Raw(`select id from `+o.links+` where workflow_id = ? and objective_id = ?`, o.w1.ID, o.o1.ID).Scan(&ids).Error; err != nil {
		t.Fatalf("read the pair: %v", err)
	}
	if len(ids) != 1 || ids[0] != "oldest" {
		t.Errorf("rows for the pair = %v, want only the oldest", ids)
	}
	err := o.db.Exec(`insert into `+o.links+` (id, draft_id, workflow_id, objective_id) values ('again', ?, ?, ?)`, o.draftA, o.w1.ID, o.o1.ID).Error
	if err == nil {
		t.Errorf("the unique index was not recreated: a second row for the pair went in")
	}
}
