package mwanachamaagency_test

import (
	"context"
	"slices"
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
