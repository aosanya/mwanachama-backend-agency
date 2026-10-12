package routes_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestTwoInstancesOnOneDatabaseNeverReachEachOthersRowsOrAttachments(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "shared.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	instance := func(name string) (mwanachamaagency.AgencyManager, string) {
		t.Helper()
		s, err := mwanachamaagency.SpecFor(name)
		if err != nil {
			t.Fatalf("SpecFor(%q): %v", name, err)
		}
		if err := mwanachamaagency.Provision(db, s); err != nil {
			t.Fatalf("Provision(%q): %v", name, err)
		}
		mgr, err := mwanachamaagency.NewAgencyManager(db, s)
		if err != nil {
			t.Fatalf("NewAgencyManager(%q): %v", name, err)
		}
		if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: name}); err != nil {
			t.Fatalf("CreateAgency(%q): %v", name, err)
		}
		d, err := mgr.CreateDraft(t.Context(), "seed", "", "")
		if err != nil {
			t.Fatalf("CreateDraft(%q): %v", name, err)
		}
		return mgr, d.ID
	}
	alice, aliceDraft := instance("isoalice")
	bob, bobDraft := instance("isobob")
	ctx := t.Context()

	aliceRule, err := alice.CreateRule(ctx, models.Rule{DraftID: aliceDraft, Text: "alice's rule"})
	if err != nil {
		t.Fatalf("CreateRule(alice): %v", err)
	}
	bobRule, err := bob.CreateRule(ctx, models.Rule{DraftID: bobDraft, Text: "bob's rule"})
	if err != nil {
		t.Fatalf("CreateRule(bob): %v", err)
	}
	bobValue, err := bob.CreateValue(ctx, models.Value{DraftID: bobDraft, Text: "bob's value"})
	if err != nil {
		t.Fatalf("CreateValue(bob): %v", err)
	}
	bobRole, err := bob.CreateRole(ctx, models.Role{DraftID: bobDraft, Name: "bob's role"})
	if err != nil {
		t.Fatalf("CreateRole(bob): %v", err)
	}
	bobRuleAtt, err := bob.CreateRuleAttachment(ctx, bobRule.ID, models.TargetTypeRole, bobRole.ID)
	if err != nil {
		t.Fatalf("CreateRuleAttachment(bob): %v", err)
	}
	bobValueAtt, err := bob.CreateValueAttachment(ctx, bobValue.ID, models.TargetTypeRole, bobRole.ID)
	if err != nil {
		t.Fatalf("CreateValueAttachment(bob): %v", err)
	}

	call := func(mgr mwanachamaagency.AgencyManager, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		serve(mgr, w, r)
		return w
	}

	if w := call(bob, http.MethodGet, "/rules/"+bobRule.ID+"/attachments", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), bobRuleAtt.ID) {
		t.Fatalf("precondition: bob's own rule attachments = %d %s, want 200 naming %s", w.Code, w.Body.String(), bobRuleAtt.ID)
	}

	crossDoors := []struct{ method, path, body string }{
		{http.MethodDelete, "/rules/" + aliceRule.ID + "/attachments/" + bobRuleAtt.ID, ""},
		{http.MethodDelete, "/rules/" + bobRule.ID + "/attachments/" + bobRuleAtt.ID, ""},
		{http.MethodDelete, "/values/" + bobValue.ID + "/attachments/" + bobValueAtt.ID, ""},
		{http.MethodGet, "/rules/" + bobRule.ID, ""},
		{http.MethodGet, "/rules/" + bobRule.ID + "/attachments", ""},
		{http.MethodGet, "/values/" + bobValue.ID, ""},
		{http.MethodPatch, "/rules/" + bobRule.ID, `{"text":"rewritten by alice"}`},
		{http.MethodDelete, "/values/" + bobValue.ID, ""},
		{http.MethodPost, "/rules/" + bobRule.ID + "/attachments", `{"target_type":"role","target_id":"` + bobRole.ID + `"}`},
	}
	for _, d := range crossDoors {
		if w := call(alice, d.method, d.path, d.body); w.Code != http.StatusNotFound {
			t.Errorf("alice's instance %s %s reached bob's row: %d %s, want 404", d.method, d.path, w.Code, w.Body.String())
		}
	}

	if w := call(bob, http.MethodGet, "/rules/"+bobRule.ID, ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"bob's rule"`) {
		t.Errorf("bob's rule after alice's attempts = %d %s, want 200 with its text unchanged", w.Code, w.Body.String())
	}
	if w := call(bob, http.MethodGet, "/values/"+bobValue.ID, ""); w.Code != http.StatusOK {
		t.Errorf("bob's value after alice's attempts = %d %s, want 200 still there", w.Code, w.Body.String())
	}
	if w := call(bob, http.MethodGet, "/values/"+bobValue.ID+"/attachments", ""); !strings.Contains(w.Body.String(), bobValueAtt.ID) {
		t.Errorf("bob's value attachments after alice's attempts = %d %s, want %s still there", w.Code, w.Body.String(), bobValueAtt.ID)
	}
	if w := call(bob, http.MethodDelete, "/rules/"+bobRule.ID+"/attachments/"+bobRuleAtt.ID, ""); w.Code != http.StatusNoContent {
		t.Errorf("bob detaching his own rule attachment = %d %s, want 204 — it should have survived alice's attempts", w.Code, w.Body.String())
	}
}
