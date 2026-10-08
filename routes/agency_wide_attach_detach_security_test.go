package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG74_OpenHole_AttachEditDetachRewritesAnAgencyWideValue(t *testing.T) {
	mgr, draftID := newTestManagerWithDraft(t)
	ctx := t.Context()
	const original = "we never sell member data"
	value, err := mgr.CreateValue(ctx, models.Value{DraftID: draftID, Text: original})
	if err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: draftID, Name: "anyone"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	send := func(method, path, body string) *httptest.ResponseRecorder {
		var req *http.Request
		if body == "" {
			req = httptest.NewRequest(method, path, nil)
		} else {
			req = httptest.NewRequest(method, path, strings.NewReader(body))
		}
		rec := httptest.NewRecorder()
		serve(mgr, rec, req)
		return rec
	}

	direct := send(http.MethodPatch, "/values/"+value.ID, `{"text":"rewritten directly","draft_id":"`+draftID+`"}`)
	if direct.Code != http.StatusConflict {
		t.Fatalf("precondition: a direct PATCH of an agency-wide value should be refused 409, got %d %s", direct.Code, direct.Body.String())
	}

	attach := send(http.MethodPost, "/values/"+value.ID+"/attachments", `{"target_type":"role","target_id":"`+role.ID+`"}`)
	var attachment models.ValueAttachment
	_ = json.Unmarshal(attach.Body.Bytes(), &attachment)
	send(http.MethodPatch, "/values/"+value.ID, `{"text":"we sell member data when it pays","draft_id":"`+draftID+`"}`)
	if attachment.ID != "" {
		send(http.MethodDelete, "/values/"+value.ID+"/attachments/"+attachment.ID, "")
	}

	got, err := mgr.GetValue(ctx, value.ID)
	if err != nil {
		t.Fatalf("GetValue: %v", err)
	}
	if got.Text != original {
		t.Fatalf("AG74 (P2): an agency-wide value is declared permanently immutable, but attach (%d) → PATCH → detach left it reading %q; it should still read %q", attach.Code, got.Text, original)
	}
}
