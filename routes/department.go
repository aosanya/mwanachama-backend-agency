package routes

import (
	"errors"
	"net/http"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func departmentStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaagency.ErrDepartmentNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaagency.ErrRootCannotMove),
		errors.Is(err, mwanachamaagency.ErrParentIsSelf),
		errors.Is(err, mwanachamaagency.ErrParentInSubtree):
		return http.StatusConflict
	case errors.Is(err, mwanachamaagency.ErrParentNotFound),
		errors.Is(err, mwanachamaagency.ErrInvalidDepartment):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeDepartmentErr(w http.ResponseWriter, err error) {
	code := departmentStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateDepartment handles POST — decode, create, encode.
func CreateDepartment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in models.Department
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.CreateDepartment(r.Context(), in)
		if err != nil {
			writeDepartmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetDepartment handles GET {departmentID}.
func GetDepartment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := m.GetDepartment(r.Context(), r.PathValue("departmentID"))
		if err != nil {
			writeDepartmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListDepartments handles GET — ?code=, ?name= each narrow by a
// case-insensitive substring match.
func ListDepartments(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.DepartmentFilter{Code: q.Get("code"), Name: q.Get("name")}
		out, err := m.ListDepartments(r.Context(), filter)
		if err != nil {
			writeDepartmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListDepartmentChildren handles GET {departmentID}/children; ?code=,
// ?name= each narrow by a case-insensitive substring match.
func ListDepartmentChildren(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter := mwanachamaagency.DepartmentFilter{Code: q.Get("code"), Name: q.Get("name")}
		out, err := m.ListDepartmentChildren(r.Context(), r.PathValue("departmentID"), filter)
		if err != nil {
			writeDepartmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// RenameDepartment handles PATCH {departmentID} — body {"name": "..."}.
func RenameDepartment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := m.RenameDepartment(r.Context(), r.PathValue("departmentID"), body.Name)
		if err != nil {
			writeDepartmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// MoveDepartment handles POST {departmentID}/move — body
// {"parent_id": "..."}. An empty parent_id is refused as bad request, not
// read as "move to root" — matches actor's MoveGroup handler.
func MoveDepartment(m mwanachamaagency.AgencyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ParentID string `json:"parent_id"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.ParentID == "" {
			writeErr(w, http.StatusBadRequest, "a new parent department is required")
			return
		}
		out, err := m.MoveDepartment(r.Context(), r.PathValue("departmentID"), body.ParentID)
		if err != nil {
			writeDepartmentErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// DepartmentRoutes is Create/Get/List/ListChildren/Rename/Move for
// Department, addressed under names.Department.
func DepartmentRoutes(m mwanachamaagency.AgencyManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Department
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateDepartment(m)},
		{Method: http.MethodGet, Path: base, Handler: ListDepartments(m)},
		{Method: http.MethodGet, Path: base + "/{departmentID}", Handler: GetDepartment(m)},
		{Method: http.MethodGet, Path: base + "/{departmentID}/children", Handler: ListDepartmentChildren(m)},
		{Method: http.MethodPatch, Path: base + "/{departmentID}", Handler: RenameDepartment(m)},
		{Method: http.MethodPost, Path: base + "/{departmentID}/move", Handler: MoveDepartment(m)},
	}
}
