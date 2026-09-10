package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func ruleStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrRuleNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrCannotEditLive):
		return http.StatusConflict
	case errors.Is(err, mwanachamaagency.ErrInvalidRule):
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

// ListRules handles GET — ?draft_id= selects scope ("" = live).
func ListRules(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListRules(r.Context(), r.URL.Query().Get("draft_id"))
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

// RuleRoutes is Create/Get/List/Update/Delete for Rule, addressed under
// names.Rule.
func RuleRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Rule
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateRule(m)},
		{Method: http.MethodGet, Path: base, Handler: ListRules(m)},
		{Method: http.MethodGet, Path: base + "/{ruleID}", Handler: GetRule(m)},
		{Method: http.MethodPatch, Path: base + "/{ruleID}", Handler: UpdateRule(m)},
		{Method: http.MethodDelete, Path: base + "/{ruleID}", Handler: DeleteRule(m)},
	}
}
