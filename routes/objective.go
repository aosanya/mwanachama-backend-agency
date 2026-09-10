package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func objectiveStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrObjectiveNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeObjectiveErr(w http.ResponseWriter, err error) {
	code := objectiveStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateObjective handles POST — decode, create, encode. A Goal may have
// more than one Objective.
func CreateObjective(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Objective
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateObjective(r.Context(), in)
		if err != nil {
			writeObjectiveErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetObjective handles GET {objectiveID}.
func GetObjective(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetObjective(r.Context(), r.PathValue("objectiveID"))
		if err != nil {
			writeObjectiveErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListObjectives handles GET — ?draft_id= selects scope ("" = live).
func ListObjectives(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListObjectives(r.Context(), r.URL.Query().Get("draft_id"))
		if err != nil {
			writeObjectiveErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateObjective handles PATCH {objectiveID}.
func UpdateObjective(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Objective
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("objectiveID")
		out, err := m.UpdateObjective(r.Context(), in)
		if err != nil {
			writeObjectiveErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListRoleAssignmentsForObjective handles GET {objectiveID}/role-assignments
// — ?draft_id= selects scope ("" = live). Moved here from WorkItemRoutes
// 2026-09-10 (AGD-001) — RACI is Objective-scoped, not WorkItem-scoped.
func ListRoleAssignmentsForObjective(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListRoleAssignmentsForObjective(r.Context(), r.PathValue("objectiveID"), r.URL.Query().Get("draft_id"))
		if err != nil {
			writeRoleAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ObjectiveRoutes is Create/Get/List/Update for Objective, addressed under
// names.Objective, plus the nested role-assignments read under
// names.RoleAssignment.
func ObjectiveRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Objective
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateObjective(m)},
		{Method: http.MethodGet, Path: base, Handler: ListObjectives(m)},
		{Method: http.MethodGet, Path: base + "/{objectiveID}", Handler: GetObjective(m)},
		{Method: http.MethodPatch, Path: base + "/{objectiveID}", Handler: UpdateObjective(m)},
		{Method: http.MethodGet, Path: base + "/{objectiveID}/" + names.RoleAssignment, Handler: ListRoleAssignmentsForObjective(m)},
	}
}
