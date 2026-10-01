package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAG56_OpenBug_RoleHintTreatsUnderscoreAndPercentAsWildcards(t *testing.T) {
	mgr, draft := newTestManagerWithDraft(t)
	call := func(method, path, body string) (int, map[string]any) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		serve(mgr, w, r)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}

	for _, name := range []string{"Engineer", "Manager", "Director"} {
		code, out := call(http.MethodPost, "/roles", fmt.Sprintf(`{"name":%q,"draft_id":%q}`, name, draft))
		if code != http.StatusCreated {
			t.Fatalf("seed role %s: %d %v", name, code, out)
		}
	}

	code, out := call(http.MethodGet, "/roles/_", "")
	if code != http.StatusNotFound {
		t.Fatalf("GET /roles/_: status = %d, want 404", code)
	}
	msg, _ := out["error"].(string)
	if strings.Contains(msg, "did you mean") {
		t.Fatalf("AG56 open bug: GET /roles/_ answered %q — a single '_' (which names no real role, code or name) matched all three seeded roles' code, name and id via the \"did you mean\" hint because '_' is spliced unescaped into a SQL LIKE pattern; want the plain not-found message with no hint, since none of Engineer/Manager/Director/R-1/R-2/R-3 contains a literal underscore", msg)
	}
}
