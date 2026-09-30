package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG54_OpenHole_OneForkRequestCopiesEveryRowWithoutLimit(t *testing.T) {
	mgr, source := newTestManagerWithDraft(t)
	ctx := t.Context()

	const goals, forks = 200, 50
	for i := 0; i < goals; i++ {
		if _, err := mgr.CreateGoal(ctx, models.Goal{DraftID: source, Title: fmt.Sprintf("g%d", i)}); err != nil {
			t.Fatalf("precondition: CreateGoal: %v", err)
		}
	}
	held, err := mgr.ListGoals(ctx, source, mwanachamaagency.GoalFilter{})
	if err != nil || len(held) != goals {
		t.Fatalf("precondition: source draft should hold %d goals, holds %d (%v)", goals, len(held), err)
	}

	accepted := 0
	for n := 0; n < forks; n++ {
		body := fmt.Sprintf(`{"description":"f%d","forked_from_id":%q,"forked_from_type":"draft"}`, n, source)
		req := httptest.NewRequest(http.MethodPost, "/drafts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		serve(mgr, w, req)
		if w.Code >= 400 && w.Code < 500 {
			return
		}
		if w.Code != http.StatusCreated {
			t.Fatalf("POST /drafts answered %d %s", w.Code, w.Body.String())
		}
		accepted++
		var fork struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &fork); err != nil || fork.ID == "" {
			t.Fatalf("decode fork: %v %s", err, w.Body.String())
		}
		archive := httptest.NewRequest(http.MethodPost, "/drafts/"+fork.ID+"/archive", nil)
		aw := httptest.NewRecorder()
		serve(mgr, aw, archive)
		if aw.Code != http.StatusOK && aw.Code != http.StatusNoContent {
			t.Fatalf("POST /drafts/%s/archive answered %d %s", fork.ID, aw.Code, aw.Body.String())
		}
	}

	drafts, err := mgr.ListDrafts(ctx, mwanachamaagency.DraftFilter{})
	if err != nil {
		t.Fatalf("ListDrafts: %v", err)
	}
	t.Errorf("AG54 (P2) open hole: %d consecutive ~100-byte POST /drafts forks of a %d-goal draft were all accepted, "+
		"each copying every row, leaving %d drafts and %d goal rows written by the forks alone; "+
		"want a fork refused with a 4xx once an agency holds more drafts, or more forked rows, than a cap "+
		"(this test assumes a cap below %d drafts)", accepted, goals, len(drafts), accepted*goals, forks)
}
