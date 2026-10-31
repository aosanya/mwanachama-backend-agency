package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type reviewDesk struct {
	t       *testing.T
	cs      *gosdkmcp.ClientSession
	mgr     mwanachamaagency.AgencyManager
	draftID string
}

func newReviewDesk(t *testing.T) *reviewDesk {
	t.Helper()
	mgr := newTestManager(t)
	mux := http.NewServeMux()
	server := newTestServer(mgr)
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	return &reviewDesk{t: t, cs: mcpConnect(t, srv.URL), mgr: mgr, draftID: draft.ID}
}

func (d *reviewDesk) call(name string, args map[string]any) (map[string]any, string, bool) {
	d.t.Helper()
	args["instance_id"] = "any"
	result, err := d.cs.CallTool(ctxT(d.t), &gosdkmcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		d.t.Fatalf("%s: %v", name, err)
	}
	out, _ := result.StructuredContent.(map[string]any)
	return out, textOf(result), result.IsError
}

func (d *reviewDesk) ok(name string, args map[string]any) map[string]any {
	d.t.Helper()
	out, text, failed := d.call(name, args)
	if failed {
		d.t.Fatalf("%s refused: %s", name, text)
	}
	return out
}

func (d *reviewDesk) start() map[string]any {
	d.t.Helper()
	return d.ok("agency_start_review", map[string]any{"draft_id": d.draftID, "started_by": "amina"})
}

func (d *reviewDesk) record(reviewID, area, status string, tokens int, findings ...map[string]any) map[string]any {
	d.t.Helper()
	items := make([]any, 0, len(findings))
	for _, f := range findings {
		items = append(items, f)
	}
	return d.ok("agency_record_review_area", map[string]any{
		"review_id": reviewID, "area": area, "status": status, "tokens": tokens,
		"summary": "Two sentences. Here.", "failure_reason": failureOf(status), "findings": items,
	})
}

func failureOf(status string) string {
	if status == "failed" {
		return "The reviewer ran out of allowance."
	}
	return ""
}

func (d *reviewDesk) finishAll(reviewID string, skipping ...string) {
	d.t.Helper()
	skip := map[string]bool{}
	for _, s := range skipping {
		skip[s] = true
	}
	for _, area := range models.ReviewAreaOrder {
		if !skip[string(area)] {
			d.record(reviewID, string(area), "done", 10)
		}
	}
}

func reviewOf(out map[string]any) map[string]any {
	review, _ := out["review"].(map[string]any)
	return review
}

func areasOf(out map[string]any) map[string]map[string]any {
	byName := map[string]map[string]any{}
	areas, _ := out["areas"].([]any)
	for _, a := range areas {
		area, _ := a.(map[string]any)
		byName[area["area"].(string)] = area
	}
	return byName
}

func findingsOf(area map[string]any) []map[string]any {
	var out []map[string]any
	items, _ := area["findings"].([]any)
	for _, f := range items {
		finding, _ := f.(map[string]any)
		out = append(out, finding)
	}
	return out
}

func finding(title string) map[string]any {
	return map[string]any{
		"title": title, "detail": "Because.", "tool": "agency_update_goal",
		"arguments": map[string]any{"goal_id": "g1", "description": "Sell maize"},
		"then":      []any{map[string]any{"tool": "agency_set_measure", "arguments": map[string]any{"name": "Shops"}}},
	}
}

func carried(from, fate string, extra map[string]any) map[string]any {
	f := finding("Carried " + from)
	f["carried_from_id"], f["fate"], f["fate_reason"] = from, fate, "Reason."
	for k, v := range extra {
		f[k] = v
	}
	return f
}

func TestReviewTools_StartWritesOneReadingPartPerReviewerAndRefusesASecondRun(t *testing.T) {
	d := newReviewDesk(t)

	started := d.start()
	review := reviewOf(started)
	if review["status"] != "running" || review["draft_id"] != d.draftID || review["started_by"] != "amina" || review["previous_review_id"] != "" {
		t.Fatalf("review = %+v", review)
	}
	areas := areasOf(started)
	for _, name := range []string{"profile", "goals", "objectives", "roles_agents", "workflows", "risks", "tests"} {
		if areas[name] == nil || areas[name]["status"] != "reading" {
			t.Fatalf("area %s = %+v, want a reading part", name, areas[name])
		}
	}
	if len(areas) != 7 {
		t.Fatalf("areas = %d, want 7", len(areas))
	}

	_, text, failed := d.call("agency_start_review", map[string]any{"draft_id": d.draftID})
	if !failed || text != "A review of this draft is already running." {
		t.Fatalf("second start = failed %v %q, want the running sentence", failed, text)
	}
}

