package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func roleStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrRoleNotFound),
		errors.Is(err, mwanachamaagency.ErrDraftNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrDraftNotOpen):
		return http.StatusConflict
	case errors.Is(err, mwanachamaagency.ErrInvalidRole),
		errors.Is(err, mwanachamaagency.ErrDraftRequired):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeRoleErr(w http.ResponseWriter, err error) {
	code := roleStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateRole handles POST — decode, create, encode.
func CreateRole(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Role
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateRole(r.Context(), in)
		if err != nil {
			writeRoleErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetRole handles GET {roleID}.
func GetRole(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetRole(r.Context(), r.PathValue("roleID"))
		if err != nil {
			writeRoleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListRoles handles GET — ?draft_id= selects scope ("" = live); ?code=,
// ?name=, ?description= each narrow by a case-insensitive substring match.
func ListRoles(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.RoleFilter{Code: q.Get("code"), Name: q.Get("name"), Description: q.Get("description")}
		out, err := m.ListRoles(r.Context(), q.Get("draft_id"), filter)
		if err != nil {
			writeRoleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateRole handles PATCH {roleID}.
func UpdateRole(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Role
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("roleID")
		out, err := m.UpdateRole(r.Context(), in)
		if err != nil {
			writeRoleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DeleteRole handles DELETE {roleID}.
func DeleteRole(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := m.DeleteRole(r.Context(), r.PathValue("roleID")); err != nil {
			writeRoleErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// RoleRoutes is Create/Get/List/Update/Delete for Role, addressed under
// names.Role.
func RoleRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Role
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateRole(m)},
		{Method: http.MethodGet, Path: base, Handler: ListRoles(m)},
		{Method: http.MethodGet, Path: base + "/{roleID}", Handler: GetRole(m)},
		{Method: http.MethodPatch, Path: base + "/{roleID}", Handler: UpdateRole(m)},
		{Method: http.MethodDelete, Path: base + "/{roleID}", Handler: DeleteRole(m)},
	}
}
