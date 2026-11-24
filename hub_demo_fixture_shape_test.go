package mwanachamaagency_test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	mwanachamaagency "github.com/aosanya/mwanachama-backend-agency"
	"github.com/aosanya/mwanachama-backend-agency/models"
)

var hubDemoFiles = []string{
	"testdata/hub-demo/farmers/agency.json",
	"testdata/hub-demo/boda-boda-riders/agency.json",
	"testdata/hub-demo/siaya-lab/agency.json",
}

var removedRecordFields = []string{"role_id", "edit_role_id", "delete_role_id", "viewer_role_id"}

var removedDocumentKeys = []string{"role_assignments", "groups", "tags"}

var recordHolders = map[string]string{
	"testdata/hub-demo/farmers/agency.json":          "Group coordinator",
	"testdata/hub-demo/boda-boda-riders/agency.json": "Rider",
}

type rawHubDemo struct {
	Records []map[string]any `json:"records"`
	Roles   []map[string]any `json:"roles"`
}

func readRawHubDemo(t *testing.T, path string) (rawHubDemo, map[string]json.RawMessage) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc rawHubDemo
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return doc, keys
}

func TestHubDemoFilesCarryNoRemovedField(t *testing.T) {
	for _, path := range hubDemoFiles {
		t.Run(path, func(t *testing.T) {
			doc, keys := readRawHubDemo(t, path)
			for _, key := range removedDocumentKeys {
				if _, found := keys[key]; found {
					t.Errorf("%s carries the key %q", path, key)
				}
			}
			for _, record := range doc.Records {
				for _, field := range removedRecordFields {
					if _, found := record[field]; found {
						t.Errorf("record %v carries %q", record["code"], field)
					}
				}
			}
			for _, role := range doc.Roles {
				if _, found := role["powers"]; found {
					t.Errorf("role %v carries powers", role["name"])
				}
			}
		})
	}
}

func TestHubDemoRecordsCarryReadableCodes(t *testing.T) {
	for path := range recordHolders {
		t.Run(path, func(t *testing.T) {
			doc, _ := readRawHubDemo(t, path)
			for _, record := range doc.Records {
				name, _ := record["name"].(string)
				if want := mwanachamaagency.RecordCodeOf(name); record["code"] != want {
					t.Errorf("record %q has the code %v, want %q", name, record["code"], want)
				}
			}
		})
	}
}

func TestHubDemoRecordHolderCarriesTheRecordsPermissions(t *testing.T) {
	for path, holder := range recordHolders {
		t.Run(path, func(t *testing.T) {
			doc, _ := readRawHubDemo(t, path)
			if len(doc.Records) == 0 {
				t.Fatalf("%s declares no record", path)
			}
			carried := permissionsOfRole(t, doc, holder)
			for _, record := range doc.Records {
				code, _ := record["code"].(string)
				for _, want := range []string{
					mwanachamaagency.RecordPermissionCode(code, models.RowActionAdd, mwanachamaagency.RecordFieldStar),
					mwanachamaagency.RecordPermissionCode(code, models.RowActionEdit, mwanachamaagency.RecordFieldStar),
					mwanachamaagency.RecordPermissionCode(code, mwanachamaagency.RecordViewAction, mwanachamaagency.RecordFieldStar),
					mwanachamaagency.RecordPermissionCode(code, models.RowActionDelete),
				} {
					if !slices.Contains(carried, want) {
						t.Errorf("%s does not carry %s", holder, want)
					}
				}
			}
		})
	}
}

func permissionsOfRole(t *testing.T, doc rawHubDemo, name string) []string {
	t.Helper()
	for _, role := range doc.Roles {
		if role["name"] != name {
			continue
		}
		var carried []string
		list, _ := role["permissions"].([]any)
		for _, p := range list {
			if code, ok := p.(string); ok {
				carried = append(carried, code)
			}
		}
		return carried
	}
	t.Fatalf("no role named %q", name)
	return nil
}
