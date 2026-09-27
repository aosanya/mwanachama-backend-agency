package models

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

type Module struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DraftID     string `json:"draft_id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Locked      bool   `json:"locked"`
	Spec        string `json:"spec,omitempty"`
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
