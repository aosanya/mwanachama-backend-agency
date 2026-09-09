package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func workPlanStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrWorkPlanNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrInvalidRegex):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeWorkPlanErr(w http.ResponseWriter, err error) {
	code := workPlanStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateWorkPlan handles POST — decode, create, encode. Returns 400 if
// TriggerTopic/PayloadCondition don't compile as a Go regexp.
func CreateWorkPlan(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.WorkPlan
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateWorkPlan(r.Context(), in)
		if err != nil {
			writeWorkPlanErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetWorkPlan handles GET {workPlanID}.
func GetWorkPlan(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetWorkPlan(r.Context(), r.PathValue("workPlanID"))
		if err != nil {
			writeWorkPlanErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListWorkPlans handles GET — every work plan, unfiltered.
func ListWorkPlans(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListWorkPlans(r.Context())
		if err != nil {
			writeWorkPlanErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateWorkPlan handles PATCH {workPlanID}.
func UpdateWorkPlan(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.WorkPlan
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		in.ID = r.PathValue("workPlanID")
		out, err := m.UpdateWorkPlan(r.Context(), in)
		if err != nil {
			writeWorkPlanErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// MatchWorkPlans handles POST .../match — body {"topic": "...", "payload":
// "..."}, POST rather than query params since payload is free-form
// content, not a short query-string-safe value.
func MatchWorkPlans(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Topic   string `json:"topic"`
			Payload string `json:"payload"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.MatchWorkPlans(r.Context(), body.Topic, body.Payload)
		if err != nil {
			writeWorkPlanErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// WorkPlanRoutes is Create/Get/List/Update/Match for WorkPlan, addressed
// under names.WorkPlan.
func WorkPlanRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.WorkPlan
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateWorkPlan(m)},
		{Method: http.MethodGet, Path: base, Handler: ListWorkPlans(m)},
		{Method: http.MethodPost, Path: base + "/match", Handler: MatchWorkPlans(m)},
		{Method: http.MethodGet, Path: base + "/{workPlanID}", Handler: GetWorkPlan(m)},
		{Method: http.MethodPatch, Path: base + "/{workPlanID}", Handler: UpdateWorkPlan(m)},
	}
}
