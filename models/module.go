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
	Settings    string `json:"settings,omitempty"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
	Deleted     bool   `json:"deleted"`
}

type ModuleCatalogEntry struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Locked      bool     `json:"locked"`
	Requires    []string `json:"requires,omitempty"`
}

//go:embed modules_catalog.json
var modulesCatalogJSON []byte

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
	for _, entry := range catalog {
		for _, required := range entry.Requires {
			if !seen[required] {
				panic(fmt.Sprintf("models: modules_catalog.json: %q requires unknown code %q", entry.Code, required))
			}
		}
	}
	return catalog
}

func ModuleRequires(code string) []string {
	for _, entry := range ModuleCatalog {
		if entry.Code == code {
			return entry.Requires
		}
	}
	return nil
}

func WithRequiredModules(enabled map[string]bool) map[string]bool {
	out := make(map[string]bool, len(enabled))
	pending := make([]string, 0, len(enabled))
	for code, on := range enabled {
		if on {
			pending = append(pending, code)
		}
	}
	for len(pending) > 0 {
		code := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if out[code] {
			continue
		}
		out[code] = true
		pending = append(pending, ModuleRequires(code)...)
	}
	return out
}
