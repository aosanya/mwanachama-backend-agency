package mcp_test

import "testing"

func TestReviewTools_ListRunningAnswersItemsAcrossDraftsAndDropsSettledOnes(t *testing.T) {
	d := newReviewDesk(t)

	empty := d.ok("agency_list_running_reviews", map[string]any{})
	if items, ok := empty["items"].([]any); !ok || len(items) != 0 {
		t.Fatalf("running list before any start = %+v, want {items: []}", empty)
	}

	running := reviewOf(d.start())["id"].(string)
	out := d.ok("agency_list_running_reviews", map[string]any{})
	items, _ := out["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != running || items[0].(map[string]any)["status"] != "running" {
		t.Fatalf("running list = %+v, want the one running review", out)
	}
	if _, has := items[0].(map[string]any)["areas"]; has {
		t.Fatalf("a listed review carries its parts: %+v", items[0])
	}

	d.finishAll(running)
	settled := d.ok("agency_list_running_reviews", map[string]any{})
	if items, _ := settled["items"].([]any); len(items) != 0 {
		t.Fatalf("running list after the review settled = %+v, want none", settled)
	}
}
