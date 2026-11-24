package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func widenedAgencySpec(t *testing.T, widen map[string]map[string]string) *spec.Spec {
	t.Helper()
	raw, err := os.ReadFile("agency.agency.json")
	if err != nil {
		t.Fatalf("read the shipped domain spec: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("read the shipped spec back: %v", err)
	}
	doc["instance"] = "widened"
	for _, o := range doc["objects"].([]any) {
		obj := o.(map[string]any)
		fields, ok := widen[obj["role"].(string)]
		if !ok {
			continue
		}
		var extra []any
		for name, value := range fields {
			extra = append(extra, map[string]any{"name": name, "values": []string{value}})
		}
		obj["fields"] = append(anySlice(obj["fields"]), extra...)
	}
	raw, err = json.Marshal(doc)
	if err != nil {
		t.Fatalf("rewrite the spec: %v", err)
	}
	s, err := mwanachamaagency.ParseSpec(raw)
	if err != nil {
		t.Fatalf("a domain widening the module's enums was refused at load: %v", err)
	}
	for role, fields := range widen {
		o, ok := s.ByRole(role)
		if !ok {
			t.Fatalf("precondition: the widened spec fills no %q role", role)
		}
		for name, value := range fields {
			found := false
			for _, f := range o.Fields {
				if f.Name == name && slices.Contains(f.Values, value) {
					found = true
				}
			}
			if !found {
				t.Fatalf("precondition: %s.%s does not permit the widened value %q", role, name, value)
			}
		}
	}
	return s
}

func TestWidenedAgencyVocabularyOpensNoGate(t *testing.T) {
	const (
		lockedStatus = "locked"
		deptTarget   = "department"
	)
	s := widenedAgencySpec(t, map[string]map[string]string{
		"draft":            {"status": lockedStatus},
		"value_attachment": {"target_type": deptTarget},
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := mwanachamaagency.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := mwanachamaagency.NewAgencyManager(db, s)
	if err != nil {
		t.Fatalf("NewAgencyManager: %v", err)
	}
	ctx := context.Background()

	_, _, obj, _, _ := seedLiveAgency(t, mgr, ctx)
	v, err := mgr.CreateValue(ctx, models.Value{DraftID: obj.DraftID, Text: "Correctness first"})
	if err != nil {
		t.Fatalf("precondition: CreateValue: %v", err)
	}
	if _, err := mgr.CreateValueAttachment(ctx, v.ID, models.TargetType(deptTarget), obj.ID); !errors.Is(err, mwanachamaagency.ErrInvalidTargetType) {
		t.Errorf("an attachment to the domain-declared target type %q was accepted: err = %v, want ErrInvalidTargetType", deptTarget, err)
	}

	locked, err := mgr.CreateDraft(ctx, "to be locked", "", "")
	if err != nil {
		t.Fatalf("precondition: CreateDraft: %v", err)
	}
	draftTable := s.TableFor(mustRole(t, s, "draft"))
	if err := db.Table(draftTable).Where("id = ?", locked.ID).Update("status", lockedStatus).Error; err != nil {
		t.Fatalf("precondition: set the draft to the widened status: %v", err)
	}
	got, err := mgr.GetDraft(ctx, locked.ID)
	if err != nil || string(got.Status) != lockedStatus {
		t.Fatalf("precondition: the draft reads back as %q, got %q, err %v", lockedStatus, got.Status, err)
	}

	if _, err := mgr.CreateGoal(ctx, models.Goal{DraftID: locked.ID, Title: "Slip one in"}); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Errorf("a goal was written into a draft in the domain-declared status %q: err = %v, want ErrDraftNotOpen", lockedStatus, err)
	}
	if _, err := mgr.CreateDraft(ctx, "fork it", locked.ID, "draft"); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Errorf("a draft in the domain-declared status %q was forked: err = %v, want ErrDraftNotOpen", lockedStatus, err)
	}
	if _, err := mgr.ArchiveDraft(ctx, locked.ID); !errors.Is(err, mwanachamaagency.ErrDraftNotOpen) {
		t.Errorf("a draft in the domain-declared status %q was archived as if open: err = %v, want ErrDraftNotOpen", lockedStatus, err)
	}
}

func anySlice(v any) []any {
	s, _ := v.([]any)
	return s
}
