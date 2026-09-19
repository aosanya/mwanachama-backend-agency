package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func workflowStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrWorkflowNotFound),
		errors.Is(err, mwanachamaagency.ErrDraftNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrDraftNotOpen):
		return http.StatusConflict
	case errors.Is(err, mwanachamaagency.ErrDraftRequired):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeWorkflowErr(w http.ResponseWriter, err error) {
	code := workflowStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateWorkflow handles POST — decode, create, encode.
func CreateWorkflow(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Workflow
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateWorkflow(r.Context(), in)
		if err != nil {
			writeWorkflowErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetWorkflow handles GET {workflowID}.
func GetWorkflow(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetWorkflow(r.Context(), r.PathValue("workflowID"))
		if err != nil {
			writeWorkflowErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListWorkflows handles GET — ?draft_id= selects scope ("" = live);
// ?code=, ?name=, ?description= each narrow by a case-insensitive
// substring match.
func ListWorkflows(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.WorkflowFilter{Code: q.Get("code"), Name: q.Get("name"), Description: q.Get("description")}
		out, err := m.ListWorkflows(r.Context(), q.Get("draft_id"), filter)
		if err != nil {
			writeWorkflowErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateWorkflow handles PATCH {workflowID}.
func UpdateWorkflow(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Workflow
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("workflowID")
		out, err := m.UpdateWorkflow(r.Context(), in)
		if err != nil {
			writeWorkflowErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// WorkflowRoutes is Create/Get/List/Update for Workflow, addressed under
// names.Workflow.
func WorkflowRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Workflow
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateWorkflow(m)},
		{Method: http.MethodGet, Path: base, Handler: ListWorkflows(m)},
		{Method: http.MethodGet, Path: base + "/{workflowID}", Handler: GetWorkflow(m)},
		{Method: http.MethodPatch, Path: base + "/{workflowID}", Handler: UpdateWorkflow(m)},
	}
}
