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

func TestReviewTools_ListRunningNarrowsToTheDraftNamed(t *testing.T) {
	d := newReviewDesk(t)
	other, err := d.mgr.CreateDraft(ctxT(t), "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	first := reviewOf(d.start())["id"].(string)
	second := reviewOf(d.ok("agency_start_review", map[string]any{"draft_id": other.ID}))["id"].(string)

	idsOf := func(args map[string]any) []string {
		out := d.ok("agency_list_running_reviews", args)
		items, ok := out["items"].([]any)
		if !ok {
			t.Fatalf("running list for %v = %+v, want {items: [...]}", args, out)
		}
		ids := []string{}
		for _, item := range items {
			ids = append(ids, item.(map[string]any)["id"].(string))
		}
		return ids
	}

	if got := idsOf(map[string]any{"draft_id": d.draftID}); len(got) != 1 || got[0] != first {
		t.Fatalf("running list of the first draft = %v, want only %s", got, first)
	}
	if got := idsOf(map[string]any{"draft_id": other.ID}); len(got) != 1 || got[0] != second {
		t.Fatalf("running list of the other draft = %v, want only %s", got, second)
	}
	if got := idsOf(map[string]any{"draft_id": "nope"}); len(got) != 0 {
		t.Fatalf("running list of an unknown draft = %v, want none", got)
	}
	if got := idsOf(map[string]any{}); len(got) != 2 {
		t.Fatalf("running list with no draft named = %v, want both", got)
	}
}
