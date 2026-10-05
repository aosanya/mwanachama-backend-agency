package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestDetachReachesOnlyItsOwnAttachmentTable(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	value, err := mgr.CreateValue(t.Context(), models.Value{DraftID: draftID, Text: "correctness over speed"})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	rule, err := mgr.CreateRule(t.Context(), models.Rule{DraftID: draftID, Text: "never touch production data"})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	role, err := mgr.CreateRole(t.Context(), models.Role{DraftID: draftID, Name: "reviewer"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	valueAtt, err := mgr.CreateValueAttachment(t.Context(), value.ID, models.TargetTypeRole, role.ID)
	if err != nil {
		t.Fatalf("CreateValueAttachment: %v", err)
	}
	ruleAtt, err := mgr.CreateRuleAttachment(t.Context(), rule.ID, models.TargetTypeRole, role.ID)
	if err != nil {
		t.Fatalf("CreateRuleAttachment: %v", err)
	}

	call := func(method, path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		serve(mgr, w, httptest.NewRequest(method, path, nil))
		return w
	}
	attachmentIDs := func(path string) []string {
		t.Helper()
		w := call(http.MethodGet, path)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s = %d %s", path, w.Code, w.Body.String())
		}
		var rows []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		ids := make([]string, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		return ids
	}

	if got := attachmentIDs("/rules/" + rule.ID + "/attachments"); len(got) != 1 || got[0] != ruleAtt.ID {
		t.Fatalf("precondition: rule attachments = %v, want [%s]", got, ruleAtt.ID)
	}
	if got := attachmentIDs("/values/" + value.ID + "/attachments"); len(got) != 1 || got[0] != valueAtt.ID {
		t.Fatalf("precondition: value attachments = %v, want [%s]", got, valueAtt.ID)
	}

	if w := call(http.MethodDelete, "/values/"+value.ID+"/attachments/"+ruleAtt.ID); w.Code != http.StatusNotFound {
		t.Errorf("detaching a rule attachment through the value door = %d %s, want 404", w.Code, w.Body.String())
	}
	if w := call(http.MethodDelete, "/rules/"+rule.ID+"/attachments/"+valueAtt.ID); w.Code != http.StatusNotFound {
		t.Errorf("detaching a value attachment through the rule door = %d %s, want 404", w.Code, w.Body.String())
	}

	if got := attachmentIDs("/rules/" + rule.ID + "/attachments"); len(got) != 1 || got[0] != ruleAtt.ID {
		t.Errorf("rule attachments after a cross-door detach = %v, want [%s] untouched", got, ruleAtt.ID)
	}
	if got := attachmentIDs("/values/" + value.ID + "/attachments"); len(got) != 1 || got[0] != valueAtt.ID {
		t.Errorf("value attachments after a cross-door detach = %v, want [%s] untouched", got, valueAtt.ID)
	}
}
