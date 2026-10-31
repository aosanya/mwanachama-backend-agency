package mwanachamaagency_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCatalogueNamesChatPostAndDocumentSearch(t *testing.T) {
	want := map[string]string{"chat.post": "comm.chat_message.post", "document.search": "sources.chunk.search"}
	for code, action := range want {
		p, ok := models.LookupPermission(code)
		if !ok {
			t.Fatalf("permission %s missing from the catalogue", code)
		}
		found := false
		for _, a := range p.Actions {
			found = found || a == action
		}
		if !found || p.Locked {
			t.Fatalf("permission %s = %+v, want unlocked and carrying %s", code, p, action)
		}
	}
}
