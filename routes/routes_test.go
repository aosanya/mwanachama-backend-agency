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
		"GET /work-plans/{workPlanID}/context-sources":   true,
		"DELETE /roles/{roleID}":                         true,
		"GET /objectives/{objectiveID}/role-assignments": true,
		"POST /objectives":                               true,
		"DELETE /values/{valueID}":                       true,
		"DELETE /rules/{ruleID}":                         true,
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

func TestRoutes_NounOverride(t *testing.T) {
	mgr := newTestManager(t)
	rts := routes.Routes(mgr, routes.ResourceNames{WorkPlan: "dispatch-plans"})

	found := false
	for _, rt := range rts {
		if rt.Pattern("") == "GET /dispatch-plans/{workPlanID}/context-sources" {
			found = true
		}
		if rt.Pattern("") == "GET /work-plans/{workPlanID}/context-sources" {
			t.Fatal("expected the overridden WorkPlan noun to also apply to the nested ContextSource path")
		}
	}
	if !found {
		t.Fatal("expected the overridden WorkPlan noun in the nested ContextSource route")
	}
}
