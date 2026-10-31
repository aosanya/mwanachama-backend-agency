package mwanachamaagency_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type libraryManager interface {
	mwanachamaagency.AgencyManager
	CreateHazard(ctx context.Context, p models.Hazard) (models.Hazard, error)
	GetHazard(ctx context.Context, id string) (models.Hazard, error)
	UpdateHazard(ctx context.Context, p models.Hazard) (models.Hazard, error)
	DeleteHazard(ctx context.Context, id string) error
	ListHazards(ctx context.Context, draftID string, filter mwanachamaagency.LibraryPartFilter) ([]models.Hazard, error)
	CreateThreat(ctx context.Context, p models.Threat) (models.Threat, error)
	GetThreat(ctx context.Context, id string) (models.Threat, error)
	UpdateThreat(ctx context.Context, p models.Threat) (models.Threat, error)
	DeleteThreat(ctx context.Context, id string) error
	ListThreats(ctx context.Context, draftID string, filter mwanachamaagency.LibraryPartFilter) ([]models.Threat, error)
	CreateConsequence(ctx context.Context, p models.Consequence) (models.Consequence, error)
	GetConsequence(ctx context.Context, id string) (models.Consequence, error)
	UpdateConsequence(ctx context.Context, p models.Consequence) (models.Consequence, error)
	DeleteConsequence(ctx context.Context, id string) error
	ListConsequences(ctx context.Context, draftID string, filter mwanachamaagency.LibraryPartFilter) ([]models.Consequence, error)
}

func newLibraryManager(t *testing.T) libraryManager {
	t.Helper()
	return newTestManager(t).(libraryManager)
}

type libraryPartKind struct {
	name     string
	prefix   string
	create   func(mgr libraryManager, ctx context.Context, draftID, title string) (id, code string, err error)
	get      func(mgr libraryManager, ctx context.Context, id string) (code, title string, err error)
	update   func(mgr libraryManager, ctx context.Context, id, title string) error
	delete   func(mgr libraryManager, ctx context.Context, id string) error
	count    func(mgr libraryManager, ctx context.Context, draftID string) (int, error)
	notFound error
	invalid  error
}

func libraryPartKinds() []libraryPartKind {
	return []libraryPartKind{
		{
			name: "hazard", prefix: "H-", notFound: mwanachamaagency.ErrHazardNotFound, invalid: mwanachamaagency.ErrInvalidHazard,
			create: func(mgr libraryManager, ctx context.Context, d, t string) (string, string, error) {
				p, err := mgr.CreateHazard(ctx, models.Hazard{DraftID: d, Title: t})
				return p.ID, p.Code, err
			},
			get: func(mgr libraryManager, ctx context.Context, id string) (string, string, error) {
				p, err := mgr.GetHazard(ctx, id)
				return p.Code, p.Title, err
			},
			update: func(mgr libraryManager, ctx context.Context, id, t string) error {
				_, err := mgr.UpdateHazard(ctx, models.Hazard{ID: id, Title: t})
				return err
			},
			delete: func(mgr libraryManager, ctx context.Context, id string) error { return mgr.DeleteHazard(ctx, id) },
			count: func(mgr libraryManager, ctx context.Context, d string) (int, error) {
				l, err := mgr.ListHazards(ctx, d, mwanachamaagency.LibraryPartFilter{})
				return len(l), err
			},
		},
		{
			name: "threat", prefix: "T-", notFound: mwanachamaagency.ErrThreatNotFound, invalid: mwanachamaagency.ErrInvalidThreat,
			create: func(mgr libraryManager, ctx context.Context, d, t string) (string, string, error) {
				p, err := mgr.CreateThreat(ctx, models.Threat{DraftID: d, Title: t})
				return p.ID, p.Code, err
			},
			get: func(mgr libraryManager, ctx context.Context, id string) (string, string, error) {
				p, err := mgr.GetThreat(ctx, id)
				return p.Code, p.Title, err
			},
			update: func(mgr libraryManager, ctx context.Context, id, t string) error {
				_, err := mgr.UpdateThreat(ctx, models.Threat{ID: id, Title: t})
				return err
			},
			delete: func(mgr libraryManager, ctx context.Context, id string) error { return mgr.DeleteThreat(ctx, id) },
			count: func(mgr libraryManager, ctx context.Context, d string) (int, error) {
				l, err := mgr.ListThreats(ctx, d, mwanachamaagency.LibraryPartFilter{})
				return len(l), err
			},
		},
		{
			name: "consequence", prefix: "Q-", notFound: mwanachamaagency.ErrConsequenceNotFound, invalid: mwanachamaagency.ErrInvalidConsequence,
			create: func(mgr libraryManager, ctx context.Context, d, t string) (string, string, error) {
				p, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: d, Title: t})
				return p.ID, p.Code, err
			},
			get: func(mgr libraryManager, ctx context.Context, id string) (string, string, error) {
				p, err := mgr.GetConsequence(ctx, id)
				return p.Code, p.Title, err
			},
			update: func(mgr libraryManager, ctx context.Context, id, t string) error {
				_, err := mgr.UpdateConsequence(ctx, models.Consequence{ID: id, Title: t})
				return err
			},
			delete: func(mgr libraryManager, ctx context.Context, id string) error { return mgr.DeleteConsequence(ctx, id) },
			count: func(mgr libraryManager, ctx context.Context, d string) (int, error) {
				l, err := mgr.ListConsequences(ctx, d, mwanachamaagency.LibraryPartFilter{})
				return len(l), err
			},
		},
	}
}

