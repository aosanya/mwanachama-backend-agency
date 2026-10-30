package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG52_TwoPublicationsHoldOneVersionAndOneChannel(t *testing.T) {
	mgr, db, s := newTestManagerDB(t)
	ctx := context.Background()
	_, o, _ := seedPublishableAgency(t, mgr, ctx)

	first, err := mgr.PublishAgency(ctx, o.DraftID, "first", "production")
	if err != nil {
		t.Fatalf("precondition: PublishAgency: %v", err)
	}
	if first.Version != 1 || first.Channel != "production" {
		t.Fatalf("precondition: first publication is %+v, want version 1 on production", first)
	}

	insertErr := db.WithContext(ctx).Table(tableOf(t, s, "agency_publication")).Create(map[string]any{
		"id":           "concurrent-publish",
		"draft_id":     o.DraftID,
		"content_hash": "a-different-draft",
		"version":      first.Version,
		"tag":          "second",
		"channel":      "production",
		"document":     "{}",
		"published_at": models.NowRFC3339(),
		"last_updated": models.NowRFC3339(),
	}).Error

	onChannel, err := mgr.ListPublications(ctx, mwanachamaagency.PublicationFilter{Channel: "production"})
	if err != nil {
		t.Fatalf("ListPublications: %v", err)
	}
	if insertErr == nil || len(onChannel) != 1 {
		t.Errorf("AG52: a second publication row carrying version %d on channel production, written as a concurrent PublishAgency writes it after reading the table before the first committed, was accepted by the store (insert error %v) and production now holds %d publications; want the store to refuse it so one version and one channel name exactly one publication", first.Version, insertErr, len(onChannel))
	}
}
