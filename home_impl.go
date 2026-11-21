package mwanachamaagency

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func invalidHomeEntry(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidHomeEntry, fmt.Sprintf(format, args...))
}

func (m *agencyManager) CreateHomeEntry(ctx context.Context, in models.HomeEntry) (models.HomeEntry, error) {
	var out models.HomeEntry
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.requireOpenDraft(ctx, tx, in.DraftID); err != nil {
			return err
		}
		in.Key, in.Title = strings.TrimSpace(in.Key), strings.TrimSpace(in.Title)
		if err := models.ValidateHomeEntry(in); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidHomeEntry, err)
		}
		held, err := list[models.HomeEntry](m, scopeQuery(tx.WithContext(ctx).Table(m.table(roleHomeEntry)), in.DraftID).Order("ordinality"), roleHomeEntry)
		if err != nil {
			return err
		}
		place, err := homePlace(held, in)
		if err != nil {
			return err
		}
		if err := m.requireHomeRoles(ctx, tx, in.DraftID, []models.DocumentHomeNode{{Key: in.Key, Roles: in.Roles}}); err != nil {
			return err
		}
		now := models.NowRFC3339()
		row := in
		row.ID, row.Ordinality, row.CreatedAt, row.LastUpdated, row.Deleted = newID(), place, now, now, false
		if err := m.insertRow(ctx, tx, roleHomeEntry, row); err != nil {
			return duplicateOr(err, invalidHomeEntry("the key %q or the row page it declares is already used in this draft", in.Key))
		}
		out = row
		return nil
	})
	if err != nil {
		return models.HomeEntry{}, fmt.Errorf("CreateHomeEntry: %w", err)
	}
	return out, nil
}

func homePlace(held []models.HomeEntry, in models.HomeEntry) (int, error) {
	if len(held) >= models.HomeMaxEntries {
		return 0, invalidHomeEntry("a home holds at most %d entries", models.HomeMaxEntries)
	}
	byID := make(map[string]models.HomeEntry, len(held))
	siblings := 0
	for _, row := range held {
		byID[row.ID] = row
		if row.ParentID == in.ParentID {
			siblings++
		}
	}
	if siblings >= models.HomeMaxPerSection {
		return 0, invalidHomeEntry("a home section holds at most %d entries", models.HomeMaxPerSection)
	}
	level, inPage := 1, false
	for parentID := in.ParentID; parentID != ""; level++ {
		parent, found := byID[parentID]
		if !found {
			return 0, ErrHomeEntryNotFound
		}
		if level == 1 && !parent.IsSection() {
			return 0, invalidHomeEntry("%q is an entry that opens something, so it holds nothing", parent.Key)
		}
		inPage = inPage || parent.PageOf != ""
		parentID = parent.ParentID
	}
	if level > models.HomeMaxDepth {
		return 0, invalidHomeEntry("a home goes at most %d levels deep", models.HomeMaxDepth)
	}
	if models.UsesHomeRow(in) && !inPage {
		return 0, invalidHomeEntry("the home entry %q uses the tapped row outside a row page", in.Key)
	}
	return siblings, nil
}

func (m *agencyManager) requireHomeRoles(ctx context.Context, tx *gorm.DB, draftID string, nodes []models.DocumentHomeNode) error {
	if len(models.HomeRoleCodes(nodes)) == 0 {
		return nil
	}
	roles, err := list[models.Role](m, scopeQuery(tx.WithContext(ctx).Table(m.table(roleRole)), draftID), roleRole)
	if err != nil {
		return err
	}
	parts := models.HomeParts{}
	for _, role := range roles {
		parts.Roles = append(parts.Roles, models.HomePart{Code: role.Code, Name: role.Name})
	}
	return homeRolesResolve(parts, nodes)
}

func homeRolesResolve(parts models.HomeParts, nodes []models.DocumentHomeNode) error {
	if err := parts.CheckHomeRoles(nodes); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidHomeEntry, err)
	}
	return nil
}

