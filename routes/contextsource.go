package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func contextSourceStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrWorkPlanNotFound),
		errors.Is(err, mwanachamaagency.ErrContextSourceNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeContextSourceErr(w http.ResponseWriter, err error) {
	code := contextSourceStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// AddContextSource handles POST {workPlanID}/context-sources — decode,
// add, encode. Returns 404 (ErrWorkPlanNotFound) if workPlanID names no
// WorkPlan.
func AddContextSource(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.ContextSource
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.AddContextSource(r.Context(), r.PathValue("workPlanID"), in)
		if err != nil {
			writeContextSourceErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// ListContextSources handles GET {workPlanID}/context-sources.
func ListContextSources(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListContextSources(r.Context(), r.PathValue("workPlanID"))
		if err != nil {
			writeContextSourceErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// RemoveContextSource handles DELETE
// {workPlanID}/context-sources/{sourceID}. Returns 404
// (ErrContextSourceNotFound) if sourceID names no ContextSource, or one
// belonging to a different WorkPlan than workPlanID.
func RemoveContextSource(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := m.RemoveContextSource(r.Context(), r.PathValue("workPlanID"), r.PathValue("sourceID"))
		if err != nil {
			writeContextSourceErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ContextSourceRoutes is Add/List/Remove for ContextSource, nested under
// names.WorkPlan.
func ContextSourceRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.WorkPlan + "/{workPlanID}/" + names.ContextSource
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: AddContextSource(m)},
		{Method: http.MethodGet, Path: base, Handler: ListContextSources(m)},
		{Method: http.MethodDelete, Path: base + "/{sourceID}", Handler: RemoveContextSource(m)},
	}
}
