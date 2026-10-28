package mwanachamaagency_test

import (
	"context"
	"strings"
	"testing"
)

func TestMarketModuleAndPowersAreCataloguedWithoutDecidingActions(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	powers, err := mgr.ListPowers(ctx)
	if err != nil {
		t.Fatalf("ListPowers: %v", err)
	}
	forbidden := []string{"answer", "approve", "decline", "mark_paid", "withdraw", "remove", "cancel"}
	found := 0
	for _, p := range powers {
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
			for _, f := range forbidden {
				if strings.Contains(a, "."+f) || strings.HasSuffix(a, "_"+f) || strings.HasSuffix(a, "."+f) {
					t.Errorf("%s grants %s, which an agent must not hold", p.Code, a)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("no market powers in the catalog")
	}
}
