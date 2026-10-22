package models

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

type Power struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Locked bool   `json:"locked"`
}

//go:embed powers_catalog.json
var powersCatalogJSON []byte

var powersCatalog = mustLoadPowers(powersCatalogJSON)

func mustLoadPowers(raw []byte) []Power {
	var out []Power
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(fmt.Sprintf("models: powers_catalog.json: %v", err))
	}
	seen := make(map[string]bool, len(out))
	for _, p := range out {
		if p.Code == "" || seen[p.Code] {
			panic(fmt.Sprintf("models: powers_catalog.json: empty or duplicate code %q", p.Code))
		}
		seen[p.Code] = true
	}
	return out
}

func PowerCatalog() []Power {
	return append([]Power(nil), powersCatalog...)
}

func LookupPower(code string) (Power, bool) {
	for _, p := range powersCatalog {
		if p.Code == code {
			return p, true
		}
	}
	return Power{}, false
}
