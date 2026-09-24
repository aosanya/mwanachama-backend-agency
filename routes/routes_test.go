package routes_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func TestRoute_PatternWithPrefix(t *testing.T) {
	r := routes.Route{Method: "GET", Path: "/goals/{goal_id}"}
	got := r.Pattern("/v1/agency")
	want := "GET /v1/agency/goals/{goal_id}"
	if got != want {
		t.Fatalf("Pattern = %q, want %q", got, want)
	}
}

func TestRoutes_AnswersTheDeclaredAddresses(t *testing.T) {
	mgr := newTestManager(t)
	rts := routes.Routes(mgr)

	want := map[string]bool{
		"POST /agency":                    true,
		"GET /agency":                     true,
		"POST /goals":                     true,
		"GET /goals":                      true,
		"POST /work-items":                true,
		"GET /drafts":                     true,
		"POST /drafts/{draft_id}/archive": true,
		"DELETE /roles/{role_id}":         true,
		"GET /objectives/{objective_id}/role-assignments": true,
		"POST /objectives":          true,
		"DELETE /values/{value_id}": true,
		"DELETE /rules/{rule_id}":   true,
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

// Every route carries the action a mount gates it on, and no two share
// one: an operation nobody can name is one nobody can protect.
func TestRoutes_EveryRouteCarriesItsOwnAction(t *testing.T) {
	seen := map[string]string{}
	for _, rt := range routes.Routes(newTestManager(t)) {
		if rt.Action == "" {
			t.Errorf("%s carries no action", rt.Pattern(""))
			continue
		}
		if first, taken := seen[rt.Action]; taken {
			t.Errorf("%s and %s both claim the action %q", first, rt.Pattern(""), rt.Action)
		}
		seen[rt.Action] = rt.Pattern("")
	}
}

// Shape is what a mount reads to decide which addresses to register when it
// has no manager to build handlers against — wakala-api mounts per instance
// and resolves the manager per request. It has to agree with the real table,
// in the same order, or that mount registers one route and serves another.
func TestShapeAgreesWithRoutes(t *testing.T) {
	built := routes.Routes(newTestManager(t))
	shape := routes.Shape()

	if len(built) != len(shape) {
		t.Fatalf("Shape has %d routes, Routes has %d", len(shape), len(built))
	}
	for i := range built {
		if built[i].Pattern("") != shape[i].Pattern("") || built[i].Action != shape[i].Action {
			t.Errorf("index %d: Shape has %s (%s), Routes has %s (%s)",
				i, shape[i].Pattern(""), shape[i].Action, built[i].Pattern(""), built[i].Action)
		}
		if shape[i].Handler != nil {
			t.Errorf("index %d: Shape carries a handler, which it has no manager to build", i)
		}
	}
}
