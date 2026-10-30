package routes_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG76_AGoalWithMoreObjectivesThanOneStatementCanBindCanNeverBeDeleted(t *testing.T) {
	const objectives = 32767
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()

	goal, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: "Goal planted with objectives"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	for i := 0; i < objectives; i++ {
		if _, err := mgr.CreateObjective(ctx, models.Objective{DraftID: draftID, GoalID: goal.ID, Title: fmt.Sprint("Objective ", i)}); err != nil {
			t.Fatalf("CreateObjective %d: %v", i, err)
		}
	}
	listed, err := mgr.ListObjectives(ctx, draftID, mwanachamaagency.ObjectiveFilter{})
	if err != nil {
		t.Fatalf("ListObjectives: %v", err)
	}
	if len(listed) != objectives {
		t.Fatalf("precondition: want %d objectives under the goal, got %d", objectives, len(listed))
	}

	del := httptest.NewRecorder()
	serve(mgr, del, httptest.NewRequest(http.MethodDelete, "/goals/"+goal.ID, nil))

	get := httptest.NewRecorder()
	serve(mgr, get, httptest.NewRequest(http.MethodGet, "/goals/"+goal.ID, nil))

	if del.Code != http.StatusNoContent || get.Code != http.StatusNotFound {
		t.Errorf("DELETE /goals/{id} on a goal holding %d objectives answered %d %s and GET then answered %d — want 204 and then 404; the cascade binds every objective id into one statement, so past the driver's bound-parameter ceiling the goal can never be deleted by anyone in the Instance",
			objectives, del.Code, del.Body.String(), get.Code)
	}
}
