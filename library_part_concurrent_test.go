package mwanachamaagency_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

const (
	simultaneousCalls  = 8
	simultaneousRounds = 12
)

func newFileLibraryManager(t *testing.T) libraryManager {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "agency.db") + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(simultaneousCalls)
	t.Cleanup(func() { sqlDB.Close() })
	s, err := mwanachamaagency.SpecFor("libraryrace")
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
	return mgr.(libraryManager)
}

func atTheSameMoment(calls int, call func(i int) error) []error {
	results := make([]error, calls)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i] = call(i)
		}()
	}
	close(start)
	wg.Wait()
	return results
}

func requireOneWinner(t *testing.T, results []error) {
	t.Helper()
	won := 0
	for _, err := range results {
		switch {
		case err == nil:
			won++
		case errors.Is(err, mwanachamaagency.ErrLibraryTitleTaken), errors.Is(err, mwanachamaagency.ErrConflict):
		default:
			t.Errorf("a loser answered %v, want ErrLibraryTitleTaken or ErrConflict", err)
		}
	}
	if won != 1 {
		t.Errorf("%d calls succeeded, want exactly 1", won)
	}
}

func TestLibraryPartTitles_SimultaneousCreatesOfOneTitleStoreOnePart(t *testing.T) {
	for _, k := range libraryPartKinds() {
		t.Run(k.name, func(t *testing.T) {
			mgr := newFileLibraryManager(t)
			ctx := context.Background()
			_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
			before, err := k.count(mgr, ctx, o.DraftID)
			if err != nil {
				t.Fatalf("count: %v", err)
			}
			for round := 1; round <= simultaneousRounds; round++ {
				title := fmt.Sprintf("Probe %d", round)
				results := atTheSameMoment(simultaneousCalls, func(int) error {
					_, _, err := k.create(mgr, ctx, o.DraftID, title)
					return err
				})
				requireOneWinner(t, results)
				held, err := k.count(mgr, ctx, o.DraftID)
				if err != nil {
					t.Fatalf("count: %v", err)
				}
				if held != before+round {
					t.Fatalf("round %d: %d parts held, want %d", round, held, before+round)
				}
			}
		})
	}
}

func TestLibraryPartTitles_SimultaneousRenamesToOneTitleLeaveOnePartWithIt(t *testing.T) {
	for _, k := range libraryPartKinds() {
		t.Run(k.name, func(t *testing.T) {
			mgr := newFileLibraryManager(t)
			ctx := context.Background()
			_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
			for round := 1; round <= simultaneousRounds; round++ {
				ids := make([]string, simultaneousCalls)
				for i := range ids {
					id, _, err := k.create(mgr, ctx, o.DraftID, fmt.Sprintf("Round %d part %d", round, i))
					if err != nil {
						t.Fatalf("create: %v", err)
					}
					ids[i] = id
				}
				target := fmt.Sprintf("Shared %d", round)
				results := atTheSameMoment(simultaneousCalls, func(i int) error {
					return k.update(mgr, ctx, ids[i], target)
				})
				requireOneWinner(t, results)
			}
		})
	}
}

func TestLibraryPartTitles_ACreateAndARenameToOneTitleStoreItOnce(t *testing.T) {
	for _, k := range libraryPartKinds() {
		t.Run(k.name, func(t *testing.T) {
			mgr := newFileLibraryManager(t)
			ctx := context.Background()
			_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
			for round := 1; round <= simultaneousRounds; round++ {
				id, _, err := k.create(mgr, ctx, o.DraftID, fmt.Sprintf("Waiting %d", round))
				if err != nil {
					t.Fatalf("create: %v", err)
				}
				target := fmt.Sprintf("Wanted %d", round)
				results := atTheSameMoment(2, func(i int) error {
					if i == 0 {
						_, _, err := k.create(mgr, ctx, o.DraftID, target)
						return err
					}
					return k.update(mgr, ctx, id, target)
				})
				requireOneWinner(t, results)
			}
		})
	}
}

func TestSharedHazard_EightCallersOnADraftWithNoHazardAllGetTheOneCode(t *testing.T) {
	for round := 1; round <= simultaneousRounds; round++ {
		mgr := newFileLibraryManager(t)
		ctx := context.Background()
		_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
		codes := make([]string, simultaneousCalls)
		atTheSameMoment(simultaneousCalls, func(i int) error {
			codes[i] = sharedHazard(mgr, ctx, o.DraftID)
			return nil
		})
		for i, code := range codes {
			if code == "" || code != codes[0] {
				t.Fatalf("round %d: caller %d got %q, caller 0 got %q, want every caller to get the one hazard code", round, i, code, codes[0])
			}
		}
	}
}