func TestReviewTools_GetAnswersNullReviewWhenTheDraftHasNone(t *testing.T) {
	d := newReviewDesk(t)

	out := d.ok("agency_get_review", map[string]any{"draft_id": d.draftID})
	if v, present := out["review"]; !present || v != nil {
		t.Fatalf("get = %+v, want {review: null}", out)
	}
	if _, text, failed := d.call("agency_get_review", map[string]any{}); !failed {
		t.Fatalf("get with nothing named = %q, want a refusal", text)
	}
	if _, _, failed := d.call("agency_get_review", map[string]any{"review_id": "no-such-review"}); !failed {
		t.Fatal("get of an unknown review succeeded")
	}
}

func TestReviewTools_RecordingDoneAndFailedPartsFinishesTheReviewAndSumsTokens(t *testing.T) {
	d := newReviewDesk(t)
	id := reviewOf(d.start())["id"].(string)

	for i, area := range []string{"profile", "goals", "objectives", "roles_agents", "workflows", "risks"} {
		out := d.record(id, area, "done", 100)
		if out["review_status"] != "running" {
			t.Fatalf("after part %d review_status = %v, want running", i+1, out["review_status"])
		}
	}
	mid := d.ok("agency_get_review", map[string]any{"review_id": id})
	if reviewOf(mid)["finished_at"] != "" || reviewOf(mid)["tokens"].(float64) != 600 {
		t.Fatalf("mid-run review = %+v", reviewOf(mid))
	}

	last := d.record(id, "tests", "failed", 7)
	if last["review_status"] != "done" {
		t.Fatalf("review_status = %v, want done", last["review_status"])
	}
	area, _ := last["area"].(map[string]any)
	if area["status"] != "failed" || area["failure_reason"] != "The reviewer ran out of allowance." || area["tokens"].(float64) != 7 {
		t.Fatalf("area = %+v", area)
	}

	done := d.ok("agency_get_review", map[string]any{"draft_id": d.draftID})
	review := reviewOf(done)
	if review["status"] != "done" || review["finished_at"] == "" || review["tokens"].(float64) != 607 {
		t.Fatalf("finished review = %+v, want done with finished_at and 607 tokens", review)
	}
	if areasOf(done)["goals"]["summary"] != "Two sentences. Here." {
		t.Fatalf("goals area = %+v", areasOf(done)["goals"])
	}

	if _, text, failed := d.call("agency_record_review_area", map[string]any{"review_id": id, "area": "goals", "status": "done"}); !failed {
		t.Fatalf("recording a part twice = %q, want a refusal", text)
	}
}

func TestReviewTools_RecordRefusesWhatCannotBeRecorded(t *testing.T) {
	d := newReviewDesk(t)
	id := reviewOf(d.start())["id"].(string)

	cases := map[string]map[string]any{
		"unknown review":    {"review_id": "nope", "area": "goals", "status": "done"},
		"unknown part":      {"review_id": id, "area": "budget", "status": "done"},
		"status reading":    {"review_id": id, "area": "goals", "status": "reading"},
		"negative tokens":   {"review_id": id, "area": "goals", "status": "done", "tokens": -1},
		"untitled finding":  {"review_id": id, "area": "goals", "status": "done", "findings": []any{map[string]any{"detail": "x"}}},
		"carry without run": {"review_id": id, "area": "goals", "status": "done", "findings": []any{carried("nope", "still_open", nil)}},
	}
	for name, args := range cases {
		if _, text, failed := d.call("agency_record_review_area", args); !failed {
			t.Errorf("%s: recorded, want a refusal (%q)", name, text)
		}
	}
	if areasOf(d.ok("agency_get_review", map[string]any{"review_id": id}))["goals"]["status"] != "reading" {
		t.Fatal("a refused record changed the part")
	}
}

