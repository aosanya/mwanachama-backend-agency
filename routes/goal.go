package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func goalStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrGoalNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeGoalErr(w http.ResponseWriter, err error) {
	code := goalStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateGoal handles POST — decode, create, encode.
func CreateGoal(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Goal
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateGoal(r.Context(), in)
		if err != nil {
			writeGoalErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetGoal handles GET {goalID}.
func GetGoal(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetGoal(r.Context(), r.PathValue("goalID"))
		if err != nil {
			writeGoalErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListGoals handles GET — ?draft_id= selects scope ("" = live).
func ListGoals(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListGoals(r.Context(), r.URL.Query().Get("draft_id"))
		if err != nil {
			writeGoalErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateGoal handles PATCH {goalID}.
func UpdateGoal(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Goal
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("goalID")
		out, err := m.UpdateGoal(r.Context(), in)
		if err != nil {
			writeGoalErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// GoalRoutes is Create/Get/List/Update for Goal, addressed under names.Goal.
func GoalRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Goal
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateGoal(m)},
		{Method: http.MethodGet, Path: base, Handler: ListGoals(m)},
		{Method: http.MethodGet, Path: base + "/{goalID}", Handler: GetGoal(m)},
		{Method: http.MethodPatch, Path: base + "/{goalID}", Handler: UpdateGoal(m)},
	}
}
