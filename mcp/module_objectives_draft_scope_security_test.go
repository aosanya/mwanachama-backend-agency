package mcp_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestModuleObjectivesToolNeverAnswersOutsideItsDraft(t *testing.T) {
	mgr := newTestManager(t)
	ctx := ctxT(t)
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "scope"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draftA, err := mgr.CreateDraft(ctx, "a", "", "")
	if err != nil {
		t.Fatalf("CreateDraft a: %v", err)
	}
	draftB, err := mgr.CreateDraft(ctx, "b", "", "")
	if err != nil {
		t.Fatalf("CreateDraft b: %v", err)
	}
	moduleA, err := mgr.CreateModule(ctx, models.Module{Code: "git", Name: "Git", DraftID: draftA.ID, Enabled: true})
	if err != nil {
		t.Fatalf("CreateModule a: %v", err)
	}
	moduleB, err := mgr.CreateModule(ctx, models.Module{Code: "git", Name: "Git", DraftID: draftB.ID, Enabled: true})
	if err != nil {
		t.Fatalf("CreateModule b: %v", err)
	}
	goalA, err := mgr.CreateGoal(ctx, models.Goal{Title: "goal-only-in-draft-a", DraftID: draftA.ID})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	objectiveA, err := mgr.CreateObjective(ctx, models.Objective{Title: "objective-only-in-draft-a", GoalID: goalA.ID, DraftID: draftA.ID})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	if _, err := mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: objectiveA.ID, ModuleID: moduleA.ID, DraftID: draftA.ID, Rationale: "rationale-only-in-draft-a"}); err != nil {
		t.Fatalf("CreateObjectiveModule a: %v", err)
	}
	links, err := mgr.ListObjectiveModules(ctx, draftA.ID, mwanachamaagency.ObjectiveModuleFilter{ModuleID: moduleA.ID})
	if err != nil || len(links) != 1 {
		t.Fatalf("precondition: draft a's module should hold exactly one link, got %d (%v)", len(links), err)
	}

	_, err = mgr.CreateObjectiveModule(ctx, models.ObjectiveModule{ObjectiveID: objectiveA.ID, ModuleID: moduleB.ID, DraftID: draftB.ID, Rationale: "borrowed"})
	if !errors.Is(err, mwanachamaagency.ErrObjectiveNotFound) {
		t.Fatalf("a link in draft b naming draft a's objective must be refused as objective not found, got %v", err)
	}

	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	for _, draftID := range []string{draftB.ID, "", "no-such-draft"} {
		for _, moduleID := range []string{moduleA.ID, moduleB.ID} {
			res, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{
				Name:      "agency_list_module_objectives",
				Arguments: map[string]any{"instance_id": "any", "draft_id": draftID, "module_id": moduleID},
			})
			if err != nil {
				t.Fatalf("draft %q module %q: CallTool: %v", draftID, moduleID, err)
			}
			var text strings.Builder
			for _, c := range res.Content {
				if tc, ok := c.(*gosdkmcp.TextContent); ok {
					text.WriteString(tc.Text)
				}
			}
			for _, leaked := range []string{"only-in-draft-a", objectiveA.ID, goalA.ID} {
				if strings.Contains(text.String(), leaked) {
					t.Errorf("draft %q module %q: the listing carried %q from draft a: %q", draftID, moduleID, leaked, text.String())
				}
			}
			if !res.IsError && !strings.Contains(text.String(), "This module backs no objectives.") {
				t.Errorf("draft %q module %q: want an empty listing, got %q", draftID, moduleID, text.String())
			}
		}
	}
}
