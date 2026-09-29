package mwanachamaagency

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func Provision(db *gorm.DB, s *spec.Spec) error {
	ready, err := upToDate(db, s)
	if err != nil {
		return err
	}
	if ready {
		return nil
	}
	if err := renameLegacy(db, s); err != nil {
		return err
	}
	if err := spec.Migrate(db, s); err != nil {
		return err
	}
	return seed(db, s)
}

func upToDate(db *gorm.DB, s *spec.Spec) (bool, error) {
	declared := make([]string, 0, len(s.Objects))
	legacy := make([]string, 0, len(s.Objects))
	for _, o := range s.Objects {
		declared = append(declared, s.TableFor(o))
		for _, name := range legacyTables(s, o) {
			if name != s.TableFor(o) {
				legacy = append(legacy, name)
			}
		}
	}

	held, err := tablesNamed(db, append(append([]string{}, declared...), legacy...))
	if err != nil {
		return false, err
	}
	for _, name := range declared {
		if !held[name] {
			return false, nil
		}
	}
	for _, name := range legacy {
		if held[name] {
			return false, nil
		}
	}

	var openDrafts int64
	if err := db.Table(tableFor(s, roleDraft)).
		Where("status = ?", string(models.DraftStatusOpen)).Count(&openDrafts).Error; err != nil {
		return false, fmt.Errorf("agency: read the working draft: %w", err)
	}
	if openDrafts == 0 {
		return false, nil
	}

	var codes int64
	if err := db.Table(tableFor(s, roleModule)).
		Distinct("code").Count(&codes).Error; err != nil {
		return false, fmt.Errorf("agency: read the module catalog: %w", err)
	}
	return int(codes) >= len(models.ModuleCatalog), nil
}

func tablesNamed(db *gorm.DB, names []string) (map[string]bool, error) {
	query := `select table_name from information_schema.tables
		where table_schema = current_schema() and table_name in ?`
	if db.Dialector.Name() != "postgres" {
		query = `select name from sqlite_master where type = 'table' and name in ?`
	}
	var found []string
	if err := db.Raw(query, names).Scan(&found).Error; err != nil {
		return nil, fmt.Errorf("agency: read the table set: %w", err)
	}
	held := make(map[string]bool, len(found))
	for _, name := range found {
		held[name] = true
	}
	return held, nil
}

func seed(db *gorm.DB, s *spec.Spec) error {
	draftID, err := ensureWorkingDraft(db, s)
	if err != nil {
		return err
	}
	if err := adoptUnscopedRows(db, s, draftID); err != nil {
		return err
	}
	if err := backfillCodes(db, s); err != nil {
		return err
	}
	if err := backfillModules(db, s, draftID); err != nil {
		return err
	}
	if err := raiseSequencesPastExistingCodes(db, s); err != nil {
		return err
	}
	return dropRetired(db, s)
}

func tableFor(s *spec.Spec, role string) string {
	o, ok := s.ByRole(role)
	if !ok {
		return ""
	}
	return s.TableFor(o)
}

func ensureWorkingDraft(db *gorm.DB, s *spec.Spec) (string, error) {
	drafts := tableFor(s, roleDraft)

	var open []map[string]any
	if err := db.Table(drafts).Where("status = ?", string(models.DraftStatusOpen)).
		Order("created_at").Limit(1).Find(&open).Error; err != nil {
		return "", fmt.Errorf("agency: read the working draft: %w", err)
	}
	if len(open) == 1 {
		id, _ := open[0]["id"].(string)
		return id, nil
	}

	now := models.NowRFC3339()
	row, err := specstore.Encode(mustObject(s, roleDraft), models.Draft{
		ID:          specstore.NewID(),
		Description: "Initial draft",
		Status:      models.DraftStatusOpen,
		CreatedAt:   now,
		LastUpdated: now,
	})
	if err != nil {
		return "", fmt.Errorf("agency: seed the working draft: %w", err)
	}
	if err := db.Table(drafts).Create(row).Error; err != nil {
		return "", fmt.Errorf("agency: seed the working draft: %w", err)
	}
	return row["id"].(string), nil
}

