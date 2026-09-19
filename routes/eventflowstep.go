package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func eventFlowStepStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrEventFlowStepNotFound),
		errors.Is(err, mwanachamaagency.ErrDraftNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrDraftNotOpen):
		return http.StatusConflict
	case errors.Is(err, mwanachamaagency.ErrInvalidLookup):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeEventFlowStepErr(w http.ResponseWriter, err error) {
	code := eventFlowStepStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// LookupFlowStep handles GET .../lookup — query params handler_code,
// trigger_topic, consumer, code, tried in that precedence order (see
// [mwanachamaagency.AgencyManager.LookupFlowStep]).
func LookupFlowStep(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		lookup := models.EventFlowStepLookup{
			HandlerCode:  q.Get("handler_code"),
			TriggerTopic: q.Get("trigger_topic"),
			Consumer:     q.Get("consumer"),
			Code:         q.Get("code"),
		}
		out, err := m.LookupFlowStep(r.Context(), q.Get("draft_id"), lookup)
		if err != nil {
			writeEventFlowStepErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListEventFlowSteps handles GET — ?workflow_code= optionally filters to
// one workflow's steps, ordered by (WorkflowCode, Ordinality, Step);
// ?code=, ?name=, ?description= each further narrow by a case-insensitive
// substring match.
func ListEventFlowSteps(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.EventFlowStepFilter{Code: q.Get("code"), Name: q.Get("name"), Description: q.Get("description")}
		out, err := m.ListEventFlowSteps(r.Context(), q.Get("draft_id"), q.Get("workflow_code"), filter)
		if err != nil {
			writeEventFlowStepErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// EventFlowStepRoutes is Lookup/List for EventFlowStep, addressed under
// names.EventFlowStep. Create is deliberately not exposed — populating
// steps is an internal seeding concern (see this package's docs), not a
// requested public write surface.
func EventFlowStepRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.EventFlowStep
	return []Route{
		{Method: http.MethodGet, Path: base, Handler: ListEventFlowSteps(m)},
		{Method: http.MethodGet, Path: base + "/lookup", Handler: LookupFlowStep(m)},
	}
}
