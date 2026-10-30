package mcp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	agencymcp "github.com/aosanya/mwanachama-backend-agency/mcp"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

type conversationRecorder struct {
	mwanachamaagency.AgencyManager
	mu   sync.Mutex
	seen []string
}

func (r *conversationRecorder) GetAgency(ctx context.Context) (models.Agency, error) {
	r.mu.Lock()
	r.seen = append(r.seen, dispatch.ConversationOf(ctx))
	r.mu.Unlock()
	return r.AgencyManager.GetAgency(ctx)
}

func conversationsSeenBy(t *testing.T, meta gosdkmcp.Meta) []string {
	t.Helper()
	recorder := &conversationRecorder{AgencyManager: newTestManager(t)}
	server := gosdkmcp.NewServer(&gosdkmcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	agencymcp.RegisterTools(server, func(context.Context, string) (mwanachamaagency.AgencyManager, error) {
		return recorder, nil
	})
	mux := http.NewServeMux()
	mux.Handle("/mcp", gosdkmcp.NewStreamableHTTPHandler(func(*http.Request) *gosdkmcp.Server { return server }, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cs := mcpConnect(t, srv.URL)

	if _, err := cs.CallTool(ctxT(t), &gosdkmcp.CallToolParams{
		Meta:      meta,
		Name:      "agency_get_agency",
		Arguments: map[string]any{"instance_id": "any"},
	}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return recorder.seen
}

func TestAToolCallCarryingMetaConversationIdReachesTheManagerOnItsContext(t *testing.T) {
	seen := conversationsSeenBy(t, gosdkmcp.Meta{dispatch.ConversationMetaKey: "chat-42"})
	if len(seen) != 1 || seen[0] != "chat-42" {
		t.Fatalf("the manager saw conversations %q, want exactly [chat-42]", seen)
	}
}

func TestAToolCallCarryingNoMetaReachesTheManagerWithNoConversation(t *testing.T) {
	seen := conversationsSeenBy(t, nil)
	if len(seen) != 1 || seen[0] != "" {
		t.Fatalf("the manager saw conversations %q, want exactly an empty one", seen)
	}
}
