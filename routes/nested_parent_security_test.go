package routes_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestAG55_OpenHole_NestedRouteActsOnAChildOfAnotherParent(t *testing.T) {
	call := func(mgr mwanachamaagency.AgencyManager, method, path, body string) (int, string) {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		serve(mgr, w, r)
		return w.Code, strings.TrimSpace(w.Body.String())
	}

	t.Run("detach_value", func(t *testing.T) {
		mgr, d := newTestManagerWithDraft(t)
		owner, _ := mgr.CreateValue(t.Context(), models.Value{DraftID: d, Text: "owner"})
		other, _ := mgr.CreateValue(t.Context(), models.Value{DraftID: d, Text: "other"})
		role, _ := mgr.CreateRole(t.Context(), models.Role{DraftID: d, Name: "reviewer"})
		att, err := mgr.CreateValueAttachment(t.Context(), owner.ID, models.TargetTypeRole, role.ID)
		if err != nil || other.ID == "" {
			t.Fatalf("precondition: CreateValueAttachment: %v", err)
		}

		code, body := call(mgr, http.MethodDelete, "/values/"+other.ID+"/attachments/"+att.ID, "")
		left, _ := mgr.ListValueAttachments(t.Context(), owner.ID)
		if code != http.StatusNotFound || len(left) != 1 {
			t.Fatalf("AG55 (P2): DELETE /values/{another value}/attachments/{owner's attachment} answered %d %q and left %d attachment(s) on the owning value; want 404 and the owner's 1 attachment untouched", code, body, len(left))
		}
	})

	t.Run("detach_rule", func(t *testing.T) {
		mgr, d := newTestManagerWithDraft(t)
		owner, _ := mgr.CreateRule(t.Context(), models.Rule{DraftID: d, Text: "owner"})
		role, _ := mgr.CreateRole(t.Context(), models.Role{DraftID: d, Name: "reviewer"})
		att, err := mgr.CreateRuleAttachment(t.Context(), owner.ID, models.TargetTypeRole, role.ID)
		if err != nil {
			t.Fatalf("precondition: CreateRuleAttachment: %v", err)
		}

		code, body := call(mgr, http.MethodDelete, "/rules/no-such-rule/attachments/"+att.ID, "")
		left, _ := mgr.ListRuleAttachments(t.Context(), owner.ID)
		if code != http.StatusNotFound || len(left) != 1 {
			t.Fatalf("AG55 (P2): DELETE /rules/no-such-rule/attachments/{owner's attachment} answered %d %q and left %d attachment(s) on the owning rule; want 404 and the owner's 1 attachment untouched", code, body, len(left))
		}
	})

	t.Run("update_objective_module", func(t *testing.T) {
		mgr := newTestManager(t)
		if _, err := mgr.CreateAgency(t.Context(), models.Agency{Name: "SWELab"}); err != nil {
			t.Fatalf("CreateAgency: %v", err)
		}
		d := bootstrapDraftID(t, mgr)
		mods, err := mgr.ListModules(t.Context(), d, mwanachamaagency.ModuleFilter{})
		if err != nil || len(mods) == 0 {
			t.Fatalf("precondition: the bootstrap draft holds no modules (%v)", err)
		}
		g, _ := mgr.CreateGoal(t.Context(), models.Goal{DraftID: d, Title: "g"})
		owner, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: d, GoalID: g.ID, Title: "owner"})
		other, _ := mgr.CreateObjective(t.Context(), models.Objective{DraftID: d, GoalID: g.ID, Title: "other"})
		link, err := mgr.CreateObjectiveModule(t.Context(), models.ObjectiveModule{DraftID: d, ObjectiveID: owner.ID, ModuleID: mods[0].ID, Rationale: "original"})
		if err != nil || other.ID == "" {
			t.Fatalf("precondition: CreateObjectiveModule: %v", err)
		}

		code, body := call(mgr, http.MethodPatch, "/objectives/"+other.ID+"/modules/"+link.ID, `{"rationale":"rewritten through another objective"}`)
		links, _ := mgr.ListObjectiveModulesForObjective(t.Context(), owner.ID, d, mwanachamaagency.ObjectiveModuleFilter{})
		if code != http.StatusNotFound || len(links) != 1 || links[0].Rationale != "original" {
			t.Fatalf("AG55 (P2): PATCH /objectives/{another objective}/modules/{owner's link} answered %d %q and the owner's link now reads %+v; want 404 and the rationale left \"original\"", code, body, links)
		}
	})
}
