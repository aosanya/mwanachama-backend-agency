package routes

import (
	"errors"
	"net/http"
	"strconv"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func publicationStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrPublicationNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrDraftNotPromoted),
		errors.Is(err, mwanachamaagency.ErrNoChangesDetected),
		errors.Is(err, mwanachamaagency.ErrPublishValidationFailed),
		errors.Is(err, mwanachamaagency.ErrDuplicateTag):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writePublicationErr(w http.ResponseWriter, err error) {
	code := publicationStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// pathVersion parses the {version} path value as an int, writing a 400
// and returning ok=false if it isn't one.
func pathVersion(w http.ResponseWriter, r *http.Request) (int, bool) {
	v, err := strconv.Atoi(r.PathValue("version"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "version must be an integer")
		return 0, false
	}
	return v, true
}

// PublishAgency handles POST — body {"draft_id": "...", "tag": "...",
// "channel": "..."}, all "" allowed, same as
// [mwanachamaagency.AgencyManager.PublishAgency].
func PublishAgency(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DraftID string `json:"draft_id"`
			Tag     string `json:"tag"`
			Channel string `json:"channel"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.PublishAgency(r.Context(), body.DraftID, body.Tag, body.Channel)
		if err != nil {
			writePublicationErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetPublication handles GET {version}.
func GetPublication(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version, ok := pathVersion(w, r)
		if !ok {
			return
		}
		out, err := m.GetPublication(r.Context(), version)
		if err != nil {
			writePublicationErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListPublications handles GET — every publication, ordered by version;
// ?tag= narrows by substring match, ?channel= by exact match.
func ListPublications(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.PublicationFilter{Tag: q.Get("tag"), Channel: q.Get("channel")}
		out, err := m.ListPublications(r.Context(), filter)
		if err != nil {
			writePublicationErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// SetPublicationChannel handles PATCH {version} — body
// {"channel": "..."}. An empty channel archives the publication; any other
// value moves that channel onto it, off whichever publication held it.
func SetPublicationChannel(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version, ok := pathVersion(w, r)
		if !ok {
			return
		}
		var body struct {
			Channel string `json:"channel"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.SetPublicationChannel(r.Context(), version, body.Channel)
		if err != nil {
			writePublicationErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// PublicationRoutes is Publish/Get/List/SetChannel for
// AgencyPublication, addressed under names.Publication.
func PublicationRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Publication
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: PublishAgency(m)},
		{Method: http.MethodGet, Path: base, Handler: ListPublications(m)},
		{Method: http.MethodGet, Path: base + "/{version}", Handler: GetPublication(m)},
		{Method: http.MethodPatch, Path: base + "/{version}", Handler: SetPublicationChannel(m)},
	}
}
