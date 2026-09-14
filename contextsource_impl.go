// contextsource_impl.go — ContextSource lifecycle for [agencyManager].
// Ported from CodeValdAgency's raci.go (AddContextSource/
// ListContextSources/RemoveContextSource).
package mwanachamaagency

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-agency/gormstore"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

// AddContextSource creates a ContextSource attached to workPlanID. Returns
// [ErrWorkPlanNotFound] if no matching WorkPlan exists.
func (m *agencyManager) AddContextSource(ctx context.Context, workPlanID string, cs models.ContextSource) (models.ContextSource, error) {
	if _, err := m.GetWorkPlan(ctx, workPlanID); err != nil {
		return models.ContextSource{}, err
	}
	cs.WorkPlanID = workPlanID
	if cs.CreatedAt == "" {
		cs.CreatedAt = models.NowRFC3339()
	}
	row := gormstore.ContextSourceToRow(cs)
	if err := m.db.WithContext(ctx).Table(m.tables.ContextSources).Create(&row).Error; err != nil {
		return models.ContextSource{}, fmt.Errorf("AddContextSource: %w", err)
	}
	return gormstore.ContextSourceFromRow(row), nil
}

// ListContextSources returns every ContextSource attached to workPlanID,
// narrowed by filter. Returns [ErrWorkPlanNotFound] if no matching
// WorkPlan exists.
func (m *agencyManager) ListContextSources(ctx context.Context, workPlanID string, filter ContextSourceFilter) ([]models.ContextSource, error) {
	if _, err := m.GetWorkPlan(ctx, workPlanID); err != nil {
		return nil, err
	}
	q := m.db.WithContext(ctx).Table(m.tables.ContextSources).Where("work_plan_id = ?", workPlanID).Order("id")
	q = eqFilter(q, "source_type", filter.SourceType)
	var rows []gormstore.ContextSourceRow
	err := q.Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ListContextSources: %w", err)
	}
	out := make([]models.ContextSource, len(rows))
	for i, r := range rows {
		out[i] = gormstore.ContextSourceFromRow(r)
	}
	return out, nil
}

// RemoveContextSource deletes the ContextSource identified by sourceID.
// Returns [ErrContextSourceNotFound] if no such ContextSource exists, or if
// it belongs to a different WorkPlan than workPlanID — a check the source
// method accepted workPlanID for but never actually performed.
func (m *agencyManager) RemoveContextSource(ctx context.Context, workPlanID, sourceID string) error {
	var row gormstore.ContextSourceRow
	err := m.db.WithContext(ctx).Table(m.tables.ContextSources).Where("id = ?", sourceID).First(&row).Error
	if err != nil {
		return fmt.Errorf("RemoveContextSource %s: %w", sourceID, ErrContextSourceNotFound)
	}
	if row.WorkPlanID != workPlanID {
		return fmt.Errorf("RemoveContextSource %s: %w", sourceID, ErrContextSourceNotFound)
	}
	if err := m.db.WithContext(ctx).Table(m.tables.ContextSources).Where("id = ?", sourceID).
		Delete(&gormstore.ContextSourceRow{}).Error; err != nil {
		return fmt.Errorf("RemoveContextSource %s: %w", sourceID, err)
	}
	return nil
}
