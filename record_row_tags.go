package mwanachamaagency

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const AllRowsTag = "all"

func cleanTags(tags []string) ([]string, error) {
	out := []string{}
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			return nil, invalidRow("a tag is not blank")
		}
		if !slices.Contains(out, tag) {
			out = append(out, tag)
		}
	}
	return out, nil
}

func reachesEveryRow(tags []string) bool {
	return len(tags) == 0 || slices.Contains(tags, AllRowsTag)
}

func tagFilter(q *gorm.DB, tags []string) *gorm.DB {
	if reachesEveryRow(tags) {
		return q
	}
	clauses := make([]string, 0, len(tags))
	needles := make([]any, 0, len(tags))
	for _, tag := range tags {
		quoted, _ := json.Marshal(tag)
		clauses = append(clauses, "tags LIKE ? ESCAPE '\\'")
		needles = append(needles, "%"+likeEscaper.Replace(string(quoted))+"%")
	}
	return q.Where("("+strings.Join(clauses, " OR ")+")", needles...)
}

func carriesAny(rowTags, tags []string) bool {
	for _, tag := range rowTags {
		if slices.Contains(tags, tag) {
			return true
		}
	}
	return false
}

func (m *agencyManager) AddRecordRowTag(ctx context.Context, rowID, tag string) (models.RecordRow, error) {
	if strings.TrimSpace(tag) == "" {
		return models.RecordRow{}, fmt.Errorf("AddRecordRowTag: %w", invalidRow("a tag is not blank"))
	}
	var out models.RecordRow
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.RecordRow
		q := tx.WithContext(ctx).Table(m.table(roleRecordRow)).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", rowID)
		if err := m.take(q, roleRecordRow, &row, ErrRecordNotFound); err != nil {
			return invalidRow("the row to tag does not exist")
		}
		if row.Deleted {
			return invalidRow("the row is retired")
		}
		if row.ReturnedOn != "" {
			return invalidRow("the row was returned, so it is history and not a current row")
		}
		replacing, err := list[models.RecordRow](m, tx.WithContext(ctx).Table(m.table(roleRecordRow)).Where("replaces = ?", row.ID), roleRecordRow)
		if err != nil {
			return err
		}
		if len(replacing) > 0 {
			return invalidRow("the row was replaced by a correction, so it is history and not a current row")
		}
		out = row
		if out.Tags == nil {
			out.Tags = []string{}
		}
		if slices.Contains(out.Tags, tag) {
			return nil
		}
		out.Tags = append(slices.Clone(out.Tags), tag)
		raw, err := json.Marshal(out.Tags)
		if err != nil {
			return err
		}
		out.LastUpdated = rowClock().Format(models.TimeLayout)
		return tx.WithContext(ctx).Table(m.table(roleRecordRow)).Where("id = ?", row.ID).
			Updates(map[string]any{"tags": string(raw), "last_updated": out.LastUpdated}).Error
	})
	if err != nil {
		return models.RecordRow{}, fmt.Errorf("AddRecordRowTag: %w", err)
	}
	return out, nil
}
