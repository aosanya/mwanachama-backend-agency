package mwanachamaagency_test

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const (
	rolePermissionsDocEnv  = "ROLE_PERMISSIONS_DOC"
	rolePermissionsDocPath = "documentation/2. design/role-permissions.md"
)

var (
	permissionTableHeads = map[string]bool{"Code": true, "Permission": true}
	permissionRowCode    = regexp.MustCompile("^\\|\\s*`([a-z][a-z0-9_.]*)`")
	tableFirstCell       = regexp.MustCompile("^\\|\\s*([^|]*?)\\s*\\|")
)

func cataloguePermissionCodes() []string {
	var codes []string
	for _, p := range models.PermissionCatalog() {
		codes = append(codes, p.Code)
	}
	sort.Strings(codes)
	return codes
}

func permissionRowsInDoc(markdown string) []string {
	seen := map[string]bool{}
	inPermissionTable := false
	previousWasTableLine := false
	for _, line := range strings.Split(markdown, "\n") {
		isTableLine := strings.HasPrefix(line, "|")
		switch {
		case isTableLine && !previousWasTableLine:
			head := tableFirstCell.FindStringSubmatch(line)
			inPermissionTable = head != nil && permissionTableHeads[head[1]]
		case isTableLine && inPermissionTable:
			if match := permissionRowCode.FindStringSubmatch(line); match != nil {
				seen[match[1]] = true
			}
		}
		previousWasTableLine = isTableLine
	}
	var codes []string
	for code := range seen {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

func codesNotIn(codes, others []string) []string {
	known := map[string]bool{}
	for _, other := range others {
		known[other] = true
	}
	var out []string
	for _, code := range codes {
		if !known[code] {
			out = append(out, code)
		}
	}
	return out
}

func permissionDocDrift(catalogue, documented []string) (undocumented, unknown []string) {
	return codesNotIn(catalogue, documented), codesNotIn(documented, catalogue)
}

func rolePermissionsDocText(t *testing.T) string {
	t.Helper()
	path := os.Getenv(rolePermissionsDocEnv)
	if path == "" {
		path = rolePermissionsDocPath
	}
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(text)
}

func TestRolePermissionsDocHasARowForEveryCataloguePermissionAndOnlyThose(t *testing.T) {
	undocumented, unknown := permissionDocDrift(cataloguePermissionCodes(), permissionRowsInDoc(rolePermissionsDocText(t)))
	if len(undocumented) > 0 {
		t.Errorf("catalogue permissions with no row in a Code or Permission table of role-permissions.md: %v", undocumented)
	}
	if len(unknown) > 0 {
		t.Errorf("rows in a Code or Permission table of role-permissions.md naming permissions that are not in the catalogue: %v", unknown)
	}
}

func TestPermissionDocDriftNamesBothDirections(t *testing.T) {
	catalogue := []string{"chat.post", "form.send", "report.publish"}
	cases := map[string]struct {
		documented                    []string
		wantUndocumented, wantUnknown []string
	}{
		"in step":       {documented: catalogue},
		"a row removed": {documented: []string{"chat.post", "report.publish"}, wantUndocumented: []string{"form.send"}},
		"a bogus row":   {documented: append([]string{"order.pool"}, catalogue...), wantUnknown: []string{"order.pool"}},
		"both": {
			documented:       []string{"chat.post", "report.publish", "order.pool"},
			wantUndocumented: []string{"form.send"},
			wantUnknown:      []string{"order.pool"},
		},
	}
	for name, c := range cases {
		undocumented, unknown := permissionDocDrift(catalogue, c.documented)
		if !reflect.DeepEqual(undocumented, c.wantUndocumented) || !reflect.DeepEqual(unknown, c.wantUnknown) {
			t.Errorf("%s: undocumented %v, unknown %v", name, undocumented, unknown)
		}
	}
}

func TestPermissionRowsInDocReadsOnlyTablesHeadedCodeOrPermission(t *testing.T) {
	markdown := strings.Join([]string{
		"| `outside_before` | a |",
		"",
		"| Code | Permission | Locked |",
		"| --- | --- | --- |",
		"| `report.publish` | Publish a report | no |",
		"| `agent.manage` | Add or remove agents | yes |",
		"prose mentioning `not.a_row` is not a row",
		"",
		"| Permission | Actions |",
		"| --- | --- |",
		"| `member.invite` Invite | `actor.invitation.create` |",
		"| `report.publish` | again |",
		"",
		"| Question | Action |",
		"|---|---|",
		"| `riskmanager.threat.list` | What are the threats |",
		"",
		"| Object | Actions |",
		"| --- | --- |",
		"| node | `digitaltwin.node.create` |",
		"",
		"| Code | Name | Actions |",
		"|---|---|",
		"| `controls.edit` | Edit controls | `agency.control.create` |",
	}, "\n")
	want := []string{"agent.manage", "controls.edit", "member.invite", "report.publish"}
	if got := permissionRowsInDoc(markdown); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
}