func TestLibraryPart_MintsCodesKeepsThemAndRefusesBlankOrRepeatedTitles(t *testing.T) {
	for _, k := range libraryPartKinds() {
		t.Run(k.name, func(t *testing.T) {
			mgr := newLibraryManager(t)
			ctx := context.Background()
			_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

			id1, code1, err := k.create(mgr, ctx, o.DraftID, "Supplier fails")
			if err != nil || code1 != k.prefix+"1" {
				t.Fatalf("first create = %q, %v", code1, err)
			}
			if _, code2, err := k.create(mgr, ctx, o.DraftID, "Flood"); err != nil || code2 != k.prefix+"2" {
				t.Fatalf("second create = %q, %v", code2, err)
			}
			if _, _, err := k.create(mgr, ctx, o.DraftID, "  "); !errors.Is(err, k.invalid) {
				t.Fatalf("blank title err = %v, want %v", err, k.invalid)
			}
			if _, _, err := k.create(mgr, ctx, o.DraftID, " supplier FAILS "); !errors.Is(err, mwanachamaagency.ErrLibraryTitleTaken) {
				t.Fatalf("repeated title err = %v, want ErrLibraryTitleTaken", err)
			}
			if err := k.update(mgr, ctx, id1, "Flood"); !errors.Is(err, mwanachamaagency.ErrLibraryTitleTaken) {
				t.Fatalf("update onto a taken title err = %v", err)
			}
			if err := k.update(mgr, ctx, id1, "Supplier fails late"); err != nil {
				t.Fatalf("update: %v", err)
			}
			code, title, err := k.get(mgr, ctx, id1)
			if err != nil || code != code1 || title != "Supplier fails late" {
				t.Fatalf("get = %q %q %v", code, title, err)
			}
			if n, err := k.count(mgr, ctx, o.DraftID); err != nil || n != 2 {
				t.Fatalf("list = %d, %v", n, err)
			}
		})
	}
}

func TestLibraryPart_UnknownIDAnswersItsOwnNotFound(t *testing.T) {
	for _, k := range libraryPartKinds() {
		t.Run(k.name, func(t *testing.T) {
			mgr := newLibraryManager(t)
			ctx := context.Background()
			if _, _, err := k.get(mgr, ctx, "nope"); !errors.Is(err, k.notFound) {
				t.Fatalf("get err = %v", err)
			}
			if err := k.update(mgr, ctx, "nope", "x"); !errors.Is(err, k.notFound) {
				t.Fatalf("update err = %v", err)
			}
			if err := k.delete(mgr, ctx, "nope"); !errors.Is(err, k.notFound) {
				t.Fatalf("delete err = %v", err)
			}
		})
	}
}

func TestLibraryPart_DeleteRemovesAnUnlinkedPartAndFreesItsTitle(t *testing.T) {
	for _, k := range libraryPartKinds() {
		t.Run(k.name, func(t *testing.T) {
			mgr := newLibraryManager(t)
			ctx := context.Background()
			_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

			id, _, err := k.create(mgr, ctx, o.DraftID, "Fire")
			if err != nil {
				t.Fatal(err)
			}
			if err := k.delete(mgr, ctx, id); err != nil {
				t.Fatalf("delete: %v", err)
			}
			if _, _, err := k.get(mgr, ctx, id); !errors.Is(err, k.notFound) {
				t.Fatalf("get after delete err = %v", err)
			}
			if _, _, err := k.create(mgr, ctx, o.DraftID, "Fire"); err != nil {
				t.Fatalf("recreate: %v", err)
			}
		})
	}
}

func TestDeleteHazard_RefusedWhileARiskNamesIt(t *testing.T) {
	mgr := newLibraryManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	h, err := mgr.CreateHazard(ctx, models.Hazard{DraftID: o.DraftID, Title: "Chemicals"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := mgr.CreateRisk(ctx, models.Risk{DraftID: o.DraftID, Title: "Spill", TopEvent: "Leak", HazardCode: h.Code})
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.DeleteHazard(ctx, h.ID); !errors.Is(err, mwanachamaagency.ErrLibraryPartInUse) {
		t.Fatalf("delete while named err = %v", err)
	}
	r.HazardCode = ""
	if _, err := mgr.UpdateRisk(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := mgr.DeleteHazard(ctx, h.ID); err != nil {
		t.Fatalf("delete after unlink: %v", err)
	}
}

func TestDeleteThreatAndConsequence_TakeTheirControlAttachmentsWithThem(t *testing.T) {
	mgr := newLibraryManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	c, err := mgr.CreateControl(ctx, models.Control{DraftID: o.DraftID, Text: "Dual sourcing"})
	if err != nil {
		t.Fatal(err)
	}
	th, err := mgr.CreateThreat(ctx, models.Threat{DraftID: o.DraftID, Title: "Supplier fails"})
	if err != nil {
		t.Fatal(err)
	}
	cq, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, Title: "Stock out"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, c.ID, models.TargetTypeThreat, th.ID); err != nil {
		t.Fatalf("attach threat: %v", err)
	}
	if _, err := mgr.CreateControlAttachment(ctx, c.ID, models.TargetTypeConsequence, cq.ID); err != nil {
		t.Fatalf("attach consequence: %v", err)
	}
	if err := mgr.DeleteThreat(ctx, th.ID); err != nil {
		t.Fatalf("delete guarded threat: %v", err)
	}
	if err := mgr.DeleteConsequence(ctx, cq.ID); err != nil {
		t.Fatalf("delete guarded consequence: %v", err)
	}
	left, err := mgr.ListControlAttachments(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("attachments left = %d, want 0", len(left))
	}
}
