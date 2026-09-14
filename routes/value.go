package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func valueStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrValueNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrCannotEditLive):
		return http.StatusConflict
	case errors.Is(err, mwanachamaagency.ErrInvalidValue):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeValueErr(w http.ResponseWriter, err error) {
	code := valueStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateValue handles POST — decode, create, encode.
func CreateValue(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Value
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateValue(r.Context(), in)
		if err != nil {
			writeValueErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetValue handles GET {valueID}.
func GetValue(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetValue(r.Context(), r.PathValue("valueID"))
		if err != nil {
			writeValueErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListValues handles GET — ?draft_id= selects scope ("" = live); ?code=,
// ?text= each narrow by a case-insensitive substring match.
func ListValues(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.ValueFilter{Code: q.Get("code"), Text: q.Get("text")}
		out, err := m.ListValues(r.Context(), q.Get("draft_id"), filter)
		if err != nil {
			writeValueErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateValue handles PATCH {valueID}. Returns 409 if the target Value is
// live — see [mwanachamaagency.ErrCannotEditLive].
func UpdateValue(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Value
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("valueID")
		out, err := m.UpdateValue(r.Context(), in)
		if err != nil {
			writeValueErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DeleteValue handles DELETE {valueID}. Returns 409 if the target Value is
// live — see [mwanachamaagency.ErrCannotEditLive].
func DeleteValue(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := m.DeleteValue(r.Context(), r.PathValue("valueID")); err != nil {
			writeValueErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ValueRoutes is Create/Get/List/Update/Delete for Value, addressed under
// names.Value.
func ValueRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Value
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateValue(m)},
		{Method: http.MethodGet, Path: base, Handler: ListValues(m)},
		{Method: http.MethodGet, Path: base + "/{valueID}", Handler: GetValue(m)},
		{Method: http.MethodPatch, Path: base + "/{valueID}", Handler: UpdateValue(m)},
		{Method: http.MethodDelete, Path: base + "/{valueID}", Handler: DeleteValue(m)},
	}
}
