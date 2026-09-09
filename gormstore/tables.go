package gormstore

import "gorm.io/gorm"

// TableNames names the physical tables one mounted copy of this package
// reads and writes.
type TableNames struct {
	Agencies  string
	Drafts    string
	Goals     string
	Workflows string
	WorkItems string
}

// DefaultTableNames builds the conventional table set for one mounted
// instance of this package.
func DefaultTableNames(instance string) TableNames {
	return TableNames{
		Agencies:  instance + "_agencies",
		Drafts:    instance + "_drafts",
		Goals:     instance + "_goals",
		Workflows: instance + "_workflows",
		WorkItems: instance + "_work_items",
	}
}

// Migrate creates or updates the tables named by t.
func Migrate(db *gorm.DB, t TableNames) error {
	if err := db.Table(t.Agencies).AutoMigrate(&AgencyRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Drafts).AutoMigrate(&DraftRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Goals).AutoMigrate(&GoalRow{}); err != nil {
		return err
	}
	if err := db.Table(t.Workflows).AutoMigrate(&WorkflowRow{}); err != nil {
		return err
	}
	if err := db.Table(t.WorkItems).AutoMigrate(&WorkItemRow{}); err != nil {
		return err
	}
	return nil
}
