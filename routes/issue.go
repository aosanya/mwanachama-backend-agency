package routes

import (
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

// issueStatusFor has no sentinel arms of its own — ListIssues' only
// failure mode is a store error. Kept as a pair with writeIssueErr to
// match every other domain file here.
func issueStatusFor(_ error) int {
	return http.StatusInternalServerError
}

func writeIssueErr(w http.ResponseWriter, err error) {
	code := issueStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// ListIssues handles both GET /issues (the live Agency) and GET
// {draftID}/issues (one Draft) — {draftID} is absent, and so reads back
// "", on the live route. Re-scans before responding (see
// [mwanachamaagency.AgencyManager.ListIssues]). ?message= narrows by a
// case-insensitive substring match; ?severity=/?entity_type= narrow by an
// exact match.
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

// IssueRoutes is List in both scopes: the top-level names.Issue
// collection scans the live Agency, the names.Draft/{draftID}-nested one
// scans that Draft.
func IssueRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	handler := ListIssues(m)
	return []Route{
		{Method: http.MethodGet, Path: "/" + names.Issue, Handler: handler},
		{Method: http.MethodGet, Path: "/" + names.Draft + "/{draftID}/" + names.Issue, Handler: handler},
	}
}
