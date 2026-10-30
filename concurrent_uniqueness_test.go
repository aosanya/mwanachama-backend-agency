package mwanachamaagency_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func newSharedCacheManager(t *testing.T, instance string) mwanachamaagency.AgencyManager {
	t.Helper()
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

	s, err := mwanachamaagency.SpecFor(instance)
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
	return mgr
}

func raceWriters(n int, write func() error) (created int, unclassified []error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := write()
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				created++
				return
			}
			unclassified = append(unclassified, err)
		}()
	}
	close(start)
	wg.Wait()
	return created, unclassified
}

func TestAG62_ConcurrentCreateModuleWithOneCodeStoresOneAndClassifiesTheRest(t *testing.T) {
	mgr := newSharedCacheManager(t, "racemodule")
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Race Co"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "race", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	for round := 0; round < 10; round++ {
		code := "racing-" + string(rune('a'+round))
		created, refused := raceWriters(6, func() error {
			_, err := mgr.CreateModule(ctx, models.Module{DraftID: d.ID, Code: code, Name: code})
			return err
		})
		if created > 1 {
			t.Fatalf("round %d: %d Modules stored with code %q, want at most one", round, created, code)
		}
		for _, err := range refused {
			if !errors.Is(err, mwanachamaagency.ErrDuplicateModule) && !errors.Is(err, mwanachamaagency.ErrConflict) {
				t.Errorf("round %d: the losing writer saw %v, want ErrDuplicateModule or ErrConflict", round, err)
			}
		}
		mods, err := mgr.ListModules(ctx, d.ID, mwanachamaagency.ModuleFilter{Code: code})
		if err != nil {
			t.Fatalf("ListModules: %v", err)
		}
		held := 0
		for _, m := range mods {
			if m.Code == code {
				held++
			}
		}
		if created == 1 && held != 1 {
			t.Fatalf("round %d: the draft holds %d Modules with code %q, want 1", round, held, code)
		}
		if held > 1 {
			t.Fatalf("round %d: the draft holds %d Modules with code %q, want at most 1", round, held, code)
		}
	}
}

func TestAG63_ConcurrentCreateValueAttachmentStoresOneAndClassifiesTheRest(t *testing.T) {
	mgr := newSharedCacheManager(t, "raceattach")
	ctx := context.Background()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "Race Co"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "race", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	v, err := mgr.CreateValue(ctx, models.Value{DraftID: d.ID, Text: "never touch production data"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "hunter"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	created, refused := raceWriters(8, func() error {
		_, err := mgr.CreateValueAttachment(ctx, v.ID, models.TargetTypeRole, role.ID)
		return err
	})
	for _, err := range refused {
		if !errors.Is(err, mwanachamaagency.ErrDuplicateAttachment) && !errors.Is(err, mwanachamaagency.ErrConflict) {
			t.Errorf("the losing writer saw %v, want ErrDuplicateAttachment or ErrConflict", err)
		}
	}
	held, err := mgr.ListValueAttachments(ctx, v.ID)
	if err != nil {
		t.Fatalf("ListValueAttachments: %v", err)
	}
	if created > 1 || len(held) != 1 {
		t.Errorf("%d writers succeeded and the value holds %d attachments to one target, want exactly 1", created, len(held))
	}
}
