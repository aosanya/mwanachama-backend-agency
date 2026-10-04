package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestForkFromPublicationMatchesTheTagExactly(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)
	to := func(w http.ResponseWriter, r *http.Request) { serve(mgr, w, r) }

	if code, pub := draftCall(t, to, http.MethodPost, "/publications", `{"draft_id":"`+draftID+`","tag":"release-one"}`); code != http.StatusCreated {
		t.Fatalf("precondition: POST /publications -> %d %v, want 201", code, pub)
	}
	if code, out := draftCall(t, to, http.MethodPost, "/drafts", `{"description":"fork","forked_from_type":"publication","forked_from_id":"release-one"}`); code != http.StatusCreated {
		t.Fatalf("precondition: forking the exact tag -> %d %v, want 201", code, out)
	}

	for _, tag := range []string{"%", "release%", "release_one", "RELEASE-ONE", "release", " release-one", "x' OR '1'='1", "release-one' --"} {
		quoted, _ := json.Marshal(tag)
		body := `{"description":"fork","forked_from_type":"publication","forked_from_id":` + string(quoted) + `}`
		if code, out := draftCall(t, to, http.MethodPost, "/drafts", body); code != http.StatusNotFound {
			t.Errorf("POST /drafts forking tag %q -> %d %v, want 404 publication not found: a tag that is not exactly a publication's own must not materialize one", tag, code, out)
		}
	}

	for _, kind := range []string{"PUBLICATION", "Publication", "publications"} {
		body := `{"description":"fork","forked_from_type":"` + kind + `","forked_from_id":"release-one"}`
		if code, out := draftCall(t, to, http.MethodPost, "/drafts", body); code != http.StatusBadRequest {
			t.Errorf("POST /drafts with forked_from_type %q -> %d %v, want 400", kind, code, out)
		}
	}
}