func (m *agencyManager) SetHome(ctx context.Context, draftID string, entries []map[string]any) ([]models.HomeEntry, error) {
	var out []models.HomeEntry
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := m.requireOpenDraft(ctx, tx, draftID); err != nil {
			return err
		}
		nodes, err := homeNodesOf(entries)
		if err != nil {
			return err
		}
		if err := models.ValidateHomeDocument(nodes); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidHomeEntry, err)
		}
		if err := m.requireHomeRoles(ctx, tx, draftID, nodes); err != nil {
			return err
		}
		if err := m.deleteWhere(ctx, tx, roleHomeEntry, "draft_id = ?", draftID); err != nil {
			return err
		}
		if err := m.applyHomeNodes(ctx, tx, draftID, "", nodes, models.NowRFC3339()); err != nil {
			return err
		}
		out, err = list[models.HomeEntry](m, scopeQuery(tx.WithContext(ctx).Table(m.table(roleHomeEntry)), draftID).Order("parent_id, ordinality"), roleHomeEntry)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("SetHome: %w", err)
	}
	return out, nil
}

func homeNodesOf(entries []map[string]any) ([]models.DocumentHomeNode, error) {
	raw, err := json.Marshal(entries)
	if err != nil {
		return nil, invalidHomeEntry("%v", err)
	}
	var nodes []models.DocumentHomeNode
	if err := json.Unmarshal(raw, &nodes); err != nil {
		return nil, invalidHomeEntry("%v", err)
	}
	return nodes, nil
}

func (m *agencyManager) ListHomeEntries(ctx context.Context, draftID string) ([]models.HomeEntry, error) {
	q := scopeQuery(m.q(ctx, roleHomeEntry).Order("parent_id, ordinality"), draftID)
	out, err := list[models.HomeEntry](m, q, roleHomeEntry)
	if err != nil {
		return nil, fmt.Errorf("ListHomeEntries: %w", err)
	}
	return out, nil
}

func (m *agencyManager) applyHomeNodes(ctx context.Context, tx *gorm.DB, draftID, parentID string, nodes []models.DocumentHomeNode, now string) error {
	for place, node := range nodes {
		row := models.HomeEntryOfNode(node)
		row.ID, row.DraftID, row.ParentID, row.Ordinality, row.CreatedAt, row.LastUpdated = newID(), draftID, parentID, place, now, now
		if err := m.copyRow(ctx, tx, roleHomeEntry, row); err != nil {
			return err
		}
		if err := m.applyHomeNodes(ctx, tx, draftID, row.ID, node.Entries, now); err != nil {
			return err
		}
	}
	return nil
}

func (m *agencyManager) homeDocument(ctx context.Context, tx *gorm.DB, draftID string) ([]models.DocumentHomeNode, error) {
	rows, err := list[models.HomeEntry](m, scopeQuery(tx.WithContext(ctx).Table(m.table(roleHomeEntry)), draftID).Order("ordinality"), roleHomeEntry)
	if err != nil {
		return nil, err
	}
	return models.NestHomeEntries(rows), nil
}

func (m *agencyManager) forkHomeEntries(ctx context.Context, tx *gorm.DB, sourceScope, newDraftID, now string) error {
	rows, err := list[models.HomeEntry](m, scopeQuery(tx.Table(m.table(roleHomeEntry)), sourceScope).Order("ordinality"), roleHomeEntry)
	if err != nil {
		return fmt.Errorf("CreateDraft: %w", err)
	}
	ids := make(map[string]string, len(rows))
	for _, row := range rows {
		ids[row.ID] = newID()
	}
	for _, row := range rows {
		row.ID, row.DraftID, row.CreatedAt, row.LastUpdated = ids[row.ID], newDraftID, now, now
		row.ParentID = remap(ids, row.ParentID)
		if err := m.copyRow(ctx, tx, roleHomeEntry, row); err != nil {
			return err
		}
	}
	return nil
}
