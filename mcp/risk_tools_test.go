package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

func TestRiskTools_CreateARiskThenItsArmsAndListThemWithCodeAndId(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}

	created, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_risk",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draft.ID, "title": "Unsold harvest", "top_event": "A harvest goes unsold", "owner": "Farmers' agent"},
	})
	if err != nil || created.IsError {
		t.Fatalf("agency_create_risk: %v %q", err, textOf(created))
	}
	risk, _ := created.StructuredContent.(map[string]any)
	riskID, _ := risk["id"].(string)
	if riskID == "" || risk["code"] != "RK-1" || !strings.Contains(textOf(created), "RK-1") {
		t.Fatalf("created risk = %+v, text %q", risk, textOf(created))
	}

	for _, arm := range []map[string]any{
		{"side": "threat", "title": "No buyer is listed"},
		{"side": "consequence", "title": "Produce spoils"},
	} {
		args := map[string]any{"instance_id": "any", "draft_id": draft.ID, "risk_id": riskID}
		for k, v := range arm {
			args[k] = v
		}
		result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{Name: "agency_create_risk_pathway", Arguments: args})
		if err != nil || result.IsError {
			t.Fatalf("agency_create_risk_pathway %v: %v %q", arm, err, textOf(result))
		}
	}

	listed, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_risk_pathways",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draft.ID, "risk_id": riskID, "side": "threat"},
	})
	if err != nil || listed.IsError {
		t.Fatalf("agency_list_risk_pathways: %v %q", err, textOf(listed))
	}
	structured, _ := listed.StructuredContent.(map[string]any)
	items, _ := structured["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %+v, want the one threat", structured)
	}
	item, _ := items[0].(map[string]any)
	if !strings.Contains(textOf(listed), "RP-1") || !strings.Contains(textOf(listed), item["id"].(string)) {
		t.Fatalf("list text = %q, want the pathway's code and id", textOf(listed))
	}

	risks, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_list_risks",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draft.ID, "title": "harvest"},
	})
	if err != nil || risks.IsError || !strings.Contains(textOf(risks), "RK-1") || !strings.Contains(textOf(risks), riskID) {
		t.Fatalf("agency_list_risks = %v %q", err, textOf(risks))
	}
}

func TestRiskTools_APathwayOnAnUnknownRiskIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	server := newTestServer(mgr)
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	if _, err := mgr.CreateAgency(ctxT(t), models.Agency{Name: "Farmers"}); err != nil {
		t.Fatalf("CreateAgency: %v", err)
	}
	draft, err := mgr.CreateDraft(ctxT(t), "seed", "", "")
	if err != nil {
		t.Fatalf("CreateDraft: %v", err)
	}
	result, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Name:      "agency_create_risk_pathway",
		Arguments: map[string]any{"instance_id": "any", "draft_id": draft.ID, "risk_id": "no-such-risk", "side": "threat", "title": "x"},
	})
	if err != nil || !result.IsError || !strings.Contains(textOf(result), "risk not found") {
		t.Fatalf("agency_create_risk_pathway on an unknown risk = %v %q", err, textOf(result))
	}
}
