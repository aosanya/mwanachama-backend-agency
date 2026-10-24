package models

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

type Power struct {
	Code    string   `json:"code"`
	Name    string   `json:"name"`
	Locked  bool     `json:"locked"`
	Module  string   `json:"module,omitempty"`
	Actions []string `json:"actions"`
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
	for i, p := range out {
		if p.Code == "" || seen[p.Code] {
			panic(fmt.Sprintf("models: powers_catalog.json: empty or duplicate code %q", p.Code))
		}
		seen[p.Code] = true
		if out[i].Actions == nil {
			out[i].Actions = []string{}
		}
	}
	return out
}

func PowerCatalog() []Power {
	return append([]Power(nil), powersCatalog...)
}

func DistinctPowers(codes []string) []string {
	if codes == nil {
		return nil
	}
	seen := make(map[string]bool, len(codes))
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

func LookupPower(code string) (Power, bool) {
	for _, p := range powersCatalog {
		if p.Code == code {
			return p, true
		}
	}
	return Power{}, false
}
