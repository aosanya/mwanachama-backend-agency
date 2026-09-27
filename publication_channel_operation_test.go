package mwanachamaagency

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
)

func TestAG48_OpenBug_ProductionChannelReadIsDeclared(t *testing.T) {
	s, err := dispatch.Parse(Operations())
	if err != nil {
		t.Fatalf("dispatch.Parse: %v", err)
	}
	for _, op := range s.Operations {
		if op.Call == "GetPublicationForChannel" {
			return
		}
	}
	t.Fatalf("AG48: agency.operations.json declares no operation calling GetPublicationForChannel, so neither routes.Routes nor mcp.RegisterTools ever reach it — a production consumer has no way through the generated REST or MCP surface to fetch \"the publication currently serving a channel\" (this method's own doc comment on AgencyManager); it can only call GET /publications?channel=<name> and take element 0 of the list itself, which loses the exactly-one guarantee GetPublicationForChannel gives")
}
