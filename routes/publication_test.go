package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

// seedPublishableAgency builds the smallest Agency that passes
// publish-time validation: one Goal, one Objective under it, and an
// Accountable Role on that Objective, inside a fresh Draft — returns that
// Draft's id.
func seedPublishableAgency(t *testing.T, mgr mwanachamaagency.AgencyManager) string {
	t.Helper()
	ctx := t.Context()
	if _, err := mgr.CreateAgency(ctx, models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(ctx, "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	g, err := mgr.CreateGoal(ctx, models.Goal{DraftID: d.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	o, err := mgr.CreateObjective(ctx, models.Objective{DraftID: d.ID, GoalID: g.ID, Title: "Grow adoption"})
	if err != nil {
		t.Fatalf("CreateObjective: %v", err)
	}
	role, err := mgr.CreateRole(ctx, models.Role{DraftID: d.ID, Name: "owner"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := mgr.CreateRoleAssignment(ctx, models.RoleAssignment{
		DraftID: d.ID, RoleID: role.ID, ObjectiveID: o.ID, Label: models.RACILabelAccountable,
	}); err != nil {
		t.Fatalf("CreateRoleAssignment: %v", err)
	}
	return d.ID
}

// TestPublishAgency_ValidationFailureIsConflict asserts a Draft with an
// error-severity Issue (here: no Goals at all) is refused with a 409
// naming what is wrong, rather than a bare 500.
func TestPublishAgency_ValidationFailureIsConflict(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "SWELab"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	d, err := mgr.CreateDraft(t.Context(), "empty", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+d.ID+`"}`))
	rec := httptest.NewRecorder()
	routes.PublishAgency(mgr)(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "no Goals") {
		t.Errorf("expected the body to name the finding, got %s", rec.Body.String())
	}
}

// TestPublishAgency_Tag covers the request body's new tag field, and the
// 409 a duplicate earns.
func TestPublishAgency_Tag(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)

	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`","tag":"launch"}`))
	rec := httptest.NewRecorder()
	routes.PublishAgency(mgr)(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var pub models.AgencyPublication
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pub.Tag != "launch" {
		t.Fatalf("expected tag %q, got %q", "launch", pub.Tag)
	}

	if _, err := mgr.CreateValue(t.Context(), models.Value{DraftID: draftID, Text: "correctness over speed"}); err != nil {
		t.Fatalf("CreateValue: %v", err)
	}
	dup := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`","tag":"launch"}`))
	dupRec := httptest.NewRecorder()
	routes.PublishAgency(mgr)(dupRec, dup)
	if dupRec.Code != http.StatusConflict {
		t.Fatalf("duplicate-tag status = %d, body = %s", dupRec.Code, dupRec.Body.String())
	}
}

func TestPublishAgencyAndGetPublication(t *testing.T) {
	mgr := newTestManager(t)
	draftID := seedPublishableAgency(t, mgr)

	req := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`"}`))
	rec := httptest.NewRecorder()
	routes.PublishAgency(mgr)(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var pub models.AgencyPublication
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pub.Version != 1 {
		t.Fatalf("expected version 1, got %d", pub.Version)
	}

	// Republishing unchanged content is refused.
	req2 := httptest.NewRequest(http.MethodPost, "/publications", strings.NewReader(`{"draft_id":"`+draftID+`"}`))
	rec2 := httptest.NewRecorder()
	routes.PublishAgency(mgr)(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("re-publish status = %d, body = %s", rec2.Code, rec2.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/publications/1", nil)
	getReq.SetPathValue("version", "1")
	getRec := httptest.NewRecorder()
	routes.GetPublication(mgr)(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/publications", nil)
	listRec := httptest.NewRecorder()
	routes.ListPublications(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodPatch, "/publications/1", strings.NewReader(`{"channel":""}`))
	statusReq.SetPathValue("version", "1")
	statusRec := httptest.NewRecorder()
	routes.SetPublicationChannel(mgr)(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", statusRec.Code, statusRec.Body.String())
	}
}

func TestGetPublication_BadVersion(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodGet, "/publications/not-a-number", nil)
	req.SetPathValue("version", "not-a-number")
	rec := httptest.NewRecorder()
	routes.GetPublication(mgr)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
