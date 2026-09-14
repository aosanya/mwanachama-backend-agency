package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func levelStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrLevelNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrLevelWornByDepartments):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeLevelErr(w http.ResponseWriter, err error) {
	code := levelStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateLevel handles POST — decode, create, encode.
func CreateLevel(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Level
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateLevel(r.Context(), in)
		if err != nil {
			writeLevelErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetLevel handles GET {levelID}.
func GetLevel(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetLevel(r.Context(), r.PathValue("levelID"))
		if err != nil {
			writeLevelErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListLevels handles GET — every level, ordered by (Depth, ID); ?code=,
// ?name= each narrow by a case-insensitive substring match.
func ListLevels(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.LevelFilter{Code: q.Get("code"), Name: q.Get("name")}
		out, err := m.ListLevels(r.Context(), filter)
		if err != nil {
			writeLevelErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// RenameLevel handles PATCH {levelID} — body {"name": "..."}.
func RenameLevel(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.RenameLevel(r.Context(), r.PathValue("levelID"), body.Name)
		if err != nil {
			writeLevelErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DeleteLevel handles DELETE {levelID}. Returns 409 if a Department still
// references it — see [mwanachamaagency.ErrLevelWornByDepartments].
func DeleteLevel(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := m.DeleteLevel(r.Context(), r.PathValue("levelID")); err != nil {
			writeLevelErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// LevelRoutes is Create/Get/List/Rename/Delete for Level, addressed under
// names.Level.
func LevelRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Level
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateLevel(m)},
		{Method: http.MethodGet, Path: base, Handler: ListLevels(m)},
		{Method: http.MethodGet, Path: base + "/{levelID}", Handler: GetLevel(m)},
		{Method: http.MethodPatch, Path: base + "/{levelID}", Handler: RenameLevel(m)},
		{Method: http.MethodDelete, Path: base + "/{levelID}", Handler: DeleteLevel(m)},
	}
}
