package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func agencyStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrAgencyNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeAgencyErr(w http.ResponseWriter, err error) {
	code := agencyStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateAgency handles POST — decode, create, encode.
func CreateAgency(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Agency
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateAgency(r.Context(), in)
		if err != nil {
			writeAgencyErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetAgency handles GET — the sole Agency row (single-agency deployment).
func GetAgency(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetAgency(r.Context())
		if err != nil {
			writeAgencyErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateAgency handles PATCH — no {agencyID}, same single-row lookup as GetAgency.
func UpdateAgency(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Agency
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.UpdateAgency(r.Context(), in)
		if err != nil {
			writeAgencyErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// AgencyRoutes is Create/Get/Update for Agency, addressed under names.Agency.
func AgencyRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Agency
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateAgency(m)},
		{Method: http.MethodGet, Path: base, Handler: GetAgency(m)},
		{Method: http.MethodPatch, Path: base, Handler: UpdateAgency(m)},
	}
}
