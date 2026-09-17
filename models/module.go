package models

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// Module is a catalog entry for a capability domain (e.g. "git",
// "taskmanager") this Agency may enable. Enabled is a real gate on a
// mounting gateway's route registration, not a passive record: an
// unselected Module's routes are absent entirely, not just
// capability-denied (AGD-002 decision #1).
//
// Module is fully read-only from the outside — which codes exist, their
// Name/Description, and each one's Locked state come from [ModuleCatalog],
// not a create/update/delete API (AGD-002 reversed to this after the
// mutation surface shipped and was seen live).
//
// Enabled is therefore derived, not stored state anyone can set: a Module
// is enabled exactly when some Objective in its own scope links it through
// an ObjectiveModule row. Selecting a Module *is* linking it to the
// Objective that needs it. Locked is the one exception — a locked code
// keeps the catalog's Enabled and can never be disabled, however few
// Objectives reference it.
//
// Code is the module's stable slug, matching a sibling backend repo's own
// name. Same lax draft-scoping as Goal, not RoleAssignment's stricter
// rule: a live Module can be toggled directly (AGD-002 decision #5).
// DraftID is "" for a live Module, or the owning Draft's id for a
// draft-scoped copy.
type Module struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DraftID     string `json:"draft_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Locked      bool   `json:"locked"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}

// ModuleCatalogEntry is one row of the embedded module catalog — the
// single source of truth for which Module codes exist, and what a fresh
// instance backfills for each.
type ModuleCatalogEntry struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Locked      bool   `json:"locked"`
}

//go:embed modules_catalog.json
var modulesCatalogJSON []byte

// ModuleCatalog is every known Module code, parsed once from
// modules_catalog.json at package init. Reused by [gormstore.BackfillModules]
// and [SeedSWELab] so both stay in sync with this one file.
var ModuleCatalog = mustParseModuleCatalog(modulesCatalogJSON)

func mustParseModuleCatalog(raw []byte) []ModuleCatalogEntry {
	var catalog []ModuleCatalogEntry
	if err := json.Unmarshal(raw, &catalog); err != nil {
		panic(fmt.Sprintf("models: invalid modules_catalog.json: %v", err))
	}
	seen := make(map[string]bool, len(catalog))
	for _, entry := range catalog {
		if entry.Code == "" {
			panic("models: modules_catalog.json contains an entry with an empty code")
		}
		if seen[entry.Code] {
			panic(fmt.Sprintf("models: modules_catalog.json contains a duplicate code %q", entry.Code))
		}
		seen[entry.Code] = true
	}
	return catalog
}
