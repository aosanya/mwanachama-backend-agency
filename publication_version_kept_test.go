package mwanachamaagency_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestGoingBackToAnEarlierVersionKeepsEveryLaterVersionReadableWithItsDocument(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)
	draftID := o.DraftID

	v1, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency v1: %v", err)
	}
	if _, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: "pay on delivery"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	v2, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency v2: %v", err)
	}
	if _, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: "group orders close on Friday"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	v3, err := mgr.PublishAgency(ctx, draftID, "", "")
	if err != nil {
		t.Fatalf("PublishAgency v3: %v", err)
	}

	before := map[int]models.AgencyPublication{}
	for _, v := range []models.AgencyPublication{v2, v3} {
		got, err := mgr.GetPublication(ctx, v.Version)
		if err != nil {
			t.Fatalf("GetPublication v%d: %v", v.Version, err)
		}
		before[v.Version] = got
	}

	if _, err := mgr.SetPublicationChannel(ctx, v1.Version, models.DefaultChannel); err != nil {
		t.Fatalf("SetPublicationChannel back to v1: %v", err)
	}

	for _, kept := range []models.AgencyPublication{v2, v3} {
		got, err := mgr.GetPublication(ctx, kept.Version)
		if err != nil {
			t.Fatalf("GetPublication v%d after going back: %v", kept.Version, err)
		}
		if got.Channel != models.ChannelArchived {
			t.Errorf("v%d channel = %q, want archived", kept.Version, got.Channel)
		}
		if !reflect.DeepEqual(got.Document, before[kept.Version].Document) || len(got.Document.Values) == 0 {
			t.Errorf("v%d document changed or emptied by going back", kept.Version)
		}
	}
	live, err := mgr.GetPublicationForChannel(ctx, models.DefaultChannel)
	if err != nil || live.Version != v1.Version {
		t.Fatalf("live = %+v, %v; want version %d", live, err, v1.Version)
	}
}
