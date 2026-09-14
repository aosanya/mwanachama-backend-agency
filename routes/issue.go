package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func issueStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrDraftIDRequired):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeIssueErr(w http.ResponseWriter, err error) {
	code := issueStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// ListIssues handles GET {draftID}/issues. Re-scans the Draft before
// responding (see [mwanachamaagency.AgencyManager.ListIssues]). ?message=
// narrows by a case-insensitive substring match; ?severity=/?entity_type=
// narrow by an exact match.
func ListIssues(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.IssueFilter{
			Message:    q.Get("message"),
			Severity:   q.Get("severity"),
			EntityType: q.Get("entity_type"),
		}
		out, err := m.ListIssues(r.Context(), r.PathValue("draftID"), filter)
		if err != nil {
			writeIssueErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// IssueRoutes is the single List route for Issue, nested under
// names.Draft/{draftID}/names.Issue — an Issue only ever exists scoped to
// one Draft, so unlike Goal/Objective/etc. there is no top-level
// names.Issue collection.
func IssueRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Draft + "/{draftID}/" + names.Issue
	return []Route{
		{Method: http.MethodGet, Path: base, Handler: ListIssues(m)},
	}
}
