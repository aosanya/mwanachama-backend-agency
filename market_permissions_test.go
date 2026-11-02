package mwanachamaagency_test

import (
	"context"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestMarketModuleAndPermissionsAreCataloguedWithoutDecidingActions(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	permissions, err := mgr.ListPermissions(ctx)
	if err != nil {
		t.Fatalf("ListPermissions: %v", err)
	}
	forbidden := []string{"answer", "approve", "decline", "mark_paid", "withdraw", "remove", "cancel"}
	posterReplies := map[string]string{"market.ask_answer": "market.ask.answer"}
	found := 0
	for _, p := range permissions {
		if !strings.HasPrefix(p.Code, "market.") {
			continue
		}
		found++
		if p.Module != "market" {
			t.Errorf("%s module = %q, want market", p.Code, p.Module)
		}
		if len(p.Actions) == 0 {
			t.Errorf("%s declares no actions", p.Code)
		}
		for _, a := range p.Actions {
			if !strings.HasPrefix(a, "market.") {
				t.Errorf("%s action %s is not a market action", p.Code, a)
			}
			if posterReplies[p.Code] == a {
				continue
			}
			for _, f := range forbidden {
				if strings.Contains(a, "."+f) || strings.HasSuffix(a, "_"+f) || strings.HasSuffix(a, "."+f) {
					t.Errorf("%s grants %s, which an agent must not hold", p.Code, a)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("no market permissions in the catalog")
	}
}

func TestOnlyTheAnswerAnAskPermissionCarriesTheAnswerToAnAsk(t *testing.T) {
	for _, p := range models.PermissionCatalog() {
		if p.Code != "market.ask_answer" && carries(p, "market.ask.answer") {
			t.Errorf("%s carries market.ask.answer, which only market.ask_answer may", p.Code)
		}
	}
	if held := heldByAnUnlockedPermission()["market.ask.answer"]; held != "market.ask_answer" {
		t.Errorf("market.ask.answer is held by %q, want market.ask_answer", held)
	}
}
