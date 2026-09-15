package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func moduleStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrModuleNotFound),
		errors.Is(err, mwanachamaagency.ErrDraftNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrDuplicateModule),
		errors.Is(err, mwanachamaagency.ErrDraftNotOpen):
		return http.StatusConflict
	case errors.Is(err, mwanachamaagency.ErrInvalidModule):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeModuleErr(w http.ResponseWriter, err error) {
	code := moduleStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// objectiveModuleStatusFor covers CreateObjectiveModule/
// DeleteObjectiveModule's own errors — used by the nested
// /{objectiveID}/modules routes registered in objective.go.
func objectiveModuleStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrObjectiveModuleNotFound),
		errors.Is(err, mwanachamaagency.ErrObjectiveNotFound),
		errors.Is(err, mwanachamaagency.ErrModuleNotFound),
		errors.Is(err, mwanachamaagency.ErrDraftNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrDuplicateObjectiveModule),
		errors.Is(err, mwanachamaagency.ErrDraftNotOpen):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeObjectiveModuleErr(w http.ResponseWriter, err error) {
	code := objectiveModuleStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateModule handles POST — decode, create, encode.
func CreateModule(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Module
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateModule(r.Context(), in)
		if err != nil {
			writeModuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetModule handles GET {moduleID}.
func GetModule(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetModule(r.Context(), r.PathValue("moduleID"))
		if err != nil {
			writeModuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListModules handles GET — ?draft_id= selects scope ("" = live); ?code=,
// ?name=, ?description= each narrow by a case-insensitive substring match;
// ?enabled=true|false narrows to an exact match.
func ListModules(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.ModuleFilter{Code: q.Get("code"), Name: q.Get("name"), Description: q.Get("description")}
		if v := q.Get("enabled"); v != "" {
			b := v == "true"
			filter.Enabled = &b
		}
		out, err := m.ListModules(r.Context(), q.Get("draft_id"), filter)
		if err != nil {
			writeModuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateModule handles PATCH {moduleID}.
func UpdateModule(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Module
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("moduleID")
		out, err := m.UpdateModule(r.Context(), in)
		if err != nil {
			writeModuleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DeleteModule handles DELETE {moduleID}. Cascades: also deletes every
// ObjectiveModule referencing it.
func DeleteModule(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := m.DeleteModule(r.Context(), r.PathValue("moduleID")); err != nil {
			writeModuleErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ModuleRoutes is Create/Get/List/Update/Delete for Module, addressed
// under names.Module.
func ModuleRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Module
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateModule(m)},
		{Method: http.MethodGet, Path: base, Handler: ListModules(m)},
		{Method: http.MethodGet, Path: base + "/{moduleID}", Handler: GetModule(m)},
		{Method: http.MethodPatch, Path: base + "/{moduleID}", Handler: UpdateModule(m)},
		{Method: http.MethodDelete, Path: base + "/{moduleID}", Handler: DeleteModule(m)},
	}
}
