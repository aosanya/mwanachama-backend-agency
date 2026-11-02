package mcp_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func readsNoPart(context.Context, string) []models.ReviewAreaName { return nil }

func TestReviewTools_AStartWhenNoPartIsReadIsRefusedAndLeavesNothingRunning(t *testing.T) {
	d := newReviewDesk(t, mwanachamaagency.WithReviewAreas(readsNoPart))

	_, text, failed := d.call("agency_start_review", map[string]any{"draft_id": d.draftID})

	if !failed || text != "There is no part of this draft to review." {
		t.Fatalf("start with no part to read = failed %v %q, want the no-part sentence", failed, text)
	}
	running := d.ok("agency_list_running_reviews", map[string]any{})
	if items, ok := running["items"].([]any); !ok || len(items) != 0 {
		t.Fatalf("running list after the refused start = %+v, want {items: []}", running)
	}
}
