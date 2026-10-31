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
	"riskmanager.mitigating_measure.list",
	"riskmanager.preventive_measure.list",
	"riskmanager.risk.bowtie",
	"riskmanager.risk.list",
	"riskmanager.risk.read",
	"riskmanager.threat.list",
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

func TestRiskReadPowerCarriesExactlyTheSixAnswersAndTheReadsThatFindThem(t *testing.T) {
	p, ok := models.LookupPower("risk.read")
	if !ok {
		t.Fatal("risk.read is missing from the powers catalogue")
	}
	if p.Locked {
		t.Fatal("risk.read is locked to owners, so no agent role may hold it")
	}
	if p.Module != "riskmanager" {
		t.Errorf("risk.read module = %q, want riskmanager", p.Module)
	}
	got := append([]string(nil), p.Actions...)
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(riskReadActions, ",") {
		t.Errorf("risk.read carries %v, want %v", got, riskReadActions)
	}
}

func TestRiskReadPowerCarriesOnlyActionsTheRiskManagerServesAsReads(t *testing.T) {
	ops := riskOperationsFromTestdata(t)
	method := map[string]string{}
	for _, op := range ops {
		method[op.Action] = op.Method
	}
	p, _ := models.LookupPower("risk.read")
	for _, a := range p.Actions {
		m, declared := method[a]
		if !declared {
			t.Errorf("risk.read carries %s, which the risk manager does not declare", a)
			continue
		}
		if m != "GET" {
			t.Errorf("risk.read carries %s, which the risk manager serves as %s", a, m)
		}
	}
}

func TestNoUnlockedPowerCarriesARiskManagerWrite(t *testing.T) {
	ops := riskOperationsFromTestdata(t)
	held := heldByAnUnlockedPower()
	for _, op := range ops {
		if op.Method == "GET" {
			continue
		}
		if power := held[op.Action]; power != "" {
			t.Errorf("%s is a %s on the risk manager and %s, an unlocked power, carries it", op.Action, op.Method, power)
		}
	}
}

func TestRiskReadIsOfferedToARoleThroughTheManager(t *testing.T) {
	mgr := newTestManager(t)
	powers, err := mgr.ListPowers(context.Background())
	if err != nil {
		t.Fatalf("ListPowers: %v", err)
	}
	for _, p := range powers {
		if p.Code == "risk.read" {
			return
		}
	}
	t.Fatal("ListPowers does not offer risk.read to a role")
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
				for _, p := range r.Powers {
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
