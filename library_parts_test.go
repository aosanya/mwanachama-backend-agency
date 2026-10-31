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
	ListConsequences(ctx context.Context, draftID string, filter mwanachamaagency.ConsequenceFilter) ([]models.Consequence, error)
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

func sharedHazard(mgr libraryManager, ctx context.Context, draftID string) string {
	existing, err := mgr.ListHazards(ctx, draftID, mwanachamaagency.LibraryPartFilter{})
	if err == nil && len(existing) > 0 {
		return existing[0].Code
	}
	h, err := mgr.CreateHazard(ctx, models.Hazard{DraftID: draftID, Title: "Shared hazard"})
	if err != nil {
		return ""
	}
	return h.Code
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
				h := sharedHazard(mgr, ctx, d)
				p, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: d, Title: t, HazardCode: h})
				return p.ID, p.Code, err
			},
			get: func(mgr libraryManager, ctx context.Context, id string) (string, string, error) {
				p, err := mgr.GetConsequence(ctx, id)
				return p.Code, p.Title, err
			},
			update: func(mgr libraryManager, ctx context.Context, id, t string) error {
				existing, err := mgr.GetConsequence(ctx, id)
				if err != nil {
					return err
				}
				existing.Title = t
				_, err = mgr.UpdateConsequence(ctx, existing)
				return err
			},
			delete: func(mgr libraryManager, ctx context.Context, id string) error { return mgr.DeleteConsequence(ctx, id) },
			count: func(mgr libraryManager, ctx context.Context, d string) (int, error) {
				l, err := mgr.ListConsequences(ctx, d, mwanachamaagency.ConsequenceFilter{})
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
	hz, err := mgr.CreateHazard(ctx, models.Hazard{DraftID: o.DraftID, Title: "Supply"})
	if err != nil {
		t.Fatal(err)
	}
	cq, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, Title: "Stock out", HazardCode: hz.Code})
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

func newHazard(t *testing.T, mgr libraryManager, ctx context.Context, draftID, title string) models.Hazard {
	t.Helper()
	h, err := mgr.CreateHazard(ctx, models.Hazard{DraftID: draftID, Title: title})
	if err != nil {
		t.Fatalf("CreateHazard %q: %v", title, err)
	}
	return h
}

func TestCreateConsequence_RefusesAnEmptyOrUnknownHazardCode(t *testing.T) {
	mgr := newLibraryManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)

	if _, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, Title: "Stock out"}); !errors.Is(err, mwanachamaagency.ErrInvalidConsequence) {
		t.Fatalf("empty hazard_code err = %v, want ErrInvalidConsequence", err)
	}
	if _, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, Title: "Stock out", HazardCode: "H-99"}); !errors.Is(err, mwanachamaagency.ErrHazardNotFound) {
		t.Fatalf("unknown hazard_code err = %v, want ErrHazardNotFound", err)
	}
}

func TestCreateConsequence_SameTitleUnderTwoHazardsIsAllowedUnderOneIsTaken(t *testing.T) {
	mgr := newLibraryManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	fire := newHazard(t, mgr, ctx, o.DraftID, "Fire")
	flood := newHazard(t, mgr, ctx, o.DraftID, "Flood")

	if _, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, Title: "Site closed", HazardCode: fire.Code}); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, Title: "site CLOSED", HazardCode: flood.Code}); err != nil {
		t.Fatalf("same title under another hazard: %v", err)
	}
	if _, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, Title: "Site closed", HazardCode: fire.Code}); !errors.Is(err, mwanachamaagency.ErrLibraryTitleTaken) {
		t.Fatalf("same title under one hazard err = %v, want ErrLibraryTitleTaken", err)
	}
}

func TestUpdateConsequence_MovesToAnotherHazardAndRefusesEmptyOrUnknown(t *testing.T) {
	mgr := newLibraryManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	fire := newHazard(t, mgr, ctx, o.DraftID, "Fire")
	flood := newHazard(t, mgr, ctx, o.DraftID, "Flood")
	cq, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, Title: "Site closed", HazardCode: fire.Code})
	if err != nil {
		t.Fatal(err)
	}
	other, err := mgr.CreateConsequence(ctx, models.Consequence{DraftID: o.DraftID, Title: "Site closed", HazardCode: flood.Code})
	if err != nil {
		t.Fatal(err)
	}

	cq.HazardCode = flood.Code
	if _, err := mgr.UpdateConsequence(ctx, cq); !errors.Is(err, mwanachamaagency.ErrLibraryTitleTaken) {
		t.Fatalf("move onto a hazard holding the title err = %v, want ErrLibraryTitleTaken", err)
	}
	cq.Title = "Site shut"
	moved, err := mgr.UpdateConsequence(ctx, cq)
	if err != nil || moved.HazardCode != flood.Code {
		t.Fatalf("move = %+v, %v", moved, err)
	}
	got, err := mgr.GetConsequence(ctx, cq.ID)
	if err != nil || got.HazardCode != flood.Code {
		t.Fatalf("get after move = %+v, %v", got, err)
	}

	cq.HazardCode = ""
	if _, err := mgr.UpdateConsequence(ctx, cq); !errors.Is(err, mwanachamaagency.ErrInvalidConsequence) {
		t.Fatalf("empty hazard_code err = %v, want ErrInvalidConsequence", err)
	}
	cq.HazardCode = "H-99"
	if _, err := mgr.UpdateConsequence(ctx, cq); !errors.Is(err, mwanachamaagency.ErrHazardNotFound) {
		t.Fatalf("unknown hazard_code err = %v, want ErrHazardNotFound", err)
	}
	if other.HazardCode != flood.Code {
		t.Fatalf("other consequence changed: %+v", other)
	}
}

func TestListConsequences_FiltersByHazardCode(t *testing.T) {
	mgr := newLibraryManager(t)
	ctx := context.Background()
	_, _, o, _, _ := seedLiveAgency(t, mgr, ctx)
	fire := newHazard(t, mgr, ctx, o.DraftID, "Fire")
	flood := newHazard(t, mgr, ctx, o.DraftID, "Flood")
	for _, c := range []models.Consequence{
		{DraftID: o.DraftID, Title: "Site closed", HazardCode: fire.Code},
		{DraftID: o.DraftID, Title: "Injury", HazardCode: fire.Code},
		{DraftID: o.DraftID, Title: "Stock lost", HazardCode: flood.Code},
	} {
		if _, err := mgr.CreateConsequence(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	onFire, err := mgr.ListConsequences(ctx, o.DraftID, mwanachamaagency.ConsequenceFilter{HazardCode: fire.Code})
	if err != nil || len(onFire) != 2 {
		t.Fatalf("fire list = %+v, %v", onFire, err)
	}
	onFlood, err := mgr.ListConsequences(ctx, o.DraftID, mwanachamaagency.ConsequenceFilter{HazardCode: flood.Code})
	if err != nil || len(onFlood) != 1 || onFlood[0].Title != "Stock lost" {
		t.Fatalf("flood list = %+v, %v", onFlood, err)
	}
	all, err := mgr.ListConsequences(ctx, o.DraftID, mwanachamaagency.ConsequenceFilter{})
	if err != nil || len(all) != 3 {
		t.Fatalf("unfiltered list = %d, %v", len(all), err)
	}
}
