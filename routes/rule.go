package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func ruleStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrRuleNotFound),
		errors.Is(err, mwanachamaagency.ErrRuleAttachmentNotFound),
		errors.Is(err, mwanachamaagency.ErrRoleNotFound),
		errors.Is(err, mwanachamaagency.ErrGoalNotFound),
		errors.Is(err, mwanachamaagency.ErrObjectiveNotFound),
		errors.Is(err, mwanachamaagency.ErrWorkflowNotFound),
		errors.Is(err, mwanachamaagency.ErrWorkItemNotFound),
		errors.Is(err, mwanachamaagency.ErrDraftNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrCannotEditLive),
		errors.Is(err, mwanachamaagency.ErrDuplicateAttachment),
		errors.Is(err, mwanachamaagency.ErrDraftNotOpen):
		return http.StatusConflict
	case errors.Is(err, mwanachamaagency.ErrInvalidRule),
		errors.Is(err, mwanachamaagency.ErrInvalidTargetType):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeRuleErr(w http.ResponseWriter, err error) {
	code := ruleStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateRule handles POST — decode, create, encode.
func CreateRule(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Rule
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateRule(r.Context(), in)
		if err != nil {
			writeRuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetRule handles GET {ruleID}.
func GetRule(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetRule(r.Context(), r.PathValue("ruleID"))
		if err != nil {
			writeRuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListRules handles GET — ?draft_id= selects scope ("" = live); ?code=,
// ?text= each narrow by a case-insensitive substring match; ?role_id=,
// ?goal_id=, ?objective_id=, ?workflow_id=, ?work_item_id= each narrow to
// Rules attached to that target (AGD-003).
func ListRules(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.RuleFilter{
			Code: q.Get("code"), Text: q.Get("text"),
			RoleID: q.Get("role_id"), GoalID: q.Get("goal_id"), ObjectiveID: q.Get("objective_id"),
			WorkflowID: q.Get("workflow_id"), WorkItemID: q.Get("work_item_id"),
		}
		out, err := m.ListRules(r.Context(), q.Get("draft_id"), filter)
		if err != nil {
			writeRuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateRule handles PATCH {ruleID}. Returns 409 if the target Rule is
// live — see [mwanachamaagency.ErrCannotEditLive].
func UpdateRule(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Rule
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("ruleID")
		out, err := m.UpdateRule(r.Context(), in)
		if err != nil {
			writeRuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DeleteRule handles DELETE {ruleID}. Returns 409 if the target Rule is
// live — see [mwanachamaagency.ErrCannotEditLive].
func DeleteRule(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := m.DeleteRule(r.Context(), r.PathValue("ruleID")); err != nil {
			writeRuleErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// CreateRuleAttachment handles POST {ruleID}/attachments — attach a Rule
// to a Role/Goal/Objective/Workflow/WorkItem. Live-editable, no Draft
// involved (AGD-003).
func CreateRuleAttachment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			TargetType models.TargetType `json:"target_type"`
			TargetID   string            `json:"target_id"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateRuleAttachment(r.Context(), r.PathValue("ruleID"), body.TargetType, body.TargetID)
		if err != nil {
			writeRuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// ListRuleAttachments handles GET {ruleID}/attachments.
func ListRuleAttachments(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListRuleAttachments(r.Context(), r.PathValue("ruleID"))
		if err != nil {
			writeRuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DeleteRuleAttachment handles DELETE {ruleID}/attachments/{attachmentID}.
func DeleteRuleAttachment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := m.DeleteRuleAttachment(r.Context(), r.PathValue("attachmentID")); err != nil {
			writeRuleErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// RuleRoutes is Create/Get/List/Update/Delete for Rule, plus the nested
// attachments sub-resource, addressed under names.Rule.
func RuleRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Rule
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateRule(m)},
		{Method: http.MethodGet, Path: base, Handler: ListRules(m)},
		{Method: http.MethodGet, Path: base + "/{ruleID}", Handler: GetRule(m)},
		{Method: http.MethodPatch, Path: base + "/{ruleID}", Handler: UpdateRule(m)},
		{Method: http.MethodDelete, Path: base + "/{ruleID}", Handler: DeleteRule(m)},
		{Method: http.MethodPost, Path: base + "/{ruleID}/attachments", Handler: CreateRuleAttachment(m)},
		{Method: http.MethodGet, Path: base + "/{ruleID}/attachments", Handler: ListRuleAttachments(m)},
		{Method: http.MethodDelete, Path: base + "/{ruleID}/attachments/{attachmentID}", Handler: DeleteRuleAttachment(m)},
	}
}
