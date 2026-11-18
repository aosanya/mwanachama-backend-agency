package models_test

import (
	"slices"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestOnlyResidentReadCarriesTheActionsThatOpenAResidentsNameAndAddress(t *testing.T) {
	for _, action := range []string{"actor.actor.list", "actor.actor.read", "actor.actor.read_many"} {
		var holders []string
		for _, p := range models.PermissionCatalog() {
			if slices.Contains(p.Actions, action) {
				holders = append(holders, p.Code)
			}
		}
		if !slices.Equal(holders, []string{"resident.read"}) {
			t.Errorf("permissions carrying %s = %v, want only resident.read", action, holders)
		}
	}
}