func TestReviewTools_CarryOverKeepsWhatTheDesignerDid(t *testing.T) {
	d := newReviewDesk(t)
	first := reviewOf(d.start())["id"].(string)
	d.record(first, "goals", "done", 1,
		finding("applied and gone"), finding("applied and back"), finding("dismissed and quiet"),
		finding("dismissed and changed"), finding("still open"), finding("withdrawn after applied"))
	d.finishAll(first, "goals")

	earlier := map[string]string{}
	for _, f := range findingsOf(areasOf(d.ok("agency_get_review", map[string]any{"review_id": first}))["goals"]) {
		earlier[f["title"].(string)] = f["id"].(string)
	}
	set := func(title, status string) {
		d.ok("agency_set_finding_status", map[string]any{"finding_id": earlier[title], "status": status})
	}
	set("applied and gone", "applied")
	set("applied and back", "applied")
	set("dismissed and quiet", "dismissed")
	set("dismissed and changed", "dismissed")
	set("withdrawn after applied", "applied")

	second := reviewOf(d.start())["id"].(string)
	if got := reviewOf(d.ok("agency_get_review", map[string]any{"review_id": second}))["previous_review_id"]; got != first {
		t.Fatalf("previous_review_id = %v, want %s", got, first)
	}
	d.record(second, "goals", "done", 1,
		carried(earlier["applied and gone"], "fixed", nil),
		carried(earlier["applied and back"], "still_open", nil),
		carried(earlier["dismissed and quiet"], "still_open", nil),
		carried(earlier["dismissed and changed"], "still_open", map[string]any{"reopen": true}),
		carried(earlier["still open"], "still_open", nil),
		carried(earlier["withdrawn after applied"], "withdrawn", nil),
		finding("brand new"),
	)

	want := map[string]string{
		"Carried " + earlier["applied and gone"]:        "applied",
		"Carried " + earlier["applied and back"]:        "open",
		"Carried " + earlier["dismissed and quiet"]:     "dismissed",
		"Carried " + earlier["dismissed and changed"]:   "open",
		"Carried " + earlier["still open"]:              "open",
		"Carried " + earlier["withdrawn after applied"]: "applied",
		"brand new": "open",
	}
	got := map[string]string{}
	var brandNew map[string]any
	for _, f := range findingsOf(areasOf(d.ok("agency_get_review", map[string]any{"review_id": second}))["goals"]) {
		got[f["title"].(string)] = f["status"].(string)
		if f["title"] == "brand new" {
			brandNew = f
		}
	}
	for title, status := range want {
		if got[title] != status {
			t.Errorf("%s = %q, want %q", title, got[title], status)
		}
	}
	if brandNew["fate"] != "new" || brandNew["carried_from_id"] != "" {
		t.Fatalf("brand new finding = %+v", brandNew)
	}
}

func TestReviewTools_ACarriedFindingMustAnswerTheEarlierReview(t *testing.T) {
	d := newReviewDesk(t)
	first := reviewOf(d.start())["id"].(string)
	d.record(first, "goals", "done", 1, finding("one"))
	d.finishAll(first, "goals")
	kept := findingsOf(areasOf(d.ok("agency_get_review", map[string]any{"review_id": first}))["goals"])[0]["id"].(string)

	second := reviewOf(d.start())["id"].(string)
	if _, text, failed := d.call("agency_record_review_area", map[string]any{
		"review_id": second, "area": "goals", "status": "done", "findings": []any{carried(kept, "new", nil)},
	}); !failed {
		t.Fatalf("a carried finding with fate new was recorded (%q)", text)
	}
	d.finishAll(second)

	third := reviewOf(d.start())["id"].(string)
	if _, text, failed := d.call("agency_record_review_area", map[string]any{
		"review_id": third, "area": "goals", "status": "done", "findings": []any{carried(kept, "still_open", nil)},
	}); !failed {
		t.Fatalf("a finding from two runs back was carried (%q)", text)
	}
}

func TestReviewTools_SetFindingStatusAndComment(t *testing.T) {
	d := newReviewDesk(t)
	id := reviewOf(d.start())["id"].(string)
	d.record(id, "risks", "done", 1, finding("Name the risk owner"))
	fid := findingsOf(areasOf(d.ok("agency_get_review", map[string]any{"review_id": id}))["risks"])[0]["id"].(string)

	set := d.ok("agency_set_finding_status", map[string]any{"finding_id": fid, "status": "dismissed"})
	if set["status"] != "dismissed" || set["id"] != fid || set["title"] != "Name the risk owner" {
		t.Fatalf("set = %+v", set)
	}
	if _, _, failed := d.call("agency_set_finding_status", map[string]any{"finding_id": fid, "status": "done"}); !failed {
		t.Fatal("a status outside open, applied, dismissed was accepted")
	}
	if _, _, failed := d.call("agency_set_finding_status", map[string]any{"finding_id": "nope", "status": "open"}); !failed {
		t.Fatal("an unknown finding was accepted")
	}

	note := d.ok("agency_comment_finding", map[string]any{"finding_id": fid, "body": "  The chair owns it.  ", "author": "amina"})
	if note["body"] != "The chair owns it." || note["author"] != "amina" || note["finding_id"] != fid || note["created_at"] == "" {
		t.Fatalf("comment = %+v", note)
	}
	if _, _, failed := d.call("agency_comment_finding", map[string]any{"finding_id": fid, "body": "   "}); !failed {
		t.Fatal("an empty comment was accepted")
	}
	if _, _, failed := d.call("agency_comment_finding", map[string]any{"finding_id": "nope", "body": "x"}); !failed {
		t.Fatal("a comment on an unknown finding was accepted")
	}

	read := findingsOf(areasOf(d.ok("agency_get_review", map[string]any{"review_id": id}))["risks"])[0]
	comments, _ := read["comments"].([]any)
	if read["status"] != "dismissed" || len(comments) != 1 {
		t.Fatalf("finding = %+v, want dismissed with one comment", read)
	}
	arguments, _ := read["arguments"].(map[string]any)
	steps, _ := read["then"].([]any)
	if arguments["goal_id"] != "g1" || len(steps) != 1 || steps[0].(map[string]any)["tool"] != "agency_set_measure" {
		t.Fatalf("finding lost its arguments or then: %+v", read)
	}
}

