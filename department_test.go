package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestCreateDepartment_RequiresName(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.CreateDepartment(context.Background(), models.Department{}); !errors.Is(err, mwanachamaagency.ErrInvalidDepartment) {
		t.Fatalf("expected ErrInvalidDepartment, got %v", err)
	}
}

func TestDepartmentTreeAndRename(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, err := mgr.CreateDepartment(ctx, models.Department{Name: "Software"})
	if err != nil {
		t.Fatalf("CreateDepartment(root): %v", err)
	}
	child, err := mgr.CreateDepartment(ctx, models.Department{Name: "Backend", ParentID: root.ID})
	if err != nil {
		t.Fatalf("CreateDepartment(child): %v", err)
	}

	children, err := mgr.ListDepartmentChildren(ctx, root.ID)
	if err != nil {
		t.Fatalf("ListDepartmentChildren: %v", err)
	}
	if len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("expected root's one child, got %+v", children)
	}

	renamed, err := mgr.RenameDepartment(ctx, child.ID, "Platform")
	if err != nil {
		t.Fatalf("RenameDepartment: %v", err)
	}
	if renamed.Name != "Platform" {
		t.Fatalf("expected renamed department, got %+v", renamed)
	}

	if _, err := mgr.RenameDepartment(ctx, child.ID, ""); !errors.Is(err, mwanachamaagency.ErrInvalidDepartment) {
		t.Fatalf("expected ErrInvalidDepartment renaming to empty, got %v", err)
	}
}

func TestMoveDepartment_RootCannotMove(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Root"})
	child, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Child", ParentID: root.ID})

	if _, err := mgr.MoveDepartment(ctx, root.ID, child.ID); !errors.Is(err, mwanachamaagency.ErrRootCannotMove) {
		t.Fatalf("expected ErrRootCannotMove, got %v", err)
	}
}

func TestMoveDepartment_ParentIsSelf(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Root"})
	child, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Child", ParentID: root.ID})

	if _, err := mgr.MoveDepartment(ctx, child.ID, child.ID); !errors.Is(err, mwanachamaagency.ErrParentIsSelf) {
		t.Fatalf("expected ErrParentIsSelf, got %v", err)
	}
}

func TestMoveDepartment_ParentInSubtree(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Root"})
	mid, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Mid", ParentID: root.ID})
	leaf, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Leaf", ParentID: mid.ID})

	if _, err := mgr.MoveDepartment(ctx, mid.ID, leaf.ID); !errors.Is(err, mwanachamaagency.ErrParentInSubtree) {
		t.Fatalf("expected ErrParentInSubtree, got %v", err)
	}
}

func TestMoveDepartment_ParentNotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Root"})
	child, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Child", ParentID: root.ID})

	if _, err := mgr.MoveDepartment(ctx, child.ID, "unknown"); !errors.Is(err, mwanachamaagency.ErrParentNotFound) {
		t.Fatalf("expected ErrParentNotFound, got %v", err)
	}
}

func TestMoveDepartment_Succeeds(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	root, _ := mgr.CreateDepartment(ctx, models.Department{Name: "Root"})
	a, _ := mgr.CreateDepartment(ctx, models.Department{Name: "A", ParentID: root.ID})
	b, _ := mgr.CreateDepartment(ctx, models.Department{Name: "B", ParentID: root.ID})

	moved, err := mgr.MoveDepartment(ctx, a.ID, b.ID)
	if err != nil {
		t.Fatalf("MoveDepartment: %v", err)
	}
	if moved.ParentID != b.ID {
		t.Fatalf("expected ParentID %q, got %q", b.ID, moved.ParentID)
	}
}

func TestGetDepartment_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetDepartment(context.Background(), "unknown"); !errors.Is(err, mwanachamaagency.ErrDepartmentNotFound) {
		t.Fatalf("expected ErrDepartmentNotFound, got %v", err)
	}
}
