package mcp

import (
	"context"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

func publishedToolNames(t *testing.T, register func(*mcp.Server)) []string {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "shadowing", Version: "0.0.0"}, nil)
	register(server)
	serverSide, clientSide := mcp.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := server.Connect(ctx, serverSide, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "shadowing-client", Version: "0.0.0"}, nil).Connect(ctx, clientSide, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	listed, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	names := make([]string, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}

func TestNoHandWrittenToolSilentlyReplacesADeclaredOneOfTheSameName(t *testing.T) {
	resolve := func(context.Context, string) (mwanachamaagency.AgencyManager, error) { return nil, nil }

	declared := map[string]bool{}
	for _, tool := range declaredTools(t) {
		declared[tool.Name] = true
	}
	if len(declared) == 0 {
		t.Fatal("precondition: agency.operations.json declares no tools")
	}

	handWritten := publishedToolNames(t, func(s *mcp.Server) {
		registerModuleObjectivesTool(s, resolve)
		registerCompositeTools(s, resolve)
		registerCatalogTools(s)
		registerDashboardUI(s)
	})
	if len(handWritten) == 0 {
		t.Fatal("precondition: no hand-written tool was registered")
	}
	for _, name := range handWritten {
		if declared[name] {
			t.Errorf("hand-written tool %q shares its name with a declared operation's tool; the go-sdk keeps whichever registers last, so one of the two is silently unreachable", name)
		}
	}

	everything := publishedToolNames(t, func(s *mcp.Server) { RegisterTools(s, resolve) })
	if want := len(declared) + len(handWritten); len(everything) != want {
		t.Errorf("RegisterTools published %d tools, want %d (%d declared + %d hand-written): a name was registered twice and the earlier tool was dropped", len(everything), want, len(declared), len(handWritten))
	}
}
