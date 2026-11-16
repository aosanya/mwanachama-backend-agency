package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const (
	composedName   = "Café"
	decomposedName = "Café"
)

func TestScenarioActorNames_AnInvisibleNameIsRefusedAndADecomposedTwinConflicts(t *testing.T) {
	w := newScenarioWorld(t)
	ctx := context.Background()
	base := models.Scenario{DraftID: w.draftID, Name: "n", WorkflowID: w.buyNow.ID}
	cases := []struct {
		name   string
		actors []models.ScenarioActor
		want   error
	}{
		{"an actor name that draws nothing", []models.ScenarioActor{w.actor("​", farmer)}, mwanachamaagency.ErrInvalidScenario},
		{"an actor named by the decomposed twin of another", []models.ScenarioActor{w.actor(composedName, farmer), w.actor(decomposedName, farmer)}, mwanachamaagency.ErrDuplicateScenarioActor},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := base
			s.Actors = c.actors
			if _, err := w.mgr.CreateScenario(ctx, s); !errors.Is(err, c.want) {
				t.Errorf("err = %v, want %v", err, c.want)
			}
		})
	}
}

func TestLibraryPartTitles_ADecomposedTwinOfATakenTitleConflicts(t *testing.T) {
	for _, k := range libraryPartKinds() {
		t.Run(k.name, func(t *testing.T) {
			mgr := newLibraryManager(t)
			ctx := context.Background()
			_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
			if _, _, err := k.create(mgr, ctx, o.DraftID, composedName); err != nil {
				t.Fatalf("first create: %v", err)
			}
			if _, _, err := k.create(mgr, ctx, o.DraftID, decomposedName); !errors.Is(err, mwanachamaagency.ErrLibraryTitleTaken) {
				t.Fatalf("decomposed twin err = %v, want ErrLibraryTitleTaken", err)
			}
		})
	}
}
