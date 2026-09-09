package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func draftStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrDraftNotFound),
		errors.Is(err, mwanachamaagency.ErrAgencyNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrDraftNotOpen):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeDraftErr(w http.ResponseWriter, err error) {
	code := draftStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateDraft handles POST — body {description, forked_from_id,
// forked_from_type}, create, encode.
func CreateDraft(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Description    string `json:"description"`
			ForkedFromID   string `json:"forked_from_id"`
			ForkedFromType string `json:"forked_from_type"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateDraft(r.Context(), body.Description, body.ForkedFromID, body.ForkedFromType)
		if err != nil {
			writeDraftErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetDraft handles GET {draftID}.
func GetDraft(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetDraft(r.Context(), r.PathValue("draftID"))
		if err != nil {
			writeDraftErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListDrafts handles GET — every draft, unfiltered.
func ListDrafts(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ListDrafts(r.Context())
		if err != nil {
			writeDraftErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// UpdateDraftDescription handles PATCH {draftID} — body
// {"description": "..."}. Returns 409 if the draft isn't open.
func UpdateDraftDescription(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Description string `json:"description"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.UpdateDraftDescription(r.Context(), r.PathValue("draftID"), body.Description)
		if err != nil {
			writeDraftErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// PromoteDraft handles POST {draftID}/promote. Returns 409 if the draft
// isn't open.
func PromoteDraft(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.PromoteDraft(r.Context(), r.PathValue("draftID"))
		if err != nil {
			writeDraftErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ArchiveDraft handles POST {draftID}/archive. Returns 409 if the draft
// isn't open.
func ArchiveDraft(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.ArchiveDraft(r.Context(), r.PathValue("draftID"))
		if err != nil {
			writeDraftErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DraftRoutes is Create/Get/List/UpdateDescription/Promote/Archive for
// Draft, addressed under names.Draft.
func DraftRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Draft
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateDraft(m)},
		{Method: http.MethodGet, Path: base, Handler: ListDrafts(m)},
		{Method: http.MethodGet, Path: base + "/{draftID}", Handler: GetDraft(m)},
		{Method: http.MethodPatch, Path: base + "/{draftID}", Handler: UpdateDraftDescription(m)},
		{Method: http.MethodPost, Path: base + "/{draftID}/promote", Handler: PromoteDraft(m)},
		{Method: http.MethodPost, Path: base + "/{draftID}/archive", Handler: ArchiveDraft(m)},
	}
}
