package mwanachamaagency_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCodesRunInSequence(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Sequence Co"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "sequence", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	for i := 1; i <= 5; i++ {
		g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: d.ID, Title: fmt.Sprintf("goal %d", i)})
		if err != nil {
			t.Fatalf("CreateGoal: %v", err)
		}
		if want := fmt.Sprintf("G-%d", i); g.Code != want {
			t.Errorf("code = %q, want %q", g.Code, want)
		}
	}
}

func TestAG35_ConcurrentCreatesNeverShareACodeAndRefusalsAreClassified(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(4)
	t.Cleanup(func() { sqlDB.Close() })

	s, err := mwanachamaagency.SpecFor("codeseqtest")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, s)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}

	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Race Co"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "race", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	const iterations = 30
	var mu sync.Mutex
	seen := map[string]bool{}
	refusals := 0

	for i := 0; i < iterations; i++ {
		var wg sync.WaitGroup
		wg.Add(2)
		for g := 0; g < 2; g++ {
			go func(n int) {
				defer wg.Done()
				goal, err := mgr.CreateGoal(ctx, models.Goal{DraftID: d.ID, Title: fmt.Sprintf("g%d", n)})
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					if errors.Is(err, mwanachamaagency.ErrConflict) {
						refusals++
						return
					}
					t.Errorf("CreateGoal: %v", err)
					return
				}
				if seen[goal.Code] {
					t.Errorf("code %q was minted twice", goal.Code)
				}
				seen[goal.Code] = true
			}(i)
		}
		wg.Wait()
	}

	t.Logf("%d codes minted, %d writes refused by the database's own locking", len(seen), refusals)
	if len(seen) == 0 {
		t.Fatal("no goal was created at all, so this proves nothing")
	}
}
