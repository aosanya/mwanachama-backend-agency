package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func TestAG48_OpenBug_UpdateAndDeleteRefuseArchivedDraft(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)

	role, err := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "original-name"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	if _, err := mgr.ArchiveDraft(t.Context(), draftID); err != nil {
		t.Fatalf("ArchiveDraft: %v", err)
	}

	mux := http.NewServeMux()
	for _, rt := range routes.Routes(mgr) {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := srv.Client()

	patchReq, _ := http.NewRequest(http.MethodPatch, srv.URL+"/roles/"+role.ID, strings.NewReader(`{"name":"renamed-after-archive"}`))
	patchResp, err := client.Do(patchReq)
	if err != nil {
		t.Fatalf("PATCH: %v", err)
	}
	patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusConflict {
		t.Errorf("AG48: PATCH /roles/%s on a Role whose Draft %s is archived returned %d, want 409 (ErrDraftNotOpen) — a caller can rename a row inside an already-archived Draft", role.ID, draftID, patchResp.StatusCode)
	}

	afterPatch, err := mgr.GetRole(t.Context(), role.ID)
	if err != nil {
		t.Fatalf("GetRole after PATCH: %v", err)
	}
	if afterPatch.Name != "original-name" {
		t.Errorf("AG48: Role.Name changed to %q via a route call against an archived Draft; want it left as %q", afterPatch.Name, "original-name")
	}

	delReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/roles/"+role.ID, nil)
	delResp, err := client.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusConflict {
		t.Errorf("AG48: DELETE /roles/%s on a Role whose Draft %s is archived returned %d, want 409 (ErrDraftNotOpen) — a caller can delete a row inside an already-archived Draft", role.ID, draftID, delResp.StatusCode)
	}

	stillThere, err := mgr.GetRole(t.Context(), role.ID)
	if err != nil {
		t.Errorf("AG48: Role %s was deleted via a route call against an archived Draft; GetRole now fails: %v", role.ID, err)
	}
	if stillThere.ID != role.ID {
		t.Errorf("AG48: Role %s no longer resolves to itself after the archived-draft DELETE call", role.ID)
	}
}