func adoptUnscopedRows(db *gorm.DB, s *spec.Spec, draftID string) error {
	for _, role := range draftScopedRoles {
		table := tableFor(s, role)
		if table == "" {
			continue
		}
		if err := db.Table(table).Where("draft_id IS NULL OR draft_id = ?", "").
			Updates(map[string]any{"draft_id": draftID}).Error; err != nil {
			return fmt.Errorf("agency: adopt unscoped rows in %s: %w", table, err)
		}
	}
	return nil
}

func backfillCodes(db *gorm.DB, s *spec.Spec) error {
	sequences := tableFor(s, roleCodeSequence)
	for _, c := range codedRoles {
		table := tableFor(s, c.role)
		err := db.Transaction(func(tx *gorm.DB) error {
			var ids []string
			if err := tx.Table(table).Where("code = '' OR code IS NULL").
				Order("created_at").Pluck("id", &ids).Error; err != nil {
				return err
			}
			for _, id := range ids {
				code, err := nextCode(context.Background(), tx, sequences, c.entityType, c.prefix)
				if err != nil {
					return err
				}
				if err := tx.Table(table).Where("id = ?", id).Update("code", code).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("agency: backfill codes in %s: %w", table, err)
		}
	}
	return nil
}

// raiseSequencesPastExistingCodes moves each counter past every code the
// tables already hold. A counter behind its own table mints a code that
// exists — which is what an import does, since an imported row keeps the
// code it was exported with, and what a database restored from elsewhere
// would do too.
func raiseSequencesPastExistingCodes(db *gorm.DB, s *spec.Spec) error {
	sequences := tableFor(s, roleCodeSequence)
	for _, c := range codedRoles {
		var codes []string
		if err := db.Table(tableFor(s, c.role)).Pluck("code", &codes).Error; err != nil {
			return fmt.Errorf("agency: read codes in %s: %w", tableFor(s, c.role), err)
		}
		highest := 0
		for _, code := range codes {
			if n := numberIn(code, c.prefix); n > highest {
				highest = n
			}
		}
		if highest == 0 {
			continue
		}
		if err := raiseSequence(context.Background(), db, sequences, c.entityType, highest+1); err != nil {
			return err
		}
	}
	return nil
}

func backfillModules(db *gorm.DB, s *spec.Spec, draftID string) error {
	table := tableFor(s, roleModule)
	object := mustObject(s, roleModule)

	return db.Transaction(func(tx *gorm.DB) error {
		for _, entry := range models.ModuleCatalog {
			var count int64
			if err := tx.Table(table).Where("code = ? AND draft_id = ?", entry.Code, draftID).
				Count(&count).Error; err != nil {
				return fmt.Errorf("agency: seed the module catalog: %w", err)
			}
			if count > 0 {
				continue
			}
			now := models.NowRFC3339()
			row, err := specstore.Encode(object, models.Module{
				ID:          specstore.NewID(),
				Code:        entry.Code,
				DraftID:     draftID,
				Name:        entry.Name,
				Description: entry.Description,
				Enabled:     entry.Enabled,
				Locked:      entry.Locked,
				CreatedAt:   now,
				LastUpdated: now,
			})
			if err != nil {
				return fmt.Errorf("agency: seed the module catalog: %w", err)
			}
			if err := tx.Table(table).Create(row).Error; err != nil {
				return fmt.Errorf("agency: seed the module catalog: %w", err)
			}
		}
		return nil
	})
}

func dropRetired(db *gorm.DB, s *spec.Spec) error {
	for _, suffix := range retiredTableSuffixes {
		retired := []string{
			s.Instance + "_" + s.Module + "_" + s.MountName() + suffix,
			s.Instance + "_" + s.Module + suffix,
			s.Instance + suffix,
		}
		for _, name := range retired {
			if err := db.Migrator().DropTable(name); err != nil {
				return fmt.Errorf("agency: drop retired table: %w", err)
			}
		}
	}

	publications := tableFor(s, rolePublication)
	has, err := hasColumn(db, publications, retiredPublicationColumn)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	stmt := fmt.Sprintf("alter table %s drop column %s", publications, retiredPublicationColumn)
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("agency: drop %s.%s: %w", publications, retiredPublicationColumn, err)
	}
	return nil
}

func mustObject(s *spec.Spec, role string) spec.Object {
	o, ok := s.ByRole(role)
	if !ok {
		panic("agency: the spec fills no object for the role " + role)
	}
	return o
}

var draftScopedRoles = []string{
	roleGoal, roleObjective, roleWorkflow, roleWorkItem, roleEventFlowStep,
	roleRole, roleRoleAssignment, roleModule, roleObjectiveModule,
	roleValue, roleRule, roleIssue,
}

var codedRoles = []struct {
	role       string
	entityType string
	prefix     string
}{
	{roleGoal, "goal", "G"},
	{roleObjective, "objective", "O"},
	{roleRole, "role", "R"},
	{roleWorkflow, "workflow", "W"},
	{roleWorkItem, "workitem", "WI"},
	{roleValue, "value", "V"},
	{roleRule, "rule", "RU"},
}

var retiredTableSuffixes = []string{"_work_plans", "_context_sources"}

const retiredPublicationColumn = "status"

func objectTable(o spec.Object) string {
	if o.Table != "" {
		return o.Table
	}
	return o.Name
}

func legacyTables(s *spec.Spec, o spec.Object) []string {
	if s.MountName() != spec.DefaultMount {
		return nil
	}
	return []string{s.Instance + "_" + objectTable(o)}
}

const legacyTimestampColumn = "updated_at"

const timestampColumn = "last_updated"

func renameLegacy(db *gorm.DB, s *spec.Spec) error {
	m := db.Migrator()
	for _, o := range s.Objects {
		declared := s.TableFor(o)

		for _, legacy := range legacyTables(s, o) {
			if legacy == declared || !m.HasTable(legacy) {
				continue
			}
			switch {
			case !m.HasTable(declared):
				if err := db.Exec(fmt.Sprintf("alter table %s rename to %s", legacy, declared)).Error; err != nil {
					return fmt.Errorf("agency: rename %s to %s: %w", legacy, declared, err)
				}
				if err := dropLegacyIndexes(db, declared); err != nil {
					return err
				}
			default:
				stranded, err := rowCount(db, legacy)
				if err != nil {
					return err
				}
				if stranded > 0 {
					return fmt.Errorf("agency: %s and %s both exist and %s still holds %d row(s), which no read would ever reach again",
						legacy, declared, legacy, stranded)
				}
			}
		}

		if !m.HasTable(declared) || !declaresColumn(o, timestampColumn) {
			continue
		}
		legacyColumn, err := hasColumn(db, declared, legacyTimestampColumn)
		if err != nil {
			return err
		}
		declaredColumn, err := hasColumn(db, declared, timestampColumn)
		if err != nil {
			return err
		}
		if !legacyColumn || declaredColumn {
			continue
		}
		stmt := fmt.Sprintf("alter table %s rename column %s to %s",
			declared, legacyTimestampColumn, timestampColumn)
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("agency: rename %s.%s: %w", declared, legacyTimestampColumn, err)
		}
	}
	return nil
}

