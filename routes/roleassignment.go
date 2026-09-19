package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func roleAssignmentStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrRoleAssignmentNotFound),
		errors.Is(err, mwanachamaagency.ErrRoleNotFound),
		errors.Is(err, mwanachamaagency.ErrObjectiveNotFound),
		errors.Is(err, mwanachamaagency.ErrDraftNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrDuplicateAccountable),
		errors.Is(err, mwanachamaagency.ErrDraftNotOpen):
		return http.StatusConflict
	case errors.Is(err, mwanachamaagency.ErrInvalidRACILabel),
		errors.Is(err, mwanachamaagency.ErrDraftRequired):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeRoleAssignmentErr(w http.ResponseWriter, err error) {
	code := roleAssignmentStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateRoleAssignment handles POST — decode, create, encode. Runs the
// full validation CreateRoleAssignment does (label, scope-matched
// Role/WorkItem existence, at-most-one-Accountable).
func CreateRoleAssignment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.RoleAssignment
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateRoleAssignment(r.Context(), in)
		if err != nil {
			writeRoleAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetRoleAssignment handles GET {assignmentID}.
func GetRoleAssignment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetRoleAssignment(r.Context(), r.PathValue("assignmentID"))
		if err != nil {
			writeRoleAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListRoleAssignments handles GET — ?draft_id= selects scope ("" = live);
// ?label= narrows to an exact RACI label.
func ListRoleAssignments(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.RoleAssignmentFilter{Label: q.Get("label")}
		out, err := m.ListRoleAssignments(r.Context(), q.Get("draft_id"), filter)
		if err != nil {
			writeRoleAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateRoleAssignment handles PATCH {assignmentID}.
func UpdateRoleAssignment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.RoleAssignment
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("assignmentID")
		out, err := m.UpdateRoleAssignment(r.Context(), in)
		if err != nil {
			writeRoleAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DeleteRoleAssignment handles DELETE {assignmentID}.
func DeleteRoleAssignment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := m.DeleteRoleAssignment(r.Context(), r.PathValue("assignmentID")); err != nil {
			writeRoleAssignmentErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// RoleAssignmentRoutes is Create/Get/List/Update/Delete for
// RoleAssignment, addressed under names.RoleAssignment. The
// Objective-nested read (ListRoleAssignmentsForObjective) is registered
// by [ObjectiveRoutes] instead, since it addresses under names.Objective.
func RoleAssignmentRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.RoleAssignment
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateRoleAssignment(m)},
		{Method: http.MethodGet, Path: base, Handler: ListRoleAssignments(m)},
		{Method: http.MethodGet, Path: base + "/{assignmentID}", Handler: GetRoleAssignment(m)},
		{Method: http.MethodPatch, Path: base + "/{assignmentID}", Handler: UpdateRoleAssignment(m)},
		{Method: http.MethodDelete, Path: base + "/{assignmentID}", Handler: DeleteRoleAssignment(m)},
	}
}
