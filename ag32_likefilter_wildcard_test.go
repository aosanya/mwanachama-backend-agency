package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// TestListGoals_LikeFilterTreatsWildcardsLiterally pins the AG32 fix: '%',
// '_' and '\' in a filter value match themselves, not every row.
func TestListGoals_LikeFilterTreatsWildcardsLiterally(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	draftID := newDraftAgency(t, mgr, ctx)

	titles := []string{"Alpha Beta", "Gamma Delta", "Contains_Underscore", "Has%Percent", `Back\slash`}
	for _, title := range titles {
		if _, err := mgr.CreateGoal(ctx, models.Goal{DraftID: draftID, Title: title, Ordinality: 1}); err != nil {
			t.Fatalf("CreateGoal(%q): %v", title, err)
		}
	}

	cases := []struct{ filter, want string }{
		{"_", "Contains_Underscore"},
		{"%", "Has%Percent"},
		{`\`, `Back\slash`},
		{"beta", "Alpha Beta"},
	}
	for _, tc := range cases {
		got, err := mgr.ListGoals(ctx, draftID, mwanachamaagency.GoalFilter{Title: tc.filter})
		if err != nil {
			t.Fatalf("ListGoals(title=%q): %v", tc.filter, err)
		}
		if len(got) != 1 || got[0].Title != tc.want {
			t.Fatalf("filter=%q matched %v, want only %q", tc.filter, got, tc.want)
		}
	}
}
