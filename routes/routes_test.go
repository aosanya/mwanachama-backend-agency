package routes_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func TestRoute_PatternWithPrefix(t *testing.T) {
	r := routes.Route{Method: "GET", Path: "/goals/{goalID}"}
	got := r.Pattern("/v1/agency")
	want := "GET /v1/agency/goals/{goalID}"
	if got != want {
		t.Fatalf("Pattern = %q, want %q", got, want)
	}
}

func TestRoutes_DefaultNouns(t *testing.T) {
	mgr := newTestManager(t)
	rts := routes.Routes(mgr, routes.ResourceNames{})

	want := map[string]bool{
		"POST /agency":                   true,
		"GET /agency":                    true,
		"POST /goals":                    true,
		"GET /goals":                     true,
		"POST /work-items":               true,
		"GET /drafts":                    true,
		"POST /drafts/{draftID}/promote": true,
		"DELETE /roles/{roleID}":         true,
		"GET /objectives/{objectiveID}/role-assignments": true,
		"POST /objectives":         true,
		"DELETE /values/{valueID}": true,
		"DELETE /rules/{ruleID}":   true,
	}
	got := map[string]bool{}
	for _, rt := range rts {
		got[rt.Pattern("")] = true
	}
	for pattern := range want {
		if !got[pattern] {
			t.Errorf("expected route %q in Routes() output, not found", pattern)
		}
	}
}

// A noun override has to reach the nested paths built from it, not just
// that noun's own top-level collection — Issue's Draft-scoped route is
// addressed entirely by names.Draft.
func TestRoutes_NounOverride(t *testing.T) {
	mgr := newTestManager(t)
	rts := routes.Routes(mgr, routes.ResourceNames{Draft: "revisions"})

	found := false
	for _, rt := range rts {
		if rt.Pattern("") == "GET /revisions/{draftID}/issues" {
			found = true
		}
		if rt.Pattern("") == "GET /drafts/{draftID}/issues" {
			t.Fatal("expected the overridden Draft noun to also apply to the nested Issue path")
		}
	}
	if !found {
		t.Fatal("expected the overridden Draft noun in the nested Issue route")
	}
}