func TestReviewTools_ListIsNewestFirstWithoutFindings(t *testing.T) {
	d := newReviewDesk(t)
	first := reviewOf(d.start())["id"].(string)
	d.record(first, "goals", "done", 1, finding("one"))
	d.finishAll(first, "goals")
	second := reviewOf(d.start())["id"].(string)

	out := d.ok("agency_list_reviews", map[string]any{"draft_id": d.draftID})
	items, _ := out["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["id"] != second || items[1].(map[string]any)["id"] != first {
		t.Fatalf("items = %+v, want the second review then the first", items)
	}
	for _, item := range items {
		if _, has := item.(map[string]any)["areas"]; has {
			t.Fatalf("a listed review carries its parts: %+v", item)
		}
	}
	if _, _, failed := d.call("agency_list_reviews", map[string]any{}); !failed {
		t.Fatal("list without a draft succeeded")
	}
}

func TestReviewTools_ReviewInputHoldsTheEarlierFindingsOfThatPartOnly(t *testing.T) {
	d := newReviewDesk(t)
	first := reviewOf(d.start())["id"].(string)
	if prior := d.ok("agency_review_input", map[string]any{"review_id": first, "area": "goals"}); len(prior["previous"].([]any)) != 0 {
		t.Fatalf("first run input = %+v, want none", prior)
	}
	d.record(first, "goals", "done", 1, finding("goal one"), finding("goal two"))
	d.record(first, "workflows", "done", 1, finding("workflow one"))
	d.finishAll(first, "goals", "workflows")
	goal := findingsOf(areasOf(d.ok("agency_get_review", map[string]any{"review_id": first}))["goals"])[0]
	d.ok("agency_comment_finding", map[string]any{"finding_id": goal["id"], "body": "Already decided."})

	second := reviewOf(d.start())["id"].(string)
	input := d.ok("agency_review_input", map[string]any{"review_id": second, "area": "goals"})
	previous, _ := input["previous"].([]any)
	if len(previous) != 2 {
		t.Fatalf("previous = %+v, want the two goals findings", previous)
	}
	commented := 0
	for _, p := range previous {
		row := p.(map[string]any)
		if row["review_id"] != first {
			t.Fatalf("finding %+v is not from the first review", row)
		}
		if len(row["comments"].([]any)) == 1 {
			commented++
		}
	}
	if commented != 1 {
		t.Fatalf("commented findings = %d, want 1", commented)
	}
	if _, _, failed := d.call("agency_review_input", map[string]any{"review_id": second, "area": "budget"}); !failed {
		t.Fatal("an unknown part was accepted")
	}
	if _, _, failed := d.call("agency_review_input", map[string]any{"review_id": "nope", "area": "goals"}); !failed {
		t.Fatal("an unknown review was accepted")
	}
}

func TestReviewTools_StartAgainOnceTheRunHasFinished(t *testing.T) {
	d := newReviewDesk(t)
	first := reviewOf(d.start())["id"].(string)
	d.finishAll(first)
	second := reviewOf(d.start())
	if second["previous_review_id"] != first || second["status"] != "running" {
		t.Fatalf("second review = %+v", second)
	}
	if _, _, failed := d.call("agency_start_review", map[string]any{"draft_id": "no-such-draft"}); !failed {
		t.Fatal("a review of an unknown draft started")
	}
}

func TestReviewTools_StartTakesOneAreaAndReadsOnlyThatPart(t *testing.T) {
	d := newReviewDesk(t)

	out := d.ok("agency_start_review", map[string]any{"draft_id": d.draftID, "started_by": "amina", "area": "risks"})
	areas, _ := out["areas"].([]any)
	if len(areas) != 1 {
		t.Fatalf("areas = %+v, want one", out["areas"])
	}
	if first, _ := areas[0].(map[string]any); first["area"] != "risks" {
		t.Fatalf("area = %+v, want risks", first)
	}
}
