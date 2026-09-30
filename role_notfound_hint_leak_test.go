package mwanachamaagency_test

import (
	"context"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG52_OpenBug_RoleNotFoundHintLeaksEveryRoleOnAWildcard(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Probe"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "probe", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	names := []string{"CEO", "CFO", "Secret Auditor"}
	for _, n := range names {
		if _, err := mgr.CreateRole(ctx, models.Role{Name: n, DraftID: d.ID}); err != nil {
			t.Fatalf("CreateRole(%s): %v", n, err)
		}
	}

	_, err = mgr.GetRole(ctx, "%")
	if err == nil {
		t.Fatal("GetRole(\"%\") unexpectedly succeeded")
	}
	msg := err.Error()
	leaked := 0
	for _, n := range names {
		if strings.Contains(msg, n) {
			leaked++
		}
	}
	if leaked > 0 {
		t.Errorf("AG52 open bug: GetRole with the SQL LIKE wildcard \"%%\" as its id got a not-found error that names %d of the %d other roles' Code, Name and internal id (%q); a lookup miss must never turn the id a caller typed into a search pattern that discloses unrelated rows", leaked, len(names), msg)
	}
}
