package mwanachamaagency_test

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type taggedRows struct {
	order routedOrder
	a     models.RecordRow
	ab    models.RecordRow
	none  models.RecordRow
}

func tagged(t *testing.T) taggedRows {
	t.Helper()
	o := buildRoutedOrder(t)
	ctx := context.Background()
	add := func(amount string, tags []string) models.RecordRow {
		in := paymentRow(o, "", "Wanjiru", amount)
		in.Tags = tags
		made, err := o.world.mgr.CreateRecordRow(ctx, in)
		if err != nil {
			t.Fatalf("CreateRecordRow %s: %v", amount, err)
		}
		return made
	}
	return taggedRows{order: o, a: add("100", []string{"a"}), ab: add("200", []string{"a", "b"}), none: add("300", nil)}
}

func idsOf(rows []models.RecordRow) []string {
	out := []string{}
	for _, r := range rows {
		out = append(out, r.ID)
	}
	slices.Sort(out)
	return out
}

func sortedIDs(rows ...models.RecordRow) []string { return idsOf(rows) }

func (g taggedRows) listed(t *testing.T, tags ...string) []models.RecordRow {
	t.Helper()
	rows, err := g.order.world.mgr.ListRecordRows(context.Background(), g.order.world.draftID, mwanachamaagency.RecordRowFilter{RecordID: g.order.record.ID, Tags: tags})
	if err != nil {
		t.Fatalf("ListRecordRows %v: %v", tags, err)
	}
	return rows
}

func TestRecordRowTags_ARowKeepsTheTagsItWasGiven(t *testing.T) {
	g := tagged(t)
	if !slices.Equal(g.a.Tags, []string{"a"}) || !slices.Equal(g.ab.Tags, []string{"a", "b"}) {
		t.Errorf("tags of the made rows = %v and %v", g.a.Tags, g.ab.Tags)
	}
	for _, r := range g.listed(t) {
		if r.ID == g.none.ID && (r.Tags == nil || len(r.Tags) != 0) {
			t.Errorf("a row with no tags lists as %#v, want an empty list", r.Tags)
		}
	}
	if g.none.Tags == nil || len(g.none.Tags) != 0 {
		t.Errorf("a row made with no tags is %#v, want an empty list", g.none.Tags)
	}
}

