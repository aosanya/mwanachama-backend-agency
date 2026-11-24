package mwanachamaagency_test

import (
	"context"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
)

var hubDemoTemplates = []struct {
	file      string
	agentRole string
}{
	{"testdata/hub-demo/farmers/agency.json", "Farmers' agent"},
	{"testdata/hub-demo/farmers/agency.json", "Suppliers' agent"},
	{"testdata/hub-demo/siaya-lab/agency.json", "Hub Steward"},
	{"testdata/hub-demo/boda-boda-riders/agency.json", "Hub Steward"},
}

func TestEveryHubDemoTemplateImportsAndPublishesWithTheAgentRoleHoldingAIUse(t *testing.T) {
	for _, tpl := range hubDemoTemplates {
		t.Run(tpl.agentRole+" in "+tpl.file, func(t *testing.T) {
			doc := readDocument(t, tpl.file)
			mgr := newTestManager(t)
			ctx := context.Background()
			out, err := mgr.ImportDocument(ctx, "hub demo", doc)
			if err != nil {
				t.Fatalf("ImportDocument: %v", err)
			}
			roles, err := mgr.ListRoles(ctx, out.Draft.ID, mwanachamaagency.RoleFilter{Name: tpl.agentRole})
			if err != nil {
				t.Fatalf("ListRoles: %v", err)
			}
			var agent []string
			for _, r := range roles {
				if r.Name == tpl.agentRole {
					agent = r.Permissions
				}
			}
			have := map[string]bool{}
			for _, p := range agent {
				have[p] = true
			}
			for _, want := range []string{"ai.use", "chat.post", "chat.thread", "group.read"} {
				if !have[want] {
					t.Errorf("%s imported without %s: %v", tpl.agentRole, want, agent)
				}
			}
			records, err := mgr.ListRecords(ctx, out.Draft.ID, mwanachamaagency.RecordFilter{})
			if err != nil {
				t.Fatalf("ListRecords: %v", err)
			}
			for _, record := range records {
				mayAdd(t, mgr, record)
			}
			if _, err := mgr.PublishAgency(ctx, out.Draft.ID, "demo-1", "production"); err != nil {
				t.Fatalf("PublishAgency refused the template: %v", err)
			}
		})
	}
}
