package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestEventFlowStepReadsNeverReachAnotherDraftsSteps(t *testing.T) {
	mgr, owner := newTestManagerWithDraft(t)
	other, err := mgr.CreateDraft(t.Context(), "other", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	step := models.EventFlowStep{DraftID: owner, Code: "w:1", WorkflowCode: "w", Step: "1", Name: "owner-only", HandlerCode: "h1", TriggerTopic: "t", Consumer: "c"}
	if _, err := mgr.CreateEventFlowStep(t.Context(), step); err != nil {
		t.Fatalf("CreateEventFlowStep: %v", err)
	}

	get := func(path string) (int, string) {
		t.Helper()
		w := httptest.NewRecorder()
		serve(mgr, w, httptest.NewRequest(http.MethodGet, path, nil))
		return w.Code, strings.TrimSpace(w.Body.String())
	}

	lookups := []string{"handler_code=h1", "code=w:1", "trigger_topic=t&consumer=c"}
	for _, l := range lookups {
		if code, body := get("/event-flow-steps/lookup?" + l + "&draft_id=" + owner); code != http.StatusOK || !strings.Contains(body, "owner-only") {
			t.Fatalf("precondition: lookup %s in the owning draft -> %d %s, want 200 with the step", l, code, body)
		}
	}
	if code, body := get("/event-flow-steps?draft_id=" + owner); code != http.StatusOK || !strings.Contains(body, "owner-only") {
		t.Fatalf("precondition: list in the owning draft -> %d %s, want 200 with the step", code, body)
	}

	scopes := map[string]string{"another draft": "&draft_id=" + other.ID, "an unknown draft": "&draft_id=no-such-draft", "no draft": ""}
	for name, scope := range scopes {
		if code, body := get("/event-flow-steps?workflow_code=w" + scope); code != http.StatusOK || body != "[]" {
			t.Errorf("list naming %s -> %d %s, want 200 [] and no step from another draft", name, code, body)
		}
		for _, l := range lookups {
			if code, body := get("/event-flow-steps/lookup?" + l + scope); code != http.StatusNotFound || strings.Contains(body, "owner-only") {
				t.Errorf("lookup %s naming %s -> %d %s, want 404 with nothing from another draft", l, name, code, body)
			}
		}
	}
}
