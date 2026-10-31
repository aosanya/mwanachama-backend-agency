package mwanachamaagency

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func (m *agencyManager) StartReview(ctx context.Context, draftID, startedBy string) (models.ReviewView, error) {
	var out models.ReviewView
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.requireOpenDraft(ctx, tx, draftID); err != nil {
			return err
		}
		running, err := m.runningReviews(ctx, tx, draftID)
		if err != nil {
			return err
		}
		if running > 0 {
			return ErrReviewRunning
		}
		latest, err := m.latestReview(ctx, tx, draftID)
		if err != nil {
			return err
		}
		review := models.Review{
			ID: newID(), DraftID: draftID, Status: models.ReviewRunning,
			StartedBy: strings.TrimSpace(startedBy), StartedAt: models.NowRFC3339(),
		}
		if latest != nil {
			review.PreviousReviewID = latest.ID
		}
		if err := m.insertRow(ctx, tx, roleReview, review); err != nil {
			return err
		}
		for _, name := range m.areasToRead(ctx, draftID) {
			area := models.ReviewArea{ID: newID(), ReviewID: review.ID, Area: name, Status: models.ReviewAreaReading}
			if err := m.insertRow(ctx, tx, roleReviewArea, area); err != nil {
				return err
			}
		}
		out, err = m.reviewView(ctx, tx, review)
		return err
	})
	if errors.Is(err, ErrReviewRunning) {
		return models.ReviewView{}, ErrReviewRunning
	}
	if err != nil {
		return models.ReviewView{}, fmt.Errorf("StartReview: %w", err)
	}
	return out, nil
}

func (m *agencyManager) runningReviews(ctx context.Context, tx *gorm.DB, draftID string) (int, error) {
	q := tx.WithContext(ctx).Table(m.table(roleReview)).Where("draft_id = ? AND status = ?", draftID, string(models.ReviewRunning))
	rows, err := list[models.Review](m, q, roleReview)
	return len(rows), err
}
