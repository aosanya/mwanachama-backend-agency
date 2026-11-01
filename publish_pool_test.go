package mwanachamaagency_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestHS446_ConcurrentPublishesOnAPoolCappedAtThePublishersDoNotStarveTheGroupLister(t *testing.T) {
	const publishers = 4
	db, err := gorm.Open(sqlite.Open("file:hs446?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	var mu sync.Mutex
	arrived := 0
	allArrived := make(chan struct{})
	groups := func(ctx context.Context) ([]models.PickerActor, error) {
		mu.Lock()
		arrived++
		if arrived == publishers {
			close(allArrived)
		}
		mu.Unlock()
		select {
		case <-allArrived:
		case <-time.After(2 * time.Second):
		}
		waitCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		conn, err := sqlDB.Conn(waitCtx)
		if err != nil {
			return nil, fmt.Errorf("the group lister could not get a connection: %w", err)
		}
		return nil, conn.Close()
	}

	s, err := mwanachamaagency.SpecFor("hs446")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, s, mwanachamaagency.WithGroupLister(groups))
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	sqlDB.SetMaxOpenConns(publishers)

	var wg sync.WaitGroup
	var out sync.Mutex
	var published int
	var failures []error
	start := make(chan struct{})
	for i := 0; i < publishers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := mgr.PublishAgency(ctx, o.DraftID, fmt.Sprintf("race-%d", i), "")
			out.Lock()
			defer out.Unlock()
			switch {
			case err == nil:
				published++
			case errors.Is(err, mwanachamaagency.ErrNoChangesDetected), errors.Is(err, mwanachamaagency.ErrConflict):
			default:
				failures = append(failures, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	for _, err := range failures {
		t.Errorf("HS446: a publish failed with %v; want the groups read before the transaction holds a connection", err)
	}
	if published == 0 && len(failures) == 0 {
		t.Errorf("HS446: no publish succeeded")
	}
}
