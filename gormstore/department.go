package gormstore

import (
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DepartmentRow is the GORM row for a [models.Department].
//
// ParentID is a nullable *string (NULL for a top-level department), not
// the domain Department's plain "" — DepartmentToRow/DepartmentFromRow map
// "" <-> nil at the boundary. It is indexed but deliberately not a GORM
// association/foreign key, matching mwanachama-backend-actor's
// GroupRow.ParentID precedent: Migrate scopes tables per instance via
// db.Table(...), so a GORM-managed self-referencing FK would resolve
// against the wrong table.
type DepartmentRow struct {
	ID        string  `gorm:"primaryKey"`
	Code      string  `gorm:"uniqueIndex"`
	ParentID  *string `gorm:"index"`
	LevelID   string  `gorm:"index"`
	Name      string
	CreatedAt string
	UpdatedAt string
	Deleted   bool
}

func (r *DepartmentRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// DepartmentToRow converts a domain Department to its row shape, stamping
// UpdatedAt.
func DepartmentToRow(d models.Department) DepartmentRow {
	return DepartmentRow{
		ID:        d.ID,
		Code:      d.Code,
		ParentID:  StringToNullable(d.ParentID),
		LevelID:   d.LevelID,
		Name:      d.Name,
		CreatedAt: d.CreatedAt,
		UpdatedAt: models.NowRFC3339(),
		Deleted:   d.Deleted,
	}
}

// DepartmentFromRow converts a row back to the domain Department.
func DepartmentFromRow(r DepartmentRow) models.Department {
	return models.Department{
		ID:          r.ID,
		Code:        r.Code,
		ParentID:    nullableToString(r.ParentID),
		LevelID:     r.LevelID,
		Name:        r.Name,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}
