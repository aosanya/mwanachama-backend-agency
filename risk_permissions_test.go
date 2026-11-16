package mwanachamaagency_test

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

const riskOperations = "testdata/operations/riskmanager.operations.json"

var riskReadActions = []string{
	"riskmanager.consequence.list",
	"riskmanager.control.list",
	"riskmanager.control.read",
	"riskmanager.control.reliance",
	"riskmanager.gap.list",
	"riskmanager.match.list",
	"riskmanager.mitigating_measure.list",
	"riskmanager.preventive_measure.list",
	"riskmanager.risk.bowtie",
	"riskmanager.risk.list",
	"riskmanager.risk.read",
	"riskmanager.threat.list",
}

var riskReadDesignActions = []string{
	"agency.consequence.list",
	"agency.consequence.read",
	"agency.hazard.list",
	"agency.hazard.read",
	"agency.risk.list",
	"agency.risk.read",
	"agency.risk_consequence.list",
	"agency.risk_pathway.list",
	"agency.risk_pathway.read",
	"agency.risk_threat.list",
	"agency.threat.list",
	"agency.threat.read",
	"agency.unguarded_parts.list",
}

type riskOperation struct {
	Method string `json:"method"`
	Action string `json:"action"`
}

func riskOperationsFromTestdata(t *testing.T) []riskOperation {
	t.Helper()
	raw, err := os.ReadFile(riskOperations)
	if err != nil {
		t.Fatalf("read the risk manager operations: %v", err)
	}
	var doc struct {
		Operations map[string]riskOperation `json:"operations"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	out := make([]riskOperation, 0, len(doc.Operations))
	for _, op := range doc.Operations {
		out = append(out, op)
	}
	return out
}

func agencyOperationsFromDeclaration(t *testing.T) []riskOperation {
	t.Helper()
	raw, err := os.ReadFile("agency.operations.json")
	if err != nil {
		t.Fatalf("read the agency operations: %v", err)
	}
	var doc struct {
		Operations map[string]riskOperation `json:"operations"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	out := make([]riskOperation, 0, len(doc.Operations))
	for _, op := range doc.Operations {
		out = append(out, op)
	}
	return out
}

func TestRiskReadPermissionCarriesExactlyTheSixAnswersAndTheReadsThatFindThem(t *testing.T) {
	p, ok := models.LookupPermission("risk.read")
	if !ok {
		t.Fatal("risk.read is missing from the permissions catalogue")
	}
	if p.Locked {
		t.Fatal("risk.read is locked to owners, so no agent role may hold it")
	}
	if p.Module != "riskmanager" {
		t.Errorf("risk.read module = %q, want riskmanager", p.Module)
	}
	got := append([]string(nil), p.Actions...)
	sort.Strings(got)
	want := append(append([]string(nil), riskReadActions...), riskReadDesignActions...)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("risk.read carries %v, want %v", got, want)
	}
}

func TestRiskReadPermissionCarriesOnlyActionsTheRiskManagerServesAsReads(t *testing.T) {
	ops := riskOperationsFromTestdata(t)
	method := map[string]string{}
	for _, op := range ops {
		method[op.Action] = op.Method
	}
	for _, op := range agencyOperationsFromDeclaration(t) {
		method[op.Action] = op.Method
	}
	p, _ := models.LookupPermission("risk.read")
	for _, a := range p.Actions {
		m, declared := method[a]
		if !declared {
			t.Errorf("risk.read carries %s, which neither the risk manager nor the agency declares", a)
			continue
		}
		if m != "GET" {
			t.Errorf("risk.read carries %s, which is served as %s", a, m)
		}
	}
}

func TestNoUnlockedPermissionCarriesARiskManagerWrite(t *testing.T) {
	ops := riskOperationsFromTestdata(t)
	held := heldByAnUnlockedPermission()
	for _, op := range ops {
		if op.Method == "GET" {
			continue
		}
		if permission := held[op.Action]; permission != "" {
			t.Errorf("%s is a %s on the risk manager and %s, an unlocked permission, carries it", op.Action, op.Method, permission)
		}
	}
}

func TestRiskReadIsOfferedToARoleThroughTheManager(t *testing.T) {
	mgr := newTestManager(t)
	permissions, err := mgr.ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("ListPermissions: %v", err)
	}
	for _, p := range permissions {
		if p.Code == "risk.read" {
			return
		}
	}
	t.Fatal("ListPermissions does not offer risk.read to a role")
}

func TestOnlyTheLabsHubStewardHoldsRiskRead(t *testing.T) {
	cases := []struct {
		file, role string
		holds      bool
	}{
		{"testdata/hub-demo/siaya-lab/agency.json", "Hub Steward", true},
		{"testdata/hub-demo/boda-boda-riders/agency.json", "Hub Steward", false},
		{"testdata/hub-demo/farmers/agency.json", "Farmers' agent", false},
	}
	for _, c := range cases {
		t.Run(c.role+" in "+c.file, func(t *testing.T) {
			doc := readDocument(t, c.file)
			mgr := newTestManager(t)
			ctx := context.Background()
			out, err := mgr.ImportDocument(ctx, "hub demo", doc)
			if err != nil {
				t.Fatalf("ImportDocument: %v", err)
			}
			roles, err := mgr.ListRoles(ctx, out.Draft.ID, mwanachamaagency.RoleFilter{Name: c.role})
			if err != nil {
				t.Fatalf("ListRoles: %v", err)
			}
			held := false
			for _, r := range roles {
				if r.Name != c.role {
					continue
				}
				for _, p := range r.Permissions {
					if p == "risk.read" {
						held = true
					}
				}
			}
			if held != c.holds {
				t.Errorf("%s holds risk.read = %v, want %v", c.role, held, c.holds)
			}
		})
	}
}