func dropLegacyIndexes(db *gorm.DB, table string) error {
	var names []string
	switch db.Dialector.Name() {
	case "postgres":
		if err := db.Raw(`select indexname from pg_indexes where tablename = ?`, table).
			Scan(&names).Error; err != nil {
			return fmt.Errorf("agency: read indexes of %s: %w", table, err)
		}
	default:
		if err := db.Raw(`select name from sqlite_master where type = 'index' and tbl_name = ?`, table).
			Scan(&names).Error; err != nil {
			return fmt.Errorf("agency: read indexes of %s: %w", table, err)
		}
	}

	for _, name := range names {
		if !strings.HasPrefix(name, "idx_") {
			continue
		}
		if err := db.Exec("drop index if exists " + name).Error; err != nil {
			return fmt.Errorf("agency: drop index %s: %w", name, err)
		}
	}
	return nil
}

func rowCount(db *gorm.DB, table string) (int64, error) {
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("agency: count %s: %w", table, err)
	}
	return n, nil
}

func hasColumn(db *gorm.DB, table, column string) (bool, error) {
	var found []int
	q := `select 1 from pragma_table_info(?) where name = ?`
	if db.Dialector.Name() == "postgres" {
		q = `select 1 from information_schema.columns where table_name = ? and column_name = ?`
	}
	if err := db.Raw(q, table, column).Scan(&found).Error; err != nil {
		return false, fmt.Errorf("agency: read columns of %s: %w", table, err)
	}
	return len(found) > 0, nil
}

func declaresColumn(o spec.Object, column string) bool {
	for _, f := range o.Fields {
		if f.Name == column {
			return true
		}
	}
	return false
}
