package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-agency/routes"
)

func TestCreateAndGetDepartment(t *testing.T) {
	mgr := newTestManager(t)
	req := httptest.NewRequest(http.MethodPost, "/departments", strings.NewReader(`{"name":"Software"}`))
	rec := httptest.NewRecorder()
	routes.CreateDepartment(mgr)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out models.Department
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/departments/"+out.ID, nil)
	getReq.SetPathValue("departmentID", out.ID)
	getRec := httptest.NewRecorder()
	routes.GetDepartment(mgr)(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}
}

func TestListDepartmentChildrenAndRename(t *testing.T) {
	mgr := newTestManager(t)
	root, _ := mgr.CreateDepartment(t.Context(), models.Department{Name: "Root"})
	child, err := mgr.CreateDepartment(t.Context(), models.Department{Name: "Child", ParentID: root.ID})
	if err != nil {
		t.Fatalf("CreateDepartment(child): %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/departments/"+root.ID+"/children", nil)
	listReq.SetPathValue("departmentID", root.ID)
	listRec := httptest.NewRecorder()
	routes.ListDepartmentChildren(mgr)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var children []models.Department
	if err := json.Unmarshal(listRec.Body.Bytes(), &children); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("unexpected children: %+v", children)
	}

	renameReq := httptest.NewRequest(http.MethodPatch, "/departments/"+child.ID, strings.NewReader(`{"name":"Renamed"}`))
	renameReq.SetPathValue("departmentID", child.ID)
	renameRec := httptest.NewRecorder()
	routes.RenameDepartment(mgr)(renameRec, renameReq)
	if renameRec.Code != http.StatusOK {
		t.Fatalf("rename status = %d, body = %s", renameRec.Code, renameRec.Body.String())
	}
}

func TestMoveDepartment(t *testing.T) {
	mgr := newTestManager(t)
	root, _ := mgr.CreateDepartment(t.Context(), models.Department{Name: "Root"})
	a, _ := mgr.CreateDepartment(t.Context(), models.Department{Name: "A", ParentID: root.ID})
	b, _ := mgr.CreateDepartment(t.Context(), models.Department{Name: "B", ParentID: root.ID})

	req := httptest.NewRequest(http.MethodPost, "/departments/"+a.ID+"/move", strings.NewReader(`{"parent_id":"`+b.ID+`"}`))
	req.SetPathValue("departmentID", a.ID)
	rec := httptest.NewRecorder()
	routes.MoveDepartment(mgr)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var moved models.Department
	if err := json.Unmarshal(rec.Body.Bytes(), &moved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if moved.ParentID != b.ID {
		t.Fatalf("expected ParentID %q, got %q", b.ID, moved.ParentID)
	}
}

func TestMoveDepartment_EmptyParentRefused(t *testing.T) {
	mgr := newTestManager(t)
	root, _ := mgr.CreateDepartment(t.Context(), models.Department{Name: "Root"})
	a, _ := mgr.CreateDepartment(t.Context(), models.Department{Name: "A", ParentID: root.ID})

	req := httptest.NewRequest(http.MethodPost, "/departments/"+a.ID+"/move", strings.NewReader(`{"parent_id":""}`))
	req.SetPathValue("departmentID", a.ID)
	rec := httptest.NewRecorder()
	routes.MoveDepartment(mgr)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestMoveDepartment_RootCannotMove(t *testing.T) {
	mgr := newTestManager(t)
	root, _ := mgr.CreateDepartment(t.Context(), models.Department{Name: "Root"})
	child, _ := mgr.CreateDepartment(t.Context(), models.Department{Name: "Child", ParentID: root.ID})

	req := httptest.NewRequest(http.MethodPost, "/departments/"+root.ID+"/move", strings.NewReader(`{"parent_id":"`+child.ID+`"}`))
	req.SetPathValue("departmentID", root.ID)
	rec := httptest.NewRecorder()
	routes.MoveDepartment(mgr)(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
