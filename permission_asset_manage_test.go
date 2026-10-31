package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCatalogueNamesAssetManage(t *testing.T) {
	p, ok := models.LookupPermission("asset.manage")
	if !ok {
		t.Fatal("permission asset.manage missing from the catalogue")
	}
	if p.Locked {
		t.Fatalf("permission asset.manage = %+v, want unlocked", p)
	}
	have := map[string]bool{}
	for _, a := range p.Actions {
		have[a] = true
	}
	for _, a := range []string{"assetmanager.asset.create", "assetmanager.movement.post"} {
		if !have[a] {
			t.Fatalf("permission asset.manage = %+v, want it carrying %s", p, a)
		}
	}
}
