package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestEventFlowStepLookupAndListingMatchOnlyTheExactSpelling(t *testing.T) {
	mgr, draft := newTestManagerWithDraft(t)
	if _, err := mgr.CreateEventFlowStep(t.Context(), models.EventFlowStep{
		DraftID: draft, Code: "planning:1", WorkflowCode: "planning", Step: "1",
		Name: "exact-only", HandlerCode: "wp-planning-1", TriggerTopic: "topic.a", Consumer: "consumer.a",
	}); err != nil {
		t.Fatalf("precondition: CreateEventFlowStep: %v", err)
	}

	get := func(path string) (int, string) {
		t.Helper()
		w := httptest.NewRecorder()
		serve(mgr, w, httptest.NewRequest(http.MethodGet, path, nil))
		return w.Code, w.Body.String()
	}
	scope := "draft_id=" + draft + "&"

	for _, exact := range []string{"code=planning:1", "handler_code=wp-planning-1", "trigger_topic=topic.a&consumer=consumer.a"} {
		if code, body := get("/event-flow-steps/lookup?" + scope + exact); code != http.StatusOK || !strings.Contains(body, "exact-only") {
			t.Fatalf("precondition: lookup %s -> %d %s, want 200 with the step", exact, code, body)
		}
	}
	if code, body := get("/event-flow-steps?" + scope + "workflow_code=planning"); code != http.StatusOK || !strings.Contains(body, "exact-only") {
		t.Fatalf("precondition: listing workflow_code=planning -> %d %s, want 200 with the step", code, body)
	}

	widened := []string{
		"code=%25", "code=planning:_", "code=PLANNING:1", "code=planning:1%20",
		"handler_code=%25", "handler_code=wp-planning-_", "handler_code=WP-PLANNING-1",
		"trigger_topic=%25&consumer=%25", "trigger_topic=topic._&consumer=consumer._",
	}
	for _, l := range widened {
		if code, body := get("/event-flow-steps/lookup?" + scope + l); code == http.StatusOK || strings.Contains(body, "exact-only") {
			t.Errorf("lookup %s -> %d %s; a lookup must match the stored spelling exactly, never as a pattern or case-folded", l, code, body)
		}
	}
	for _, l := range []string{"workflow_code=%25", "workflow_code=plan_ing", "workflow_code=PLANNING"} {
		if code, body := get("/event-flow-steps?" + scope + l); code != http.StatusOK || strings.Contains(body, "exact-only") {
			t.Errorf("listing %s -> %d %s; want 200 with no step, the filter matching only the exact workflow code", l, code, body)
		}
	}
}
