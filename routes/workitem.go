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
	case errors.Is(err, mwanachamaagency.ErrInvalidWorkItem):
		return http.StatusBadRequest
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

// ListWorkItems handles GET — ?draft_id= selects scope ("" = live);
// ?code=, ?title=, ?description= each narrow by a case-insensitive
// substring match.
func ListWorkItems(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.WorkItemFilter{Code: q.Get("code"), Title: q.Get("title"), Description: q.Get("description")}
		out, err := m.ListWorkItems(r.Context(), q.Get("draft_id"), filter)
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

// WorkItemRoutes is Create/Get/List/Update for WorkItem, addressed under
// names.WorkItem. The nested role-assignments read moved to
// [ObjectiveRoutes] 2026-09-10 (AGD-001) — RACI is Objective-scoped, not
// WorkItem-scoped.
func WorkItemRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.WorkItem
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateWorkItem(m)},
		{Method: http.MethodGet, Path: base, Handler: ListWorkItems(m)},
		{Method: http.MethodGet, Path: base + "/{workItemID}", Handler: GetWorkItem(m)},
		{Method: http.MethodPatch, Path: base + "/{workItemID}", Handler: UpdateWorkItem(m)},
	}
}
