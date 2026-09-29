package routes_test

import (
	"net/http"
	"testing"
)

func TestPublicationDoorsIgnoreForgedVersionHashDocumentAndTag(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)
	to := func(w http.ResponseWriter, r *http.Request) { serve(mgr, w, r) }

	code, pub := draftCall(t, to, http.MethodPost, "/publications",
		`{"draft_id":"`+draftID+`","tag":"v1","channel":"production","version":99,"content_hash":"forged","document":{"agency":{"name":"forged"}},"created_at":"1999-01-01T00:00:00Z","id":"chosen"}`)
	if code != http.StatusCreated {
		t.Fatalf("precondition: POST /publications -> %d %v, want 201", code, pub)
	}
	if pub["version"] != float64(1) || pub["content_hash"] == "forged" || pub["id"] == "chosen" || pub["created_at"] == "1999-01-01T00:00:00Z" {
		t.Errorf("POST /publications kept a caller-set server-owned field: %v", pub)
	}
	if doc, _ := pub["document"].(map[string]any); doc == nil || doc["agency"].(map[string]any)["name"] != "SWELab" {
		t.Errorf("POST /publications stored a caller-supplied document instead of the draft's own: %v", pub["document"])
	}

	if code, out := draftCall(t, to, http.MethodPatch, "/publications/1", `{"channel":"staging","tag":"hijack","content_hash":"forged","document":{}}`); code != http.StatusOK {
		t.Fatalf("precondition: PATCH /publications/1 -> %d %v, want 200", code, out)
	}
	_, read := draftCall(t, to, http.MethodGet, "/publications/1", "")
	if read["channel"] != "staging" || read["tag"] != "v1" || read["content_hash"] != pub["content_hash"] {
		t.Errorf("PATCH /publications/{version} changed more than the channel: %v", read)
	}

	if code, out := draftCall(t, to, http.MethodPatch, "/publications/42", `{"channel":"staging"}`); code != http.StatusNotFound {
		t.Errorf("PATCH /publications/42 on a version that does not exist -> %d %v, want 404", code, out)
	}
	if code, out := draftCall(t, to, http.MethodPost, "/drafts", `{"description":"f","forked_from_id":"no-such-tag","forked_from_type":"publication"}`); code != http.StatusNotFound {
		t.Errorf("fork from a tag that does not exist -> %d %v, want 404", code, out)
	}
}
