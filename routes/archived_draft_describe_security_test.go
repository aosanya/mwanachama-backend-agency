package routes_test

import (
	"net/http"
	"testing"
)

func TestArchivedDraftDescriptionCannotBeEditedOrReopened(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	to := func(w http.ResponseWriter, r *http.Request) { serve(mgr, w, r) }

	if code, out := draftCall(t, to, http.MethodPatch, "/drafts/"+draftID, `{"description":"before archive"}`); code != http.StatusOK {
		t.Fatalf("precondition: PATCH /drafts/%s on an open draft -> %d %v", draftID, code, out)
	}
	if code, out := draftCall(t, to, http.MethodPost, "/drafts/"+draftID+"/archive", ""); code != http.StatusOK || out["status"] != "archived" {
		t.Fatalf("precondition: archiving draft %s -> %d %v", draftID, code, out)
	}

	for _, body := range []string{
		`{"description":"after archive"}`,
		`{"description":"reopen","status":"open"}`,
		`{}`,
	} {
		if code, out := draftCall(t, to, http.MethodPatch, "/drafts/"+draftID, body); code != http.StatusConflict {
			t.Errorf("PATCH /drafts/%s %s on an archived draft -> %d %v; an archived draft must refuse every edit with 409", draftID, body, code, out)
		}
	}

	_, read := draftCall(t, to, http.MethodGet, "/drafts/"+draftID, "")
	if read["status"] != "archived" || read["description"] != "before archive" {
		t.Errorf("archived draft %s read back %v after refused edits; want status archived and the description it was archived with", draftID, read)
	}
}
