package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAG57_CreateIgnoresABodySetDeletedFlag(t *testing.T) {
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

	cases := []struct{ name, path, body string }{
		{"goal", "/goals", `{"title":"G","draft_id":"` + draft + `","deleted":true}`},
		{"role", "/roles", `{"name":"R","draft_id":"` + draft + `","deleted":true}`},
		{"workflow", "/workflows", `{"name":"W","draft_id":"` + draft + `","deleted":true}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, created := call(http.MethodPost, c.path, c.body)
			id, _ := created["id"].(string)
			if code != http.StatusCreated || id == "" {
				t.Fatalf("precondition: POST %s %s -> %d %v", c.path, c.body, code, created)
			}
			code, got := call(http.MethodGet, c.path+"/"+id, "")
			if code != http.StatusOK {
				t.Fatalf("precondition: GET %s/%s -> %d %v", c.path, id, code, got)
			}
			if got["deleted"] != false {
				t.Errorf("AG57 (P2): POST %s with a body \"deleted\":true stored deleted=%v, want false — a create mints its own lifecycle state the way it mints id, code and created_at", c.path, got["deleted"])
			}
		})
	}
}