func TestRecordRowTags_TheListKeepsTheRowsCarryingAnyOfTheGivenTags(t *testing.T) {
	g := tagged(t)
	cases := []struct {
		name string
		tags []string
		want []string
	}{
		{"one tag", []string{"a"}, sortedIDs(g.a, g.ab)},
		{"another tag", []string{"b"}, sortedIDs(g.ab)},
		{"a tag no row carries", []string{"c"}, []string{}},
		{"any of two tags", []string{"b", "c"}, sortedIDs(g.ab)},
		{"two tags both carried", []string{"a", "b"}, sortedIDs(g.a, g.ab)},
		{"no filter", nil, sortedIDs(g.a, g.ab, g.none)},
		{"the all tag reaches every row", []string{"all"}, sortedIDs(g.a, g.ab, g.none)},
		{"the all tag among others reaches every row", []string{"c", "all"}, sortedIDs(g.a, g.ab, g.none)},
	}
	for _, c := range cases {
		if got := idsOf(g.listed(t, c.tags...)); !slices.Equal(got, c.want) {
			t.Errorf("%s: listed %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRecordRowTags_ATagIsMatchedWholeAndExactly(t *testing.T) {
	o := buildRoutedOrder(t)
	ctx := context.Background()
	in := paymentRow(o, "", "Wanjiru", "100")
	in.Tags = []string{"north-east", `say "hi"`}
	made, err := o.world.mgr.CreateRecordRow(ctx, in)
	if err != nil {
		t.Fatalf("CreateRecordRow: %v", err)
	}
	for tag, want := range map[string]int{"north": 0, "east": 0, "north-east": 1, `say "hi"`: 1, "NORTH-EAST": 0, "%": 0} {
		rows, err := o.world.mgr.ListRecordRows(ctx, o.world.draftID, mwanachamaagency.RecordRowFilter{RecordID: o.record.ID, Tags: []string{tag}})
		if err != nil || len(rows) != want || (want == 1 && rows[0].ID != made.ID) {
			t.Errorf("tag %q listed %d rows, want %d, %v", tag, len(rows), want, err)
		}
	}
}

func TestRecordRowTags_ABlankTagIsRefused(t *testing.T) {
	o := buildRoutedOrder(t)
	in := paymentRow(o, "", "Wanjiru", "100")
	in.Tags = []string{"a", " "}
	if _, err := o.world.mgr.CreateRecordRow(context.Background(), in); err == nil {
		t.Fatal("a row with a blank tag was made")
	}
}

func TestRecordRowTags_AddingATagKeepsTheOnesTheRowHas(t *testing.T) {
	g := tagged(t)
	ctx := context.Background()
	got, err := g.order.world.mgr.AddRecordRowTag(ctx, g.a.ID, "b")
	if err != nil || !slices.Equal(got.Tags, []string{"a", "b"}) {
		t.Fatalf("AddRecordRowTag b = %v, %v", got.Tags, err)
	}
	again, err := g.order.world.mgr.AddRecordRowTag(ctx, g.a.ID, "b")
	if err != nil || !slices.Equal(again.Tags, []string{"a", "b"}) {
		t.Fatalf("AddRecordRowTag b twice = %v, %v", again.Tags, err)
	}
	if got := idsOf(g.listed(t, "b")); !slices.Equal(got, sortedIDs(g.a, g.ab)) {
		t.Errorf("rows carrying b = %v, want both rows that carry it now", got)
	}
}

func TestRecordRowTags_ARowWithNoTagsTakesItsFirstOne(t *testing.T) {
	g := tagged(t)
	got, err := g.order.world.mgr.AddRecordRowTag(context.Background(), g.none.ID, "c")
	if err != nil || !slices.Equal(got.Tags, []string{"c"}) {
		t.Fatalf("AddRecordRowTag c = %v, %v", got.Tags, err)
	}
}

func TestRecordRowTags_AddingARefusedTagChangesNothing(t *testing.T) {
	g := tagged(t)
	ctx := context.Background()
	mgr := g.order.world.mgr
	if _, err := mgr.AddRecordRowTag(ctx, g.a.ID, " "); err == nil {
		t.Error("a blank tag was added")
	}
	if _, err := mgr.AddRecordRowTag(ctx, "no-such-row", "b"); err == nil {
		t.Error("a tag was added to a row that does not exist")
	}
	if _, err := mgr.RetireRecordRow(ctx, g.ab.ID, "member-1", "entered twice"); err != nil {
		t.Fatalf("RetireRecordRow: %v", err)
	}
	if _, err := mgr.AddRecordRowTag(ctx, g.ab.ID, "c"); err == nil {
		t.Error("a tag was added to a retired row")
	}
	fix := paymentRow(g.order, "", "Wanjiru", "150")
	fix.Replaces = g.a.ID
	if _, err := mgr.CreateRecordRow(ctx, fix); err != nil {
		t.Fatalf("replacing row: %v", err)
	}
	if _, err := mgr.AddRecordRowTag(ctx, g.a.ID, "c"); err == nil {
		t.Error("a tag was added to a row a correction replaced")
	}
	if got := idsOf(g.listed(t, "c")); len(got) != 0 {
		t.Errorf("rows carrying c = %v, want none", got)
	}
}

func TestRecordRowTags_NothingTakesATagOffARow(t *testing.T) {
	var tagMethods []string
	managerType := reflect.TypeOf((*mwanachamaagency.AgencyManager)(nil)).Elem()
	for i := 0; i < managerType.NumMethod(); i++ {
		if name := managerType.Method(i).Name; strings.Contains(name, "RecordRowTag") {
			tagMethods = append(tagMethods, name)
		}
	}
	if !slices.Equal(tagMethods, []string{"AddRecordRowTag"}) {
		t.Errorf("manager methods on a row's tags = %v, want only AddRecordRowTag", tagMethods)
	}
}

func TestRecordRowTags_AReplacingRowCarriesTheTagsOfTheRowItReplaces(t *testing.T) {
	g := tagged(t)
	ctx := context.Background()
	mgr := g.order.world.mgr
	if _, err := mgr.AddRecordRowTag(ctx, g.a.ID, "b"); err != nil {
		t.Fatalf("AddRecordRowTag: %v", err)
	}
	fix := paymentRow(g.order, "", "Wanjiru", "150")
	fix.Replaces = g.a.ID
	made, err := mgr.CreateRecordRow(ctx, fix)
	if err != nil || !slices.Equal(made.Tags, []string{"a", "b"}) {
		t.Fatalf("replacing row = %v, %v", made.Tags, err)
	}
	listed, err := mgr.ListRecordRows(ctx, g.order.world.draftID, mwanachamaagency.RecordRowFilter{RecordID: g.order.record.ID, Tags: []string{"b"}})
	if err != nil || !slices.Contains(idsOf(listed), made.ID) {
		t.Errorf("the replacing row is not listed under b: %v, %v", idsOf(listed), err)
	}
}

func TestRecordRowTags_AReplacingRowCannotChangeTheTags(t *testing.T) {
	g := tagged(t)
	ctx := context.Background()
	mgr := g.order.world.mgr
	fix := paymentRow(g.order, "", "Wanjiru", "150")
	fix.Replaces = g.ab.ID
	fix.Tags = []string{"z"}
	made, err := mgr.CreateRecordRow(ctx, fix)
	if err != nil || !slices.Equal(made.Tags, []string{"a", "b"}) {
		t.Fatalf("replacing row that names z = %v, %v", made.Tags, err)
	}
	fix = paymentRow(g.order, "", "Wanjiru", "400")
	fix.Replaces = g.none.ID
	fix.Tags = []string{"z"}
	made, err = mgr.CreateRecordRow(ctx, fix)
	if err != nil || made.Tags == nil || len(made.Tags) != 0 {
		t.Fatalf("replacing a row with no tags = %#v, %v", made.Tags, err)
	}
}
