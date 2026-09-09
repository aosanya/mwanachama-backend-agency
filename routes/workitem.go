package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func workItemStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrWorkItemNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeWorkItemErr(w http.ResponseWriter, err error) {
	code := workItemStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateWorkItem handles POST — decode, create, encode.
func CreateWorkItem(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.WorkItem
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateWorkItem(r.Context(), in)
		if err != nil {
			writeWorkItemErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetWorkItem handles GET {workItemID}.
func GetWorkItem(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetWorkItem(r.Context(), r.PathValue("workItemID"))
		if err != nil {
			writeWorkItemErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListWorkItems handles GET — ?draft_id= selects scope ("" = live).
func ListWorkItems(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListWorkItems(r.Context(), r.URL.Query().Get("draft_id"))
		if err != nil {
			writeWorkItemErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateWorkItem handles PATCH {workItemID}.
func UpdateWorkItem(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.WorkItem
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("workItemID")
		out, err := m.UpdateWorkItem(r.Context(), in)
		if err != nil {
			writeWorkItemErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListRoleAssignmentsForWorkItem handles GET {workItemID}/role-assignments
// — ?draft_id= selects scope ("" = live).
func ListRoleAssignmentsForWorkItem(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListRoleAssignmentsForWorkItem(r.Context(), r.PathValue("workItemID"), r.URL.Query().Get("draft_id"))
		if err != nil {
			writeRoleAssignmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// WorkItemRoutes is Create/Get/List/Update for WorkItem, addressed under
// names.WorkItem, plus the nested role-assignments read under
// names.RoleAssignment.
func WorkItemRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.WorkItem
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateWorkItem(m)},
		{Method: http.MethodGet, Path: base, Handler: ListWorkItems(m)},
		{Method: http.MethodGet, Path: base + "/{workItemID}", Handler: GetWorkItem(m)},
		{Method: http.MethodPatch, Path: base + "/{workItemID}", Handler: UpdateWorkItem(m)},
		{Method: http.MethodGet, Path: base + "/{workItemID}/" + names.RoleAssignment, Handler: ListRoleAssignmentsForWorkItem(m)},
	}
}
