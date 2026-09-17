package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// TestListGoals_PinsLikeFilterWildcardInjection pins AG32: draft_impl.go's
// likeFilter builds its LIKE pattern as "%"+value+"%" without escaping any
// '%'/'_' already present in the caller-supplied value, so a filter value
// of a single "_" or "%" matches every row instead of only the one row
// that genuinely contains that character. Every List* method built on
// likeFilter (AG18 — 17 methods across Goal/Objective/Role/Workflow/
// WorkItem/Value/Rule/...) shares this gap.
//
// Asserts the CURRENT broken behavior (both filters match all 4 seeded
// goals). Once fixed (escape '%'/'_'/the escape character itself before
// building the pattern, e.g. via a REPLACE chain or ESCAPE clause), this
// must be updated to assert exactly 1 match per filter — the goal whose
// title actually contains that character.
func TestListGoals_PinsLikeFilterWildcardInjection(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	titles := []string{"Alpha Beta", "Gamma Delta", "Contains_Underscore", "Has%Percent"}
	for _, title := range titles {
		if _, err := mgr.CreateGoal(ctx, models.Goal{Title: title, Ordinality: 1}); err != nil {
			t.Fatalf("CreateGoal(%q): %v", title, err)
		}
	}

	cases := []struct {
		filter string
		want   string // the one title that should match once escaping lands
	}{
		{filter: "_", want: "Contains_Underscore"},
		{filter: "%", want: "Has%Percent"},
	}

	for _, tc := range cases {
		got, err := mgr.ListGoals(ctx, "", mwanachamaagency.GoalFilter{Title: tc.filter})
		if err != nil {
			t.Fatalf("ListGoals(title=%q): %v", tc.filter, err)
		}
		// Pinning the CURRENT bug: the un-escaped wildcard matches every
		// seeded row, not just tc.want. len(titles) == 4 today.
		if len(got) != len(titles) {
			t.Fatalf("AG32 regression? filter=%q matched %d goals, want %d (all rows, the current broken behavior) — got %v",
				tc.filter, len(got), len(titles), got)
		}
	}
}
